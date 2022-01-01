package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type quipSuffix string

type issueNumber string

func main() {
	args := struct {
		Retrieved string
		Root      string
		Quip      string
		Issues    string
		MaxIssues int
	}{
		MaxIssues: -1,
	}
	arg.MustParse(&args)

	retrieved, err := localpath.NewAbsolute(args.Retrieved)
	if err != nil {
		log.Fatal(err)
	}
	root, err := localpath.NewAbsolute(args.Root)
	if err != nil {
		log.Fatal(err)
	}
	quip, err := localpath.NewAbsolute(args.Quip)
	if err != nil {
		log.Fatal(err)
	}
	issues, err := localpath.NewAbsolute(args.Issues)
	if err != nil {
		log.Fatal(err)
	}

	c, err := readCorpus(quip, issues)
	if err != nil {
		log.Fatal(err)
	}
	r, err := buildRecommender(root)
	if err != nil {
		log.Fatal(err)
	}
	recs, err := collect(r, c, args.MaxIssues)
	if err != nil {
		log.Fatal(err)
	}
	err = write(recs, retrieved)
	if err != nil {
		log.Fatal(err)
	}
}

func collect(r recommend.Recommender, c corpus, maxIssues int) (map[quipSuffix][]issueNumber, error) {
	data := make(map[quipSuffix][]issueNumber)
	for suffix := range c.quip {
		abs := filepath.Join(string(c.quipDir), string(suffix)+".py")
		req := recommend.Request{
			MaxFileRecs: -1,
			Location: recommend.Location{
				CurrentPath: string(abs),
			},
		}
		recs, err := r.Recommend(kitectx.Background(), req)
		if err != nil {
			return nil, err
		}
		var issues []issueNumber
		for _, rec := range recs {
			internalPath, err := localpath.NewAbsolute(rec.Path)
			if err != nil {
				return nil, err
			}
			if internalPath.Dir() != c.issuesDir {
				continue
			}
			issue, ok := getIssueNumber(internalPath)
			if !ok {
				continue
			}
			issues = append(issues, issue)
			if len(issues) == maxIssues {
				break
			}
		}
		data[suffix] = issues
	}
	return data, nil
}

func getQuipSuffix(path localpath.Absolute) (quipSuffix, bool) {
	id, ok := getID(path)
	return quipSuffix(id), ok
}

func getIssueNumber(path localpath.Absolute) (issueNumber, bool) {
	id, ok := getID(path)
	return issueNumber(id), ok
}

func getID(path localpath.Absolute) (string, bool) {
	base := filepath.Base(string(path))
	if len(base) < 3 {
		return "", false
	}
	if base[len(base)-3:] != ".py" {
		return "", false
	}
	return base[:len(base)-3], true
}

func buildRecommender(root localpath.Absolute) (recommend.Recommender, error) {
	var (
		ignoreOpts = ignore.Options{
			Root: root,
		}
		storageOpts = git.StorageOptions{
			UseDisk: true,
			Path:    string(root.Join("kite-go", "navigation", "offline", "git-cache.json")),
		}
		recOpts = recommend.Options{
			Root:        root,
			MaxFileSize: 1e6,
			MaxFiles:    1e5,
			UseCommits:  false,
		}
	)
	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		return nil, err
	}
	s, err := git.NewStorage(storageOpts)
	if err != nil {
		return nil, err
	}
	return recommend.NewRecommender(kitectx.Background(), recOpts, ignorer, s)
}

type corpus struct {
	quip      map[quipSuffix]bool
	quipDir   localpath.Absolute
	issuesDir localpath.Absolute
}

func readCorpus(quipDir, issuesDir localpath.Absolute) (corpus, error) {
	quip := make(map[quipSuffix]bool)
	quipDirnames, err := quipDir.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}
	for _, rel := range quipDirnames {
		abs := quipDir.Join(rel)
		suffix, ok := getQuipSuffix(abs)
		if !ok {
			continue
		}
		quip[suffix] = true
	}
	return corpus{
		quip:      quip,
		quipDir:   quipDir,
		issuesDir: issuesDir,
	}, nil
}

func readRelevant(relevantPath localpath.Absolute) (map[git.File][]quipSuffix, error) {
	file, err := relevantPath.Open()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var triples map[git.File]map[quipSuffix][]string
	err = json.Unmarshal(data, &triples)
	if err != nil {
		return nil, err
	}
	relevant := make(map[git.File][]quipSuffix)
	for issue, quips := range triples {
		for quip := range quips {
			relevant[issue] = append(relevant[issue], quip)
		}
	}
	return relevant, nil
}

func write(recs map[quipSuffix][]issueNumber, retrievedPath localpath.Absolute) error {
	data, err := json.MarshalIndent(recs, "", "")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(string(retrievedPath), data, 0600)
}
