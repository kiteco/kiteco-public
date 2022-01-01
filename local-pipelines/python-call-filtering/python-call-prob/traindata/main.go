package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync/atomic"
	"time"

	"github.com/kiteco/kiteco/kite-golib/fileutil"

	callprobutils "github.com/kiteco/kiteco/local-pipelines/python-call-filtering/python-call-prob/call-prob-utils"

	"github.com/kiteco/kiteco/kite-golib/tensorflow"
	"github.com/kiteco/kiteco/local-pipelines/python-call-filtering/internal/utils"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprob"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const (
	maxFileSize = 50000
)

var (
	datasetPath = pythoncode.DedupedCodeDumpPath
)

// sampleCounter is used to count the number of samples for each label
// Each slice contains the count for 0 arg, 1 arg, 2 args and more than 2 args
// labels0 -> Negative samples, labels1 -> Positive samples
type sampleCounter struct {
	labels0 [utils.ArgCategories + 1]uint64 // count of negative samples (per arg)
	labels1 [utils.ArgCategories + 1]uint64 // count of positive samples (per arg)
}

// sampleStore is used to store positive and negative sample. It is used as a temp storage before doing the subsampling
// Each sample store contains samples with the same number of args (1 sampleStore per arg count)
type sampleStore struct {
	labels0 []flatTrainSample // TrainSamples associated with a negative label
	labels1 []flatTrainSample // TrainSamples associated with a positive label
}

// sampleMap is a map of sampleStore indexed by the number of arg in the samples
// We need that to subsample the dataset per number of args
type sampleMap map[int]sampleStore

func (sm sampleMap) Add(other sample.Addable) sample.Addable {
	smo := other.(sampleMap)
	for k, sso := range smo {
		ss, ok := sm[k]
		if !ok {
			sm[k] = sso
		} else {
			ss.labels0 = append(ss.labels0, sso.labels0...)
			ss.labels1 = append(ss.labels1, sso.labels1...)
			sm[k] = ss
		}
	}
	return sm
}

func (sampleMap) SampleTag() {}

type distributionStore map[string][]float32

func (distributionStore) SampleTag() {}

func newDistributionStore() sample.Addable {
	return make(distributionStore)
}

func (ds distributionStore) Add(other sample.Addable) sample.Addable {
	ods := other.(distributionStore)
	for n, osl := range ods {
		sl := ds[n]
		sl = append(sl, osl...)
		ds[n] = sl
	}
	return ds
}

func storeDistribution(s pipeline.Sample) sample.Addable {
	result := make(distributionStore)
	ts := s.(flatTrainSample)
	for _, v := range ts.Features.Contextual.Weights() {
		sl := result[v.Name]
		sl = append(sl, v.Weight)
		result[v.Name] = sl
	}
	for _, v := range ts.Features.Comp.Weights() {
		sl := result[v.Name]
		sl = append(sl, v.Weight)
		result[v.Name] = sl
	}
	l := float32(0.0)
	if ts.Label {
		l = 1.0
	}
	result["label"] = []float32{l}
	return result
}

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func storeSample(s pipeline.Sample) sample.Addable {
	result := newSampleMap().(sampleMap)
	ts := s.(flatTrainSample)

	if argCount := ts.Features.Comp.NumArgs; argCount < utils.ArgCategories {
		st := result[argCount]
		if ts.Label {
			st.labels1 = append(st.labels1, ts)
		} else {
			st.labels0 = append(st.labels0, ts)
		}
		result[argCount] = st
	}
	return result
}

// truncateCallsTransformer filter predictions to only keep partial call (without closing parenthesis)
func truncateCallsTransformer(sample pipeline.Sample) pipeline.Sample {
	si := sample.(callprobutils.SampleInputs)
	si.CallComps = utils.TruncateCalls(si.CallComps)
	return si
}

// removeIncompleteCalls filter predictions to only keep complete call (with closing parenthesis)
func removeIncompleteCalls(sample pipeline.Sample) pipeline.Sample {
	si := sample.(callprobutils.SampleInputs)
	si.CallComps = utils.KeepOnlyCompleteCalls(si.CallComps)
	return si
}

