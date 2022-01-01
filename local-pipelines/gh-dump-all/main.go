package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func splitKey(key string) (repo, fn string) {
	parts := strings.Split(key, ":")
	repo, fn = parts[0], parts[1]
	return
}

func clean(extension string) string {
	return strings.TrimLeft(extension, ".")
}

func getRepos(metadata string, uniqueReposOnly bool) []string {
	gh, err := source.NewGHMetadata(source.DefaultDatasetOpts, "metadata", metadata)
	maybeQuit(err)

	recs, err := source.ReadAll(gh)
	maybeQuit(err)

	var repos []string
	for _, r := range recs {
		md := r.Value.(sample.GHRepoMetadata)
		if uniqueReposOnly && md.ForkedFrom != -1 {
			continue
		}
		repos = append(repos, md.Path)
	}

	return repos
}

type fileAndHash struct {
	Name     string
	Repo     string
	Contents []byte
	Hash     uint64
}

func extFilter(maxSize int, extMap map[string]bool) func(string, int) bool {
	return func(name string, size int) bool {
		if size > maxSize {
			return true
		}

		if !extMap[filepath.Ext(name)] {
			return true
		}

		return false
	}
}

// This is always write to every emr data sets toa avoid empty file on s3
// Please filter it out when actually using the dataset
var dummyFile = fileAndHash{
	Name:     "KITE_DUMMY_FILE",
	Repo:     "KITE_DUMMY_REPO",
	Contents: []byte("KITE_DUMMY_CONTENT"),
	Hash:     spooky.Hash64([]byte("KITE_DUMMY_CONTENT")),
}

func main() {
	args := struct {
		Metadata         string
		Out              string
		RunDB            string
		FilesPerBlock    int
		UniqueRepos      bool
		MaxFileSizeBytes int
		Verbose          bool
		TmpDir           string
	}{
		Metadata:         "s3://kite-local-pipelines/gh-dump-metadata-non-python/2019-11-06_09-47-07-PM",
		Out:              "s3://kite-local-pipelines/gh-dump-all/",
		RunDB:            rundb.DefaultRunDB,
		FilesPerBlock:    1e6,
		UniqueRepos:      true,
		MaxFileSizeBytes: 500000,
		Verbose:          true,
		TmpDir:           "/data/kite-local-pipelines/gh-dump-all/tmp",
	}

	arg.MustParse(&args)
	defer os.RemoveAll(args.TmpDir)

	// Get the list of file extensions
	extMap := make(map[string]bool)
	for _, e := range extensions {
		extMap[e] = true
	}

	start := time.Now()

	repos := getRepos(args.Metadata, args.UniqueRepos)
	fmt.Printf("Found %d repos to extract files from\n", len(repos))

	var logger io.Writer
	if args.Verbose {
		logger = os.Stderr
	}

	crawlOpts := source.DefaultRawGHOpts
	crawlOpts.Logger = logger
	crawlOpts.NumGo = 4
	crawlOpts.NoCache = true
	crawlOpts.UTF8EncodeNames = true
	crawlOpts.UTF8EncodeContents = true
	crawlOpts.Skip = extFilter(args.MaxFileSizeBytes, extMap)
	crawl := source.NewRawGHCrawl(crawlOpts, "crawl", repos)

	var m sync.Mutex
	var total int64
	seen := make(map[uint64]bool, 3*len(repos))
	dedupe := transform.NewMap("dedupe", func(s pipeline.Sample) []pipeline.Sample {
		ghr := s.(sample.GHRepo)
		repo := fmt.Sprintf("%s/%s", ghr.Meta.Owner, ghr.Meta.Repo)

		fs := make([]fileAndHash, 0, len(ghr.Files))
		for _, f := range ghr.Files {
			fs = append(fs, fileAndHash{
				Name:     f.Name,
				Repo:     repo,
				Contents: f.Contents,
				Hash:     spooky.Hash64(f.Contents),
			})
		}

		m.Lock()
		defer m.Unlock()
		deduped := make([]pipeline.Sample, 0, len(fs))
		for _, f := range fs {
			if seen[f.Hash] {
				continue
			}
			seen[f.Hash] = true
			deduped = append(deduped, pipeline.Keyed{
				Key:    fmt.Sprintf("%s:%s", f.Repo, f.Name),
				Sample: sample.ByteSlice(f.Contents),
			})
			total++
		}

		// Add dummyFile
		if !seen[dummyFile.Hash] {
			deduped = append(deduped, pipeline.Keyed{
				Key:    fmt.Sprintf("%s:%s", dummyFile.Repo, dummyFile.Name),
				Sample: sample.ByteSlice(dummyFile.Contents),
			})
			seen[dummyFile.Hash] = true
		}

		return deduped
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(crawl, dedupe)
	counts := make(map[string]int)

	getExtensionFilter := func(extension string) *transform.Filter {
		return transform.NewFilter("filter-"+clean(extension), func(s pipeline.Sample) bool {
			kv := s.(pipeline.Keyed)
			repoName, fileName := splitKey(kv.Key)
			// For each filter, at least include the dummy file so that we don't write empty content to s3
			if repoName == dummyFile.Repo && fileName == dummyFile.Name {
				return true
			}
			if filepath.Ext(fileName) == extension {
				counts[extension]++
				return true
			}
			return false
		})
	}

	writerOpts := aggregator.DefaultWriterOpts
	writerOpts.NumGo = 1
	writerOpts.FilePrefix = "files"
	writerOpts.SamplesPerFile = args.FilesPerBlock
	timestamp := time.Now().Format("2006-01-02_03-04-05-PM")

	for e := range extMap {
		filter := getExtensionFilter(e)

		out := fileutil.Join(args.Out, timestamp, clean(e))
		writerOpts.TmpDir = fileutil.Join(args.TmpDir, clean(e))
		writer := aggregator.NewEMRWriter(writerOpts, "writer-"+clean(e), out)

		pm.Chain(dedupe, filter)
		pm.Chain(filter, writer)
	}

	opts := pipeline.DefaultEngineOptions
	opts.NumWorkers = 1
	opts.Role = pipeline.Standalone
	opts.RunName = args.Metadata
	opts.RunDBPath = args.RunDB

	pipe := pipeline.Pipeline{
		Name:    "gh-dump-all",
		Parents: pm,
		Sources: []pipeline.Source{crawl},
		Params: map[string]interface{}{
			"metadata": args.Metadata,
		},
		ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
			return []rundb.Result{
				{
					Name:  "runtime",
					Value: fmt.Sprintf("%v", time.Since(start)),
				},
				{
					Name:  "num files",
					Value: total,
				},
				{
					Name:  "extension to count",
					Value: counts,
				},
				{
					Name:  "metadata",
					Value: rundb.RenderS3ObjectLink(args.Metadata, args.Metadata),
				},
				{
					Name:  "results",
					Value: rundb.RenderS3DirLink(args.Out, args.Out),
				},
			}
		},
	}

	engine, err := pipeline.NewEngine(pipe, opts)
	maybeQuit(err)

	_, err = engine.Run()
	maybeQuit(err)

	fmt.Println("Done! Took", time.Since(start))
}
