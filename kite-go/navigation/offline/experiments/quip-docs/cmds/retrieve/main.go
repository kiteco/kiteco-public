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

type quipSuffix string

func main() {
	args := struct {
		Relevant  string
		Retrieved string
		Docs      string
	}{}
	arg.MustParse(&args)
	relevant, err := localpath.NewAbsolute(args.Relevant)
	if err != nil {
		log.Fatal(err)
	}
	retrieved, err := localpath.NewAbsolute(args.Retrieved)
	if err != nil {
		log.Fatal(err)
	}
	docs, err := localpath.NewAbsolute(args.Docs)
	if err != nil {
		log.Fatal(err)
	}

	kiteco := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco")
	root, err := localpath.NewAbsolute(kiteco)
	if err != nil {
		log.Fatal(err)
	}
	c, err := readCorpus(root, relevant, docs)
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

func collect(r recommend.Recommender, c corpus) (map[git.File][]quipSuffix, error) {
	data := make(map[git.File][]quipSuffix)
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
		var docs []quipSuffix
		for _, rec := range recs {
			internalPath, err := localpath.NewAbsolute(rec.Path)
			if err != nil {
				return nil, err
			}
			if internalPath.Dir() != c.docsDir {
				continue
			}
			doc, ok := getQuipDoc(internalPath)
			if !ok {
				continue
			}
			docs = append(docs, doc)
		}
		data[path] = docs
	}
	return data, nil
}

func getQuipDoc(path localpath.Absolute) (quipSuffix, bool) {
	base := filepath.Base(string(path))
	if len(base) < 3 {
		return "", false
	}
	if base[len(base)-3:] != ".py" {
		return "", false
	}
	return quipSuffix(base[:len(base)-3]), true
}

func buildRecommender(root localpath.Absolute) (recommend.Recommender, error) {
	var (
		ignoreOpts = ignore.Options{
			Root:           root,
			IgnorePatterns: []string{".*", "bindata.go", "node_modules"},
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
	root    localpath.Absolute
	files   []git.File
	docsDir localpath.Absolute
}

func readCorpus(root, relevantPath, docsDir localpath.Absolute) (corpus, error) {
	relevant, err := readRelevant(relevantPath)
	if err != nil {
		return corpus{}, err
	}
	var files []git.File
	for f := range relevant {
		_, err := f.ToLocalFile(root).Lstat()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return corpus{}, err
		}
		files = append(files, f)
	}
	return corpus{
		root:    root,
		files:   files,
		docsDir: docsDir,
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
	for code, quips := range triples {
		for quip := range quips {
			relevant[code] = append(relevant[code], quip)
		}
	}
	return relevant, nil
}

func write(recs map[git.File][]quipSuffix, retrievedPath localpath.Absolute) error {
	data, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(string(retrievedPath), data, 0600)
}