func newSampleAggregator(name string) pipeline.Aggregator {
	return aggregator.NewSumAggregator(name, newSampleMap, storeSample)
}

type trainSample callprob.TrainSample

func (trainSample) SampleTag() {}

type options struct {
	MaxFiles int
	// OutDir can be a local or S3 directory
	OutDir    string
	NumReader int
}

type pipe struct {
	pipeline.Pipeline
	TruncatedCallStorer, FullCallStorer pipeline.Aggregator
	TruncDist, FullDist                 pipeline.Aggregator
}

func inSlice(s []int, i int) bool {
	for _, ii := range s {
		if i == ii {
			return true
		}
	}
	return false
}

type flatTrainSample callprob.FlatTrainSample

func (flatTrainSample) SampleTag() {}

func flattenSamples(sample pipeline.Sample) []pipeline.Sample {
	ts := sample.(trainSample)
	var result []pipeline.Sample
	for i, s := range ts.Features.Comp {
		result = append(result, flatTrainSample{
			Features: callprob.FlatFeatures{
				Contextual: ts.Features.Contextual,
				Comp:       s,
			},
			Label: inSlice(ts.Labels, i),
			Meta: callprob.FlatTrainSampleMeta{
				Hash:           ts.Meta.Hash,
				Cursor:         ts.Meta.Cursor,
				CompIdentifier: ts.Meta.CompIdentifiers[i],
			},
		})
	}
	return result
}

func countSample(counter *sampleCounter) func(pipeline.Sample) pipeline.Sample {
	return func(sample pipeline.Sample) pipeline.Sample {
		ts := sample.(trainSample)
		for i, c := range ts.Features.Comp {
			target := &counter.labels0
			if inSlice(ts.Labels, i) {
				target = &counter.labels1
			}
			count := c.NumArgs
			if count > utils.ArgCategories {
				count = utils.ArgCategories
			}
			atomic.AddUint64(&target[count], 1)
		}
		return sample
	}
}

func newSampleMap() sample.Addable {
	return make(sampleMap)
}

func bucketByLabels(targets, labels []float32) (negative, positive []rundb.HistogramData) {
	var result [2][]rundb.HistogramData
	for i, v := range targets {
		result[int(labels[i])] = append(result[int(labels[i])], rundb.HistogramData(v))
	}
	return result[0], result[1]
}

func getMinMax(d []float32) (min, max float32) {
	min, max = d[0], d[0]
	for _, v := range d {
		if min > v {
			min = v
		}
		if max < v {
			max = v
		}
	}
	return min, max
}

func getHistograms(dsTrunc, dsFull distributionStore, title string, drawer *rundb.HistogramDrawer) []rundb.Result {
	var result []rundb.Result
	maxLength := len(dsTrunc)
	if len(dsFull) > maxLength {
		maxLength = len(dsFull)
	}

	keys := make([]string, 0, maxLength)
	for n := range dsTrunc {
		keys = append(keys, n)
	}
	sort.Strings(keys)

	for _, n := range keys {
		negT, posT := bucketByLabels(dsTrunc[n], dsTrunc["label"])
		negF, posF := bucketByLabels(dsFull[n], dsFull["label"])
		title := fmt.Sprintf("%s - %s", n, title)
		min, max := getMinMax(dsTrunc[n])
		minF, maxF := getMinMax(dsFull[n])
		if min > minF {
			min = minF
		}
		if max < maxF {
			max = maxF
		}
		data := [][]rundb.HistogramData{negT, negF, posT, posF}
		labels := []string{"Neg trunc", "Neg full", "Pos trunc", "Pos full"}
		str, err := drawer.GetMultiSeriesHistogramString(data, labels, 800, 450, title, min, max)
		fail(err)
		result = append(result, rundb.Result{
			Name:       title,
			Value:      str,
			Aggregator: title,
		})
	}
	return result
}

