package main

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func main() {
	args := struct {
		MaxRecsPerFile int
		RetrievedPath  string
		RepoRoot       string
		GitCache       string
		UseCommits     bool
	}{}
	arg.MustParse(&args)

	c, err := newCollector(args.RepoRoot, args.GitCache, args.MaxRecsPerFile, args.UseCommits)
	if err != nil {
		log.Fatal(err)
	}
	paths, err := c.collectPaths()
	if err != nil {
		log.Fatal(err)
	}
	records, err := c.collectRecords(args.RepoRoot, paths)
	if err != nil {
		log.Fatal(err)
	}
	err = writeRecords(args.RetrievedPath, records)
	if err != nil {
		log.Fatal(err)
	}
}

type collector struct {
	recommender    recommend.Recommender
	maxRecsPerFile int
}

func newCollector(repoRoot, gitCache string, maxRecsPerFile int, useCommits bool) (collector, error) {
	root, err := localpath.NewAbsolute(repoRoot)
	if err != nil {
		return collector{}, err
	}
	var (
		ignoreOpts = ignore.Options{
			Root: root,
			IgnorePatterns: []string{
				".*",
				"bindata.go",
				"node_modules",
				"*.framework/",
				"*.pb.go",
				"testdata/",
				"venv/",
				"kite-go/navigation/offline/experiments",
				"*.css",
				"*.sh",
				"*.html",
				"*.less",
			},
		}
		storageOpts = git.StorageOptions{
			UseDisk: true,
			Path:    gitCache,
		}
		recOpts = recommend.Options{
			Root:                 root,
			MaxFileSize:          1e6,
			MaxFiles:             1e5,
			UseCommits:           useCommits,
			ComputedCommitsLimit: git.DefaultComputedCommitsLimit,
		}
	)

	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		return collector{}, err
	}
	s, err := git.NewStorage(storageOpts)
	if err != nil {
		return collector{}, err
	}
	r, err := recommend.NewRecommender(kitectx.Background(), recOpts, ignorer, s)
	if err != nil {
		return collector{}, err
	}
	return collector{
		recommender:    r,
		maxRecsPerFile: maxRecsPerFile,
	}, nil
}

func (c collector) collectPaths() ([]string, error) {
	files, err := c.recommender.RankedFiles()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, file := range files {
		info, err := os.Stat(file.Path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		paths = append(paths, file.Path)
	}
	return paths, nil
}

type record struct {
	Base        string
	Recommended string
	Score       float64
}

func (c collector) collectRecords(repoRoot string, paths []string) ([]record, error) {
	var records []record
	for _, path := range paths {
		req := recommend.Request{
			MaxFileRecs: c.maxRecsPerFile,
			Location: recommend.Location{
				CurrentPath: path,
			},
		}
		recs, err := c.recommender.Recommend(kitectx.Background(), req)
		if err != nil {
			return nil, err
		}
		relBase, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil, err
		}
		for _, rec := range recs {
			relRecommended, err := filepath.Rel(repoRoot, rec.Path)
			if err != nil {
				return nil, err
			}
			records = append(records, record{
				Base:        relBase,
				Recommended: relRecommended,
				Score:       rec.Probability,
			})
		}
	}
	return records, nil
}

func writeRecords(path string, records []record) error {
	if path == "" {
		log.Println("path is empty, skipping writeRecords")
		return nil
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	for _, record := range records {
		row := []string{
			record.Base,
			record.Recommended,
			strconv.FormatFloat(record.Score, 'f', 6, 64),
		}
		err := writer.Write(row)
		if err != nil {
			return err
		}
	}

	return nil
}
