package main

import (
	"encoding/json"
	"errors"
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
		KeepUnderscores bool
		WritePath       string
	}{}
	arg.MustParse(&args)
	if args.WritePath == "" {
		log.Fatal(errors.New("WritePath must not be empty"))
	}
	recOpts.KeepUnderscores = args.KeepUnderscores

	log.Println("building recommender")
	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		log.Fatal(err)
	}
	s, err := git.NewStorage(storageOpts)
	if err != nil {
		log.Fatal(err)
	}
	rec, err := recommend.NewRecommender(kitectx.Background(), recOpts, ignorer, s)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("processing examples")
	var examples []example
	for _, in := range inputs {
		res, err := in.computeResult(rec)
		if err != nil {
			log.Fatal(err)
		}
		ex := example{
			Input:  in,
			Result: res,
		}
		examples = append(examples, ex)
	}

	log.Println("writing")
	data, err := json.MarshalIndent(examples, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile(args.WritePath, data, 0600)
}

type example struct {
	Input  input
	Result result
}

type input struct {
	CurrentPath string
	RelatedPath string
}

type result struct {
	PathRank int
	Keywords []string
}

func (in input) computeResult(r recommend.Recommender) (result, error) {
	pathRank, err := in.computePathRank(r)
	if err != nil {
		return result{}, err
	}
	keywords, err := in.computeTopKeywords(r, numKeywords)
	if err != nil {
		return result{}, err
	}
	res := result{
		PathRank: pathRank,
		Keywords: keywords,
	}
	return res, nil
}

func (in input) computePathRank(r recommend.Recommender) (int, error) {
	request := recommend.Request{
		Location: recommend.Location{
			CurrentPath: toAbsolutePath(in.CurrentPath),
		},
		MaxFileRecs: -1,
	}
	files, err := r.Recommend(kitectx.Background(), request)
	if err != nil {
		return -1, err
	}
	absoluteRelatedPath := toAbsolutePath(in.RelatedPath)
	for i, file := range files {
		if file.Path == absoluteRelatedPath {
			return i + 1, nil
		}
	}
	return -1, errors.New("RelatedPath unranked")
}

func (in input) computeTopKeywords(r recommend.Recommender, numKeywords int) ([]string, error) {
	request := recommend.BlockRequest{
		Request: recommend.Request{
			Location: recommend.Location{
				CurrentPath: toAbsolutePath(in.CurrentPath),
			},
			MaxBlockRecs:     -1,
			MaxFileKeywords:  -1,
			MaxBlockKeywords: -1,
		},
		InspectFiles: []recommend.File{{Path: toAbsolutePath(in.RelatedPath)}},
	}
	files, err := r.RecommendBlocks(kitectx.Background(), request)
	if err != nil {
		return nil, err
	}
	var keywords []string
	for _, keyword := range files[0].Keywords {
		keywords = append(keywords, keyword.Word)
	}
	if len(keywords) > numKeywords {
		keywords = keywords[:numKeywords]
	}
	return keywords, nil
}

func toAbsolutePath(slashRelativePath string) string {
	return string(kiteco.Join(localpath.Relative(filepath.FromSlash(slashRelativePath))))
}

var (
	inputs = []input{
		input{
			CurrentPath: "sidebar/src/components/WindowMode/index.tsx",
			RelatedPath: "sidebar/src/components/WindowMode/index.module.css",
		},
		input{
			CurrentPath: "sidebar/src/containers/Logs.js",
			RelatedPath: "sidebar/src/assets/logs.css",
		},
		input{
			CurrentPath: "sidebar/src/containers/Examples/assets/code-example.css",
			RelatedPath: "sidebar/src/containers/Examples/components/CodeExample.js",
		},
		input{
			CurrentPath: "kite-python/analysis/conversion-model/model.py",
			RelatedPath: "kite-go/client/internal/conversion/monetizable/model/model.go",
		},
		input{
			CurrentPath: "kite-go/lang/python/pythongraph/graph.go",
			RelatedPath: "kite-python/kite_ml/kite/graph_data/graph.py",
		},
		input{
			CurrentPath: "kite-go/lang/python/pythongraph/graph.go",
			RelatedPath: "kite-go/lang/python/pythongraph/rendered/templates/graph.html",
		},
		input{
			CurrentPath: "kite-python/kite_emr/kite/emr/bundle.py",
			RelatedPath: "kite-python/kite_emr/kite/emr/constants.py",
		},
	}

	kiteco     = localpath.Absolute(os.Getenv("GOPATH")).Join("src", "github.com", "kiteco", "kiteco")
	ignoreOpts = ignore.Options{
		Root:           kiteco,
		IgnorePatterns: []string{".*", "vendor", "bindata", "node_modules"},
	}
	storageOpts = git.StorageOptions{
		UseDisk: true,
		Path: filepath.Join(
			os.Getenv("GOPATH"),
			"src", "github.com", "kiteco", "kiteco",
			"kite-go", "navigation", "offline", "git-cache.json",
		),
	}
	recOpts = recommend.Options{
		Root:                 kiteco,
		MaxFileSize:          1e6,
		MaxFiles:             1e5,
		UseCommits:           true,
		ComputedCommitsLimit: git.DefaultComputedCommitsLimit,
	}
	numKeywords = 5
)
