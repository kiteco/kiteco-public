package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync/atomic"

	"github.com/alexflint/go-arg"
	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type localFile struct {
	Key   string
	Value []byte
}

func main() {
	args := struct {
		Input      string
		Output     string
		ShuffleDir string
		CacheRoot  string
		TmpDir     string
	}{
		Input:      "s3://kite-local-pipelines/gh-dump-js/2019-11-19_12-02-10-AM/js/",
		Output:     "s3://kite-local-pipelines/gh-dump-js-shuffled/2020-01-13_10-30-09-AM/js/",
		ShuffleDir: "/data/shuffle-dump",
		CacheRoot:  "/data/kite",
		TmpDir:     "/data/kite/tmp",
	}

	arg.MustParse(&args)

	err := os.RemoveAll(args.CacheRoot)
	maybeQuit(err)

	err = os.RemoveAll(args.ShuffleDir)
	maybeQuit(err)

	files, err := aggregator.ListDir(args.Input)
	maybeQuit(err)

	sort.Strings(files)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = runtime.NumCPU()
	emrOpts.MaxFileSize = 1 << 18 // 256kb
	emrOpts.CacheRoot = args.CacheRoot

	srcs := source.NewEMRDataset("deduped-corpus", emrOpts, files)

	var count int64
	var dircount int64
	writeLocal := dependent.NewFromFunc("write-local", func(s pipeline.Sample) {
		d := atomic.LoadInt64(&dircount)
		var rootDir string
		if atomic.AddInt64(&count, 1)%100000 == 1 {
			d = atomic.AddInt64(&dircount, 1)
		}

		rootDir = filepath.Join(args.ShuffleDir, fmt.Sprintf("%d", d))
		err := os.MkdirAll(rootDir, os.ModePerm)
		maybeQuit(err)

		kv := s.(pipeline.Keyed)
		bs := []byte(kv.Sample.(sample.ByteSlice))
		fp := spooky.Hash64(bs)
		fn := filepath.Join(rootDir, fmt.Sprintf("%x", fp))

		lf := localFile{
			Key:   kv.Key,
			Value: bs,
		}

		buf, err := json.Marshal(&lf)
		maybeQuit(err)
		err = ioutil.WriteFile(fn, buf, os.ModePerm)
		maybeQuit(err)
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(
		srcs,
		writeLocal,
	)

	pipe := pipeline.Pipeline{
		Name:    "crawl-shuffle",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: runtime.NumCPU(),
	})
	maybeQuit(err)

	_, err = engine.Run()
	maybeQuit(err)

	// --

	log.Println("getting file list")
	var filelist []string
	err = filepath.Walk(args.ShuffleDir, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return err
		}
		filelist = append(filelist, path)
		return err
	})
	maybeQuit(err)

	log.Printf("found %d files, shuffling", len(filelist))
	rand.Shuffle(len(filelist), func(i, j int) {
		filelist[i], filelist[j] = filelist[j], filelist[i]
	})

	log.Println("starting upload of shuffled files")

	localSrcs := source.NewLocalFiles("local-files", runtime.NumCPU(), filelist, os.Stderr)
	reKey := transform.NewOneInOneOut("rekey", func(s pipeline.Sample) pipeline.Sample {
		kv := s.(pipeline.Keyed)
		bs := []byte(kv.Sample.(sample.ByteSlice))

		var lf localFile
		err := json.Unmarshal(bs, &lf)
		maybeQuit(err)

		return pipeline.Keyed{
			Key:    lf.Key,
			Sample: sample.ByteSlice(lf.Value),
		}
	})

	writerOpts := aggregator.DefaultWriterOpts
	writerOpts.NumGo = 1
	writerOpts.FilePrefix = "files"
	writerOpts.SamplesPerFile = 1e6
	writerOpts.TmpDir = args.TmpDir

	err = os.RemoveAll(args.TmpDir)
	maybeQuit(err)

	err = os.MkdirAll(args.TmpDir, os.ModePerm)
	maybeQuit(err)

	writer := aggregator.NewEMRWriter(writerOpts, "writer", args.Output)

	pmShuffled := make(pipeline.ParentMap)
	pmShuffled.Chain(localSrcs, reKey, writer)
	pipeShuffled := pipeline.Pipeline{
		Name:    "crawl-shuffle",
		Parents: pmShuffled,
		Sources: []pipeline.Source{localSrcs},
	}

	engineShuffled, err := pipeline.NewEngine(pipeShuffled, pipeline.EngineOptions{
		NumWorkers: 1,
	})
	maybeQuit(err)

	_, err = engineShuffled.Run()
	maybeQuit(err)
}