func createPipeline(res utils.Resources, opts options, rng *callprobutils.RNG, truncatedCounters, fullCounters *sampleCounter, printHistograms bool) pipe {

	files, err := aggregator.ListDir(datasetPath)
	fail(err)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.MaxRecords = opts.MaxFiles
	emrOpts.MaxFileSize = maxFileSize
	emrOpts.NumGo = opts.NumReader

	dataset := source.NewEMRDataset("dataset", emrOpts, files)

	calls := transform.NewMap("calls", func(s pipeline.Sample) []pipeline.Sample {
		k := s.(pipeline.Keyed)
		buf := k.Sample.(sample.ByteSlice)

		ins, err := callprobutils.GetCallsToPredict(k.Key, buf, res, rng.Random())
		if err != nil {
			return []pipeline.Sample{pipeline.WrapError("getCalls error", err)}
		}

		var samples []pipeline.Sample
		for _, in := range ins {
			samples = append(samples, in)
		}
		return samples
	})

	predict := transform.NewOneInOneOut("predict", func(s pipeline.Sample) pipeline.Sample {
		c := s.(callprobutils.CallToPredict)

		return callprobutils.Predict(res, c, rng.Random())
	})

	callTruncator := transform.NewOneInOneOut("Call truncator", truncateCallsTransformer)
	fullCallFilter := transform.NewOneInOneOut("Full Call filter", removeIncompleteCalls)

	samplesFullCall := transform.NewOneInOneOut("samplesFullCall", func(s pipeline.Sample) pipeline.Sample {
		in := s.(callprobutils.SampleInputs)

		ts, err := callprobutils.GetTrainSample(in, res, false)
		if err != nil {
			return pipeline.WrapError("getTrainSample error", err)
		}
		return trainSample(ts)
	})

	samplesTruncated := transform.NewOneInOneOut("samplesTruncated", func(s pipeline.Sample) pipeline.Sample {
		in := s.(callprobutils.SampleInputs)

		ts, err := callprobutils.GetTrainSample(in, res, true)
		if err != nil {
			return pipeline.WrapError("getTrainSample error", err)
		}
		return trainSample(ts)
	})

	counterFullCall := transform.NewOneInOneOut("counterFullCall", countSample(fullCounters))
	counterTruncatedCall := transform.NewOneInOneOut("counterTruncatedCall", countSample(truncatedCounters))

	flattenerFullCall := transform.NewMap("flattener full call", flattenSamples)
	flattenerTruncatedCall := transform.NewMap("flattener truncated calls", flattenSamples)

	fullCallStorer := newSampleAggregator("full call storer")
	truncatedCallStorer := newSampleAggregator("trunc call storer")

	pm := make(pipeline.ParentMap)

	callPrep := pm.Chain(
		dataset,
		calls,
		predict)

	truncCallFlat := pm.Chain(callPrep,
		callTruncator,
		samplesTruncated,
		counterTruncatedCall,
		flattenerTruncatedCall,
	)
	pm.Chain(truncCallFlat,
		truncatedCallStorer,
	)

	fullCallFlat := pm.Chain(callPrep,
		fullCallFilter,
		samplesFullCall,
		counterFullCall,
		flattenerFullCall,
	)
	pm.Chain(fullCallFlat,
		fullCallStorer,
	)

	var truncDist, fullDist pipeline.Aggregator
	if printHistograms {
		fullCallDist := aggregator.NewSumAggregator("Full call Distribution storer", newDistributionStore, storeDistribution)
		truncCallDist := aggregator.NewSumAggregator("Truncated call Distribution storer", newDistributionStore, storeDistribution)
		pm.Chain(fullCallFlat, fullCallDist)
		pm.Chain(truncCallFlat, truncCallDist)
		truncDist = truncCallDist
		fullDist = fullCallDist
	}

	return pipe{
		Pipeline: pipeline.Pipeline{
			Name:    "call-prob-traindata",
			Parents: pm,
			Sources: []pipeline.Source{dataset},
			Params: map[string]interface{}{
				"MaxFiles": opts.MaxFiles,
				"OutDir":   opts.OutDir,
			},
		},
		TruncatedCallStorer: truncatedCallStorer,
		FullCallStorer:      fullCallStorer,
		TruncDist:           truncDist,
		FullDist:            fullDist,
	}
}

