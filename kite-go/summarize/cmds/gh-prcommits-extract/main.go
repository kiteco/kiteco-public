package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/summarize/data"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/githubcorpus"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	start := time.Now()
	args := struct {
		InDir string
		Out   string
	}{}
	arg.MustParse(&args)

	corpus, err := githubcorpus.NewPullRequestCorpus(args.InDir)
	fail(err)

	f, err := fileutil.NewBufferedWriter(args.Out)
	fail(err)
	defer f.Close()

	js := json.NewEncoder(f)

	var prCount, commitCount int
	err = corpus.Scan(func(bundle githubcorpus.PullRequestBundle, contents githubcorpus.Contents) bool {
		pr := data.GHPullRequest{
			RepoOwner: bundle.PullRequest.GetBase().GetUser().GetLogin(),
			RepoName:  bundle.PullRequest.GetBase().GetRepo().GetName(),
			Number:    bundle.PullRequest.GetNumber(),
			Title:     bundle.PullRequest.GetTitle(),
			Body:      bundle.PullRequest.GetBody(),
		}

		for _, c := range bundle.Commits {
			cc := data.GHCommit{
				Message: c.Commit.GetMessage(),
			}

			for _, f := range c.Files {
				base, err := githubcorpus.GetSourceState(c, f.GetFilename(), contents)
				if err != nil {
					continue
				}

				cc.Files = append(cc.Files, data.GHCommitFile{
					Patch:       f.GetPatch(),
					BaseContent: string(base),
				})
			}
			if len(cc.Files) == 0 {
				continue
			}

			pr.Commits = append(pr.Commits, cc)
		}

		if len(pr.Commits) == 0 {
			return true
		}
		fail(js.Encode(pr))

		prCount++
		commitCount += len(pr.Commits)

		return true
	})
	fail(err)

	fmt.Printf("Done, took %v to encode %d PRs containing %d commits\n",
		time.Since(start), prCount, commitCount)
}
