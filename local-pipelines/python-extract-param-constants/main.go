package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const maxSizeBytes = 1000000
const maxAnalysisInterval = 2 * time.Second
const maxParseInterval = 1 * time.Second

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mergeConstInfo(c1 pythoncode.ConstInfo, c2 pythoncode.ConstInfo) pythoncode.ConstInfo {
	for k, v := range c2 {
		c1[k] += v
	}
	return c1
}

// Pair is a helper data structure to hold a key/value pair.
type pair struct {
	Key   string
	Value int32
}

func topKConstInfo(ci pythoncode.ConstInfo, topK int) (pythoncode.ConstInfo, int32) {
	var p []pair
	for k, v := range ci {
		p = append(p, pair{k, v})
	}
	sort.Slice(p, func(i, j int) bool {
		return p[i].Value > p[j].Value
	})
	if len(p) > topK {
		p = p[:topK]
	}
	var topKCounts int32
	newInfo := make(pythoncode.ConstInfo)
	for _, entry := range p {
		newInfo[entry.Key] = entry.Value
		topKCounts += entry.Value
	}
	return newInfo, topKCounts
}

func mergeInfo(current pythoncode.ArgConstInfo, addOn pythoncode.ArgConstInfo) pythoncode.ArgConstInfo {
	for k, v := range addOn {
		if _, ok := current[k]; !ok {
			current[k] = v
		} else {
			current[k] = pythoncode.TypedConstInfo{
				IntConstInfo:    mergeConstInfo(current[k].IntConstInfo, v.IntConstInfo),
				StringConstInfo: mergeConstInfo(current[k].StringConstInfo, v.StringConstInfo),
			}
		}
	}

	return current
}

func filterConstInfo(info pythoncode.ConstInfo, minFreq int32, topK int, topKMinRatio float64) pythoncode.ConstInfo {
	var totalCounts int32
	for _, v := range info {
		totalCounts += v
	}
	newInfo, topKCounts := topKConstInfo(info, topK)

	if float64(topKCounts)/float64(totalCounts) < topKMinRatio {
		fmt.Printf("Top %v consts are only %f%% of the total counts, need to be at least %f%%, skipping...\n",
			topK, 100*float64(topKCounts)/float64(totalCounts), 100*topKMinRatio)
		return pythoncode.ConstInfo{}
	}

	for k, v := range newInfo {
		if v < minFreq {
			delete(newInfo, k)
		}
	}

	return newInfo
}

func filterFunc(info pythoncode.ArgConstInfo, minFreq int32, topK int, topKMinRatio float64) pythoncode.ArgConstInfo {
	newInfo := make(pythoncode.ArgConstInfo)
	for k, v := range info {
		newIntInfo := filterConstInfo(v.IntConstInfo, minFreq, topK, topKMinRatio)
		newStringInfo := filterConstInfo(v.StringConstInfo, minFreq, topK, topKMinRatio)

		if len(newIntInfo) > 0 || len(newStringInfo) > 0 {
			newInfo[k] = pythoncode.TypedConstInfo{
				IntConstInfo:    newIntInfo,
				StringConstInfo: newStringInfo,
			}
		}
	}
	return newInfo
}

func main() {
	args := struct {
		In           string
		OutBase      string
		NumReaders   int
		NumAnalysis  int
		RunDBPath    string
		CacheRoot    string
		MinConstFreq int32
		TopK         int
		TopKMinRatio float64
	}{
		In:           pythoncode.DedupedCodeDumpPath,
		NumReaders:   3,
		NumAnalysis:  8,
		CacheRoot:    "/data/kite/",
		RunDBPath:    rundb.DefaultRunDB,
		TopK:         10,
		MinConstFreq: 50,
		TopKMinRatio: 0.9,
	}

	arg.MustParse(&args)

	files, err := aggregator.ListDir(args.In)
	fail(err)

	sort.Strings(files)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	outPathFunc := fileutil.Join(args.OutBase, "func-stats.json")
	outPathAttr := fileutil.Join(args.OutBase, "attr-stats.json")

	fmt.Println("datasets loaded, starting processing, writing outputs to:", args.OutBase)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = args.NumReaders
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.CacheRoot = args.CacheRoot

	srcs := source.NewEMRDataset("srcs", emrOpts, files)

	parsed := transform.NewOneInOneOut("parsed", func(s pipeline.Sample) pipeline.Sample {
		kv := s.(pipeline.Keyed)
		return pythonpipeline.ParsedNonNil(parseOpts, maxParseInterval)(kv.Sample)
	})

	resolved := transform.NewOneInOneOut("resolved", pythonpipeline.ResolvedNonNil(rm, maxAnalysisInterval))

	extractedFunc := transform.NewOneInOneOut("extracted-func", func(s pipeline.Sample) pipeline.Sample {
		return extractFuncConstants(rm, s)
	})

	extractedAttr := transform.NewOneInOneOut("extracted-attr", func(s pipeline.Sample) pipeline.Sample {
		return extractAttrStr(s)
	})

	var fileCount uint64
	var m sync.Mutex
	argConsts := make(pythoncode.ArgConstInfoByFunc)
	mergedFunc := dependent.NewFromFunc("merged", func(s pipeline.Sample) {
		m.Lock()
		defer m.Unlock()

		// Symbol (function call) to the map of keyword and counts
		for _, pcs := range s.(symbols) {
			symbol := pcs.Symbol.PathString()
			info := pcs.ArgConstInfo

			if _, ok := argConsts[symbol]; !ok {
				argConsts[symbol] = info
				continue
			}

			argConsts[symbol] = mergeInfo(argConsts[symbol], info)
		}

		fileCount++
	})

	attrBaseConsts := make(pythoncode.ConstInfo)
	mergedAttr := dependent.NewFromFunc("mergedAttr", func(s pipeline.Sample) {
		m.Lock()
		defer m.Unlock()
		attrBaseConsts = mergeConstInfo(attrBaseConsts, pythoncode.ConstInfo(s.(attrConsts)))
	})

	pm := make(pipeline.ParentMap)

	pm.Chain(
		srcs,
		parsed,
		resolved,
		extractedFunc,
		mergedFunc,
	)

	pm.Chain(
		resolved,
		extractedAttr,
		mergedAttr,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-extract-param-constants",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
	}

	start := time.Now()

	eOpts := pipeline.DefaultEngineOptions
	eOpts.NumWorkers = 4
	eOpts.RunDBPath = args.RunDBPath
	engine, err := pipeline.NewEngine(pipe, eOpts)
	fail(err)

	_, err = engine.Run()
	fail(err)

	// Filter out the constants with low frequency
	for k, v := range argConsts {
		fv := filterFunc(v, args.MinConstFreq, args.TopK, args.TopKMinRatio)
		if len(fv) > 0 {
			argConsts[k] = fv
		} else {
			delete(argConsts, k)
		}
	}

	attrBaseConsts = filterConstInfo(attrBaseConsts, args.MinConstFreq, args.TopK, 0)

	fFunc, err := fileutil.NewBufferedWriter(outPathFunc)
	fail(err)

	defer fFunc.Close()
	fail(json.NewEncoder(fFunc).Encode(argConsts))

	fAttr, err := fileutil.NewBufferedWriter(outPathAttr)
	fail(err)

	defer fAttr.Close()
	fail(json.NewEncoder(fAttr).Encode(attrBaseConsts))

	fmt.Printf("Done! Took %v to process %v files, wrote results to %s\n", time.Since(start), fileCount, args.OutBase)
}