func main() {
	datadeps.Enable()
	args := struct {
		MaxFiles             int
		OutDir               string
		RunDBPath            string
		NumReaders           int
		NumWorkers           int
		NumTensorflowThreads int
		Role                 pipeline.Role
		Port                 int
		Endpoints            []string
		ExprShards           string
		RunName              string
		PrintHistograms      bool
	}{
		OutDir:               "./out",
		MaxFiles:             1000,
		NumReaders:           4,
		NumWorkers:           16,
		NumTensorflowThreads: 8,
		RunDBPath:            rundb.DefaultRunDB,
		PrintHistograms:      true,
	}
	tensorflow.SetTensorflowThreadpoolSize(args.NumTensorflowThreads)

	arg.MustParse(&args)

	var runDB string
	if args.RunName != "" {
		runDB = args.RunDBPath
	}

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	modelOpts := pythonmodels.DefaultOptions
	if args.ExprShards != "" {
		shards, err := pythonexpr.ShardsFromFile(args.ExprShards)
		fail(err)
		modelOpts.ExprModelShards = shards
	}

	expr, err := pythonexpr.NewShardedModel(context.Background(), modelOpts.ExprModelShards, modelOpts.ExprModelOpts)
	fail(err)

	models := pythonmodels.Models{
		Expr: expr,
	}

	res := utils.Resources{RM: rm, Models: &models}

	opts := options{
		MaxFiles:  args.MaxFiles,
		OutDir:    args.OutDir,
		NumReader: args.NumReaders,
	}
	fullWriter, fullOutFile, err := createOutputFile(opts.OutDir, "full_call_samples.json")
	fail(err)
	defer fullWriter.Close()
	truncWriter, truncOutFile, err := createOutputFile(opts.OutDir, "truncated_call_samples.json")
	fail(err)
	defer truncWriter.Close()

	start := time.Now()

	rng := callprobutils.NewRNG(1)

	var fullCounters, truncatedCounters sampleCounter
	pipe := createPipeline(res, opts, rng, &truncatedCounters, &fullCounters, args.PrintHistograms)

	eOpts := pipeline.DefaultEngineOptions
	outf, err := os.Create("log.txt")
	fail(err)
	eOpts.Logger = outf

	eOpts.RunDBPath = runDB
	eOpts.RunName = args.RunName
	eOpts.Role = args.Role
	eOpts.Port = args.Port
	eOpts.ShardEndpoints = args.Endpoints
	eOpts.NumWorkers = args.NumWorkers

	engine, err := pipeline.NewEngine(pipe.Pipeline, eOpts)
	fail(err)

	out, err := engine.Run()

	fail(err)
	fmt.Println("Counter before subsampling")
	printCounters(&truncatedCounters, &fullCounters)

	fullCallStore := out[pipe.FullCallStorer]
	truncCallStore := out[pipe.TruncatedCallStorer]

	var truncatedAfterCounter, fullAfterCounter sampleCounter

	fullCallSamples := subSample(fullCallStore.(sampleMap), rng.Random(), &fullAfterCounter)
	writeSamples(fullCallSamples, fullWriter, rng.Random())
	truncCallSamples := subSample(truncCallStore.(sampleMap), rng.Random(), &truncatedAfterCounter)
	writeSamples(truncCallSamples, truncWriter, rng.Random())
	fmt.Println("After subsampling")
	printCounters(&truncatedAfterCounter, &fullAfterCounter)
	log.Printf("files written: %s and %s\n", fullOutFile, truncOutFile)
	log.Printf("Done! took %v", time.Since(start))

	if args.PrintHistograms {
		subsampledFullCallDist := buildDist(fullCallSamples)
		subsampledTruncCallDist := buildDist(truncCallSamples)
		rundbInstance, err := rundb.NewRunDB(rundb.DefaultRunDB)
		fail(err)
		runInfo := rundb.NewRunInfo(rundbInstance, "call_prob_traindata_distribution", "Summary")
		runInfo.Results = buildRunDbResults(subsampledTruncCallDist, subsampledFullCallDist,
			out[pipe.TruncDist].(distributionStore), out[pipe.FullDist].(distributionStore))
		runInfo.SetStatus(rundb.StatusFinished)
		fail(rundbInstance.SaveRun(runInfo))
	}

}

