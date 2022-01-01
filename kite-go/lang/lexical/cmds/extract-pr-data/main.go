package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/githubcorpus"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/githubdata"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		Owner              string
		Name               string
		MinAdds            int
		MaxAdds            int
		Language           string
		SitesSavePath      string
		Live               bool
		MinNumTokensWindow int
	}{
		Owner:              "kiteco",
		Name:               "kiteco",
		MinAdds:            1,
		MaxAdds:            5,
		Language:           "go",
		SitesSavePath:      "sites.json",
		MinNumTokensWindow: 5,
	}

	arg.MustParse(&args)

	opts := githubdata.Options{
		AllowedStatus: map[string]bool{"modified": true},
	}

	extractor, err := githubdata.NewExtractor(opts)
	fail(err)

	var corpus githubcorpus.Corpus
	repoLang := lang.FromName(args.Language)
	langLexer, err := lexicalv0.NewLexer(repoLang)
	fail(err)

	if args.Live {
		corpus, err = githubcorpus.NewAPIPullRequestCorpus(githubcorpus.APIScanOptions{
			MaxPages: 5,
			PerPage:  100,
		})
	} else {
		corpus, err = githubcorpus.NewPullRequestCorpus("s3://kite-github-pullrequests/v1/")
	}
	fail(err)

	var earliestSHA string
	var earliest time.Time
	var sites []githubdata.PredictionSiteWithMetrics

	err = corpus.ScanRepo(args.Owner, args.Name, func(bundle githubcorpus.PullRequestBundle, contents githubcorpus.Contents) bool {
		owner := bundle.PullRequest.GetBase().GetUser().GetLogin()
		name := bundle.PullRequest.GetBase().GetRepo().GetName()
		mergedAt := bundle.PullRequest.GetMergedAt()

		if mergedAt == (time.Time{}) {
			fmt.Printf("skipping PR %s/%s#%d, closed but not-merged\n", owner, name, bundle.PullRequest.GetNumber())
			return true
		}

		if len(bundle.DataFiles) == 0 {
			return true
		}

		if earliest == (time.Time{}) || mergedAt.Before(earliest) {
			earliestSHA = bundle.PullRequest.GetBase().GetSHA()
			earliest = mergedAt
		}

		for _, file := range bundle.CommitFiles {
			if lang.FromFilename(file.GetFilename()) != repoLang {
				continue
			}

			if strings.HasSuffix(file.GetFilename(), "bindata.go") {
				continue
			}

			newSites, err := extractor.ExtractPredictionSites(bundle.PullRequest, file, contents)
			if err != nil {
				continue
			}

			var added int
			for _, s := range newSites {
				swm, err := githubdata.NewPredictionSiteWithMetrics(s, langLexer)
				if err != nil {
					continue
				}
				if swm.NumTokensAfter < args.MinNumTokensWindow {
					continue
				}

				nLines := strings.Count(swm.DstWindow, "\n")
				if nLines < args.MinAdds || (args.MaxAdds > 0 && nLines > args.MaxAdds) {
					continue
				}

				added++
				sites = append(sites, swm)
			}

			/*
				fmt.Printf("extracting data from PR %s/%s#%d, (%d/%d) file, generated %d sites.\n",
					owner, name, bundle.PullRequest.GetNumber(), i, len(bundle.CommitFiles), added)
			*/
		}
		return true
	})
	fail(err)

	fmt.Printf("extracted %d sites in total.\n", len(sites))
	fmt.Printf("earliest sha is: %s", earliestSHA)

	sitesJSON, err := json.Marshal(sites)
	fail(err)

	err = ioutil.WriteFile(args.SitesSavePath, sitesJSON, os.ModePerm)
	fail(err)
}
