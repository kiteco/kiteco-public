package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/githubcorpus"
	"github.com/mholt/archiver"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type ownerName struct {
	Owner, Name string
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

func main() {
	start := time.Now()
	args := struct {
		ReposFile string
		OutDir    string
	}{
		ReposFile: "./repos",
		OutDir:    "./data",
	}
	arg.MustParse(&args)

	fail(os.MkdirAll(args.OutDir, os.ModePerm))

	repos := readRepos(args.ReposFile)
	for _, repo := range repos {
		func() {
			dirName := githubcorpus.RepoCorpusFilename(repo.Owner, repo.Name, 0)
			finalPath := filepath.Join(args.OutDir, dirName)
			if _, err := os.Stat(finalPath); err == nil {
				fmt.Printf("skipping repo %s/%s because it already exists\n", repo.Owner, repo.Name)
				return
			}

			tmpDir, err := ioutil.TempDir("", dirName)
			fail(err)
			defer os.RemoveAll(tmpDir)

			tmpPath := filepath.Join(tmpDir, repo.Name)
			fail(os.MkdirAll(tmpPath, os.ModePerm))

			_, err = git.PlainClone(tmpPath, false, &git.CloneOptions{
				Auth: &http.BasicAuth{
					Username: "abc123", // yes, this can be anything except an empty string
					Password: envutil.MustGetenv("GITHUB_AUTH_TOKEN"),
				},
				URL:      fmt.Sprintf("https://github.com/%s/%s", repo.Owner, repo.Name),
				Progress: os.Stdout,
			})
			fail(err)

			// remove symlinked files since this breaks archiving
			remove := make(map[string]bool)
			fail(filepath.Walk(tmpPath, func(path string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return err
				}

				if info.Mode()&os.ModeSymlink != 0 {
					remove[path] = true
					return err
				}
				return err
			}))

			for path := range remove {
				fail(os.Remove(path))
			}

			fail(archiver.NewTarGz().Archive([]string{tmpPath}, finalPath))
		}()
	}
	fmt.Printf("Done, took %v to clone %d repos to %s\n",
		time.Since(start), len(repos), args.OutDir)
}