func buildRunDbResults(sampledTruncDist, sampledFullDist, truncDist, fullDist distributionStore) []rundb.Result {
	var result []rundb.Result
	drawer, err := rundb.NewHistogramDrawer()
	fail(err)
	result = append(result, getHistograms(sampledTruncDist, sampledFullDist, "Subsampled", drawer)...)
	result = append(result, getHistograms(truncDist, fullDist, "All samples", drawer)...)
	return result
}

func buildDist(samples []flatTrainSample) distributionStore {
	result := make(distributionStore)
	for _, s := range samples {
		for _, v := range s.Features.Contextual.Weights() {
			sl := result[v.Name]
			sl = append(sl, v.Weight)
			result[v.Name] = sl
		}
		for _, v := range s.Features.Comp.Weights() {
			sl := result[v.Name]
			sl = append(sl, v.Weight)
			result[v.Name] = sl
		}
		l := float32(0.0)
		if s.Label {
			l = 1.0
		}
		result["label"] = append(result["label"], l)
	}
	return result
}

func subSample(sMap sampleMap, r *rand.Rand, counter *sampleCounter) []flatTrainSample {
	var result []flatTrainSample
	for i := 0; i < utils.ArgCategories; i++ {
		samples, ok := sMap[i]
		if !ok {
			continue
		}
		rate0, rate1 := float64(1), float64(1)
		if len(samples.labels0) > len(samples.labels1) {
			rate0 = float64(len(samples.labels1)) / float64(len(samples.labels0))
		} else {
			if len(samples.labels1) == 0 {
				continue
			}
			rate1 = float64(len(samples.labels0)) / float64(len(samples.labels1))
		}
		for _, s := range samples.labels0 {
			if r.Float64() < rate0 {
				result = append(result, s)
				counter.labels0[i]++
			}
		}
		for _, s := range samples.labels1 {
			if r.Float64() < rate1 {
				result = append(result, s)
				counter.labels1[i]++
			}
		}
	}
	return result
}

func createOutputFile(outDirPath, filename string) (fileutil.NamedWriteCloser, string, error) {
	outFilePath := fileutil.Join(outDirPath, filename)
	outFile, err := fileutil.NewBufferedWriter(outFilePath)
	if err != nil {
		return nil, "", err
	}
	return outFile, outFilePath, nil
}

func writeSamples(samples []flatTrainSample, outFile fileutil.NamedWriteCloser, r *rand.Rand) {
	encoder := json.NewEncoder(outFile)
	for i := range r.Perm(len(samples)) {
		err := encoder.Encode(samples[i])
		fail(err)
	}
}

func printCounters(truncated, full *sampleCounter) {
	fmt.Println("Truncated calls : ")
	printCounter(truncated)
	fmt.Println("Full calls : ")
	printCounter(full)
}

func printCounter(counter *sampleCounter) {
	fmt.Printf("Labels 0 :\n # no arg %d\n # 1 arg %d\n # 2 args %d\n # 3 args and more %d\n", counter.labels0[0], counter.labels0[1], counter.labels0[2], counter.labels0[3])
	fmt.Printf("Labels 1 :\n # no arg %d\n # 1 arg %d\n # 2 args %d\n # 3 args and more %d\n", counter.labels1[0], counter.labels1[1], counter.labels1[2], counter.labels1[3])
}
