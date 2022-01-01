package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
	"unicode"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func getDataset(dirs []string, maxFiles int, cacheRoot string) *source.Dataset {
	var files []string
	for _, dir := range dirs {
		log.Println("using input dir:", dir)
		dirFiles, err := aggregator.ListDir(dir)
		fail(err)
		files = append(files, dirFiles...)
	}

	sort.Strings(files)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = runtime.NumCPU()
	emrOpts.MaxFileSize = 1 << 17 // 128kb
	emrOpts.MaxRecords = maxFiles
	emrOpts.CacheRoot = cacheRoot
	return source.NewEMRDataset("train-and-validate-corpus", emrOpts, files)
}

func isSymbol(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
		return false
	}
	return true
}

type counts map[rune]int
type metric struct {
	Symbol string  `json:"sym"`
	Ratio  float64 `json:"ratio"`
	Total  int     `json:"total"`
}

func main() {
	args := struct {
		Output    string
		MaxFiles  int
		CacheRoot string
	}{
		Output:    "lineendings.json",
		MaxFiles:  1e5,
		CacheRoot: "./data/kite",
	}

	arg.MustParse(&args)
	lineEndings := make(map[string]counts)
	midLines := make(map[string]counts)

	var m sync.Mutex

	for _, ext := range utils.TextExtensions {
		lineEndings[ext] = make(counts)
		midLines[ext] = make(counts)
		start := time.Now()
		var numFiles int
		root := utils.TextSplitRootForExt(ext)
		dirs := []string{
			fileutil.Join(root, string(utils.TrainDataset)),
			fileutil.Join(root, string(utils.ValidateDataset)),
		}

		srcs := getDataset(dirs, args.MaxFiles, args.CacheRoot)
		count := dependent.NewFromFunc("count-line-endings-"+ext, func(s pipeline.Sample) {
			kv := s.(pipeline.Keyed)
			currentEndings := make(counts)
			currentMiddles := make(counts)

			bs := []byte(kv.Sample.(sample.ByteSlice))
			for i, r := range string(bs) {
				if !isSymbol(r) || i == len(bs)-1 {
					continue
				}
				next := rune(bs[i+1])
				if next == '\r' || next == '\n' {
					currentEndings[r]++
				} else {
					currentMiddles[r]++
				}
			}

			m.Lock()
			defer m.Unlock()
			for s, c := range currentEndings {
				lineEndings[ext][s] += c
			}
			for s, c := range currentMiddles {
				midLines[ext][s] += c
			}
			numFiles++
		})
		pm := make(pipeline.ParentMap)
		pm.Chain(
			srcs,
			count,
		)

		pipe := pipeline.Pipeline{
			Name:    "wordcount",
			Parents: pm,
			Sources: []pipeline.Source{srcs},
			ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
				res := []rundb.Result{
					{
						Name:  "Duration",
						Value: fmt.Sprintf("%v", time.Since(start)),
					},
					{
						Name:  "Files",
						Value: numFiles,
					},
				}
				for _, r := range res {
					fmt.Println(r.Name, r.Value)
				}
				return res
			},
		}

		engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
			NumWorkers: runtime.NumCPU() * 3,
		})
		fail(err)

		_, err = engine.Run()
		fail(err)
	}

	// Write to output
	ranked := make(map[string][]metric)
	for _, ext := range utils.TextExtensions {
		var rs []metric
		for k, v := range lineEndings[ext] {
			if m, ok := midLines[ext][k]; ok {
				rs = append(rs, metric{
					Symbol: string(k),
					Ratio:  float64(v) / float64(v+m),
					Total:  v + m,
				})
			}
		}
		sort.Slice(rs, func(i int, j int) bool {
			return rs[i].Ratio > rs[j].Ratio
		})
		ranked[ext] = rs
	}

	f, err := os.Create(args.Output)
	fail(err)
	defer f.Close()

	err = json.NewEncoder(f).Encode(&ranked)
	fail(err)
}
