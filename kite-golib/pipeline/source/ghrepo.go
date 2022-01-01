package source

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

// GithubRepo ...
type GithubRepo struct {
	path     string
	branch   string
	maxFile  int
	delegate *Dataset
}

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

// NewGitHubRepo returns a source that produce one sample per file in this repo
// if localFolder is an empty string, an temp folder will be used instead
// if localFolder is the path to a non empty folder, the repo is not cloned and instead the content of the folder is used for the local source
// That allows to not clone the repo at each execution
// The function return the new source and a cleaning function that can be executed at the end of the pipeline if you want to remove the folder
func NewGitHubRepo(repoPath, branch, localFolder, fileExtension string, rand *rand.Rand, numgo, maxFile int, logger io.Writer) (*GithubRepo, func()) {
	if localFolder == "" {
		dir, err := ioutil.TempDir("", "gh_source_")
		if err != nil {
			log.Fatal(err)
		}
		localFolder = dir
	}
	os.MkdirAll(localFolder, 0777)
	cleanFunc := func() {}
	isEmp, err := isEmpty(localFolder)
	if err != nil {
		panic(err)
	}
	if !isEmp {
		fmt.Printf("INFO: target folder (%s) is not empty, skipping repo cloning\n", localFolder)
	} else {
		cmdClone := exec.Command("git", "clone", repoPath, ".")
		cmdClone.Dir = localFolder
		err := cmdClone.Run()
		if err != nil {
			panic(err)
		}
		if branch != "" && branch != "master" {
			cmdCheckout := exec.Command("git", "checkout", branch)
			err = cmdCheckout.Run()
			if err != nil {
				panic(err)
			}
		}
		fmt.Println("Repo ", repoPath, " cloned in ", localFolder)
		cleanFunc = func() {
			err := os.RemoveAll(localFolder)
			if err != nil {
				fmt.Println("Error while cleaning the folder : ", err)
			}
		}

	}

	fileList, err := GetFilelist(localFolder, NewFileExtensionPredicate(fileExtension), true)
	if err != nil {
		panic(err)
	}
	if maxFile > 0 && len(fileList) > maxFile {
		rand.Shuffle(len(fileList), func(i, j int) {
			fileList[i], fileList[j] = fileList[j], fileList[i]
		})
		fileList = fileList[:maxFile]
	}

	delegate := NewLocalFiles("gh_source_delegate", numgo, fileList, logger)

	return &GithubRepo{
		path:     repoPath,
		branch:   branch,
		maxFile:  maxFile,
		delegate: delegate,
	}, cleanFunc
}

// SourceOut ...
func (ghr *GithubRepo) SourceOut() pipeline.Record {
	return ghr.delegate.SourceOut()
}

// Name implements Feed
func (ghr *GithubRepo) Name() string {
	n := ghr.path[strings.LastIndex(ghr.path, ":")+1:]
	return fmt.Sprintf("GH_repo_%s", n)
}

// ForShard ...
func (ghr *GithubRepo) ForShard(shard int, total int) (pipeline.Source, error) {
	if total > 1 {
		return nil, errors.Errorf("func source only works in non distributed environment (e.g total shards = 1, got %d)", total)
	}
	return ghr, nil
}
