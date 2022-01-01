package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func main() {
	args := struct {
		Retrieved string
		Relevant  string
		Index     string
	}{}
	arg.MustParse(&args)
	retrieved, err := localpath.NewAbsolute(args.Retrieved)
	if err != nil {
		log.Fatal(err)
	}
	relevant, err := localpath.NewAbsolute(args.Relevant)
	if err != nil {
		log.Fatal(err)
	}
	index, err := localpath.NewAbsolute(args.Index)
	if err != nil {
		log.Fatal(err)
	}

	kiteco := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco")
	root, err := localpath.NewAbsolute(kiteco)
	if err != nil {
		log.Fatal(err)
	}
	c, err := readCorpus(root, relevant, index)
	if err != nil {
		log.Fatal(err)
	}
	r, err := buildRecommender(root)
	if err != nil {
		log.Fatal(err)
	}
	recs, err := collect(r, c)
	if err != nil {
		log.Fatal(err)
	}
	err = write(recs, retrieved)
}

func collect(r recommend.Recommender, c corpus) (map[git.File][]git.File, error) {
	data := make(map[git.File][]git.File)
	for _, path := range c.files {
		abs := path.ToLocalFile(c.root)
		_, err := abs.Lstat()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
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
		var docs []git.File
		for _, rec := range recs {
			internalPath, err := localpath.NewAbsolute(rec.Path)
			if err != nil {
				return nil, err
			}
			truePath, ok := c.index[internalPath]
			if !ok {
				continue
			}
			rel, err := truePath.RelativeTo(c.root)
			if err != nil {
				return nil, err
			}
			gitFile := git.File(filepath.ToSlash(string(rel)))
			if _, ok := c.docs[gitFile]; ok {
				docs = append(docs, gitFile)
			}
		}
		data[path] = docs
	}
	return data, nil
}

func buildRecommender(kiteco localpath.Absolute) (recommend.Recommender, error) {
	var (
		ignoreOpts = ignore.Options{
			Root:           kiteco,
			IgnorePatterns: []string{".*", "bindata.go", "node_modules"},
		}
		storageOpts = git.StorageOptions{
			UseDisk: true,
			Path:    string(kiteco.Join("kite-go", "navigation", "offline", "git-cache.json")),
		}
		recOpts = recommend.Options{
			Root:        kiteco,
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
	root  localpath.Absolute
	files []git.File
	docs  map[git.File]bool
	index map[localpath.Absolute]localpath.Absolute
}

func readCorpus(root, relevantPath, indexPath localpath.Absolute) (corpus, error) {
	relevant, err := readRelevant(relevantPath)
	if err != nil {
		return corpus{}, err
	}
	index, err := readIndex(indexPath)
	if err != nil {
		return corpus{}, err
	}
	docs := make(map[git.File]bool)
	files := make(map[git.File]bool)
	for f, rs := range relevant {
		_, err := f.ToLocalFile(root).Lstat()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return corpus{}, err
		}
		var keepFile bool
		for _, r := range rs {
			_, err := r.ToLocalFile(root).Lstat()
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return corpus{}, err
			}
			docs[r] = true
			keepFile = true
		}
		if keepFile {
			files[f] = true
		}
	}
	var flat []git.File
	for f := range files {
		flat = append(flat, f)
	}
	return corpus{
		root:  root,
		files: flat,
		docs:  docs,
		index: index,
	}, nil
}

func readRelevant(relevantPath localpath.Absolute) (map[git.File][]git.File, error) {
	file, err := relevantPath.Open()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var relevant map[git.File][]git.File
	err = json.Unmarshal(data, &relevant)
	if err != nil {
		return nil, err
	}
	return relevant, nil
}

func readIndex(indexPath localpath.Absolute) (map[localpath.Absolute]localpath.Absolute, error) {
	file, err := indexPath.Open()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var index map[localpath.Absolute]localpath.Absolute
	err = json.Unmarshal(data, &index)
	if err != nil {
		return nil, err
	}
	return index, nil
}

func write(recs map[git.File][]git.File, retrievedPath localpath.Absolute) error {
	data, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(string(retrievedPath), data, 0600)
}
