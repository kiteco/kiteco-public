package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/githubcorpus"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type ownerName struct {
	Owner string
	Name  string
}

func readRepos(reposFile string) []ownerName {
	f, err := fileutil.NewReader(reposFile)
	fail(err)
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	fail(err)

	var repos []ownerName
	for _, line := range strings.Split(string(buf), "\n") {
		parts := strings.Split(line, "/")
		if len(parts) != 2 {
			continue
		}
		owner, name := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if owner == "" || name == "" {
			continue
		}
		repos = append(repos, ownerName{Owner: owner, Name: name})
	}

	if len(repos) == 0 {
		fail(errors.New("no repos found"))
	}
	return repos
}

func watchStats(start time.Time, interval time.Duration) {
	for range time.Tick(interval) {
		var buf bytes.Buffer
		fmt.Fprintln(&buf, "=== Stats ===")
		fmt.Fprintf(&buf, "Time elapsed: %v\n", time.Since(start))
		fmt.Fprintf(&buf, "GetContentSuccessRate: %v\n",
			githubcorpus.GetContentSuccessRate.Value())
		fmt.Fprintf(&buf, "GetCommitSuccessRate: %v\n",
			githubcorpus.GetCommitSuccessRate.Value())
		fmt.Fprintf(&buf, "MergeCommitRatio: %v\n",
			githubcorpus.MergeCommitRatio.Value())
		fmt.Fprintf(&buf, "FullCommitSuccessRate: %v\n",
			githubcorpus.FullCommitSuccessRate.Value())

		fmt.Println(buf.String())
	}
}

func main() {
	start := time.Now()
	args := struct {
		MaxFilesPerPR int
		ReposFile     string
		OutDir        string
		PRState       string
	}{
		ReposFile:     "./repos",
		PRState:       "closed",
		MaxFilesPerPR: 100,
		OutDir:        "./data1",
	}
	arg.MustParse(&args)

	corpus, err := githubcorpus.NewAPIPullRequestCorpus(githubcorpus.APIScanOptions{
		State:              args.PRState,
		MaxPages:           40,
		PerPage:            100,
		IncludeCommits:     true,
		IncludeCommitFiles: true,
	})
	fail(err)

	fail(os.MkdirAll(args.OutDir, os.ModePerm))

	go watchStats(start, 10*time.Minute)

	var repoCount, pullCount int
	for _, repo := range readRepos(args.ReposFile) {
		func() {
			fn := fileutil.Join(args.OutDir, githubcorpus.PRCorpusFilename(repo.Owner, repo.Name))

			f, err := fileutil.NewBufferedWriter(fn)
			fail(err)
			defer f.Close()

			gz := gzip.NewWriter(f)
			defer gz.Close()

			js := json.NewEncoder(gz)

			err = corpus.ScanRepo(repo.Owner, repo.Name, func(bundle githubcorpus.PullRequestBundle, contents githubcorpus.Contents) bool {
				owner := bundle.PullRequest.GetBase().GetUser().GetLogin()
				name := bundle.PullRequest.GetBase().GetRepo().GetName()
				mergedAt := bundle.PullRequest.GetMergedAt()

				if len(bundle.CommitFiles) == 0 || (args.MaxFilesPerPR > 0 && len(bundle.CommitFiles) > args.MaxFilesPerPR) {
					fmt.Printf("skipping PR %s/%s#%d because it has an unsupported number of files (%d) \n",
						repo.Owner, repo.Name, bundle.PullRequest.GetNumber(), len(bundle.CommitFiles))
					return true
				}

				if mergedAt == (time.Time{}) {
					fmt.Printf("skipping PR %s/%s#%d, closed but not-merged\n", owner, name, bundle.PullRequest.GetNumber())
					return true
				}

				dataFiles := corpus.FetchDataFiles(bundle.PullRequest, bundle.CommitFiles)

				commits, dataFiles := corpus.FetchCommits(bundle.PullRequest, bundle.Commits, dataFiles)

				bundle.DataFiles = dataFiles
				bundle.Commits = commits

				fail(js.Encode(bundle))
				pullCount++
				return true
			})
			fail(err)
			repoCount++
		}()
	}

	fmt.Printf("extracted %d pull requests from %d repos in %v\n", pullCount, repoCount, time.Since(start))
}
