package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"golang.org/x/oauth2"

	"github.com/alexflint/go-arg"
	"github.com/google/go-github/github"
)

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	args := struct {
		Repo         string
		Owner        string
		PullsPath    string
		RepoCloneDir string
	}{
		Owner:        "sergi",
		Repo:         "go-diff",
		PullsPath:    "pulls.json",
		RepoCloneDir: "tmp",
	}

	arg.MustParse(&args)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var pullNumbers []int
	data, err := ioutil.ReadFile(args.PullsPath)
	fail(err)
	err = json.Unmarshal(data, &pullNumbers)
	fail(err)

	repo, _, err := client.Repositories.Get(ctx, args.Owner, args.Repo)
	fail(err)
	url := repo.GetCloneURL()

	var sha string
	var earliest time.Time
	for _, pullNumber := range pullNumbers {
		// Hack for rate limiting
		time.Sleep(720 * time.Millisecond)
		pull, _, err := client.PullRequests.Get(ctx, args.Owner, args.Repo, pullNumber)
		fail(err)
		mergedAt := pull.GetMergedAt()
		current := pull.GetBase().GetSHA()
		if mergedAt == (time.Time{}) {
			continue
		}
		if earliest == (time.Time{}) || mergedAt.Before(earliest) {
			earliest = mergedAt
			sha = current
		}
	}

	// Clone the repo and check out to the correct sha
	cmd := exec.Command("git", "clone", url)
	cmd.Dir = args.RepoCloneDir
	fail(cmd.Run())

	cmd = exec.Command("git", "checkout", sha)
	cmd.Dir = path.Join(args.RepoCloneDir, args.Repo)
	fail(cmd.Run())

	fmt.Printf("Cloned repo into %s with sha %s\n", path.Join(args.RepoCloneDir, args.Repo), sha)
}
