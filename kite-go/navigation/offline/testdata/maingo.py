"""
package main

import (
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"

	_ "net/http/pprof"
)

const (
	maxSizeBytes          = 1 << 17 // 128kb
	minContextToConcat    = 512
	contextConcatMultiple = 4
	validate              = true
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		Lang         string
		Vocab        string
		ContextSize  int
		BatchSize    int
		StepsPerFile int
		Steps        int
		SkipSteps    int
		TrainDir     string
		ValidateDir  string
		CacheRoot    string
		TmpDir       string
		NumGPU       int
	}{
		CacheRoot: "/data/kite",
	}

	arg.MustParse(&args)

	if args.ContextSize <= 0 {
		log.Fatalln("--contextsize must be set to a value > 0")
	}

	if args.BatchSize <= 0 {
		log.Fatalln("--batchsize must be set to a value > 0")
	}

	if args.StepsPerFile <= 0 {
		log.Fatalln("--stepsperfile must be set to a value > 0")
	}

	if args.Steps <= 0 {
		log.Fatalln("--steps must be set to a value > 0")
	}

	if args.NumGPU <= 0 {
		log.Fatalln("--numgpu must be set to a value > 0")
	}

	if args.TrainDir == "" {
		log.Fatalln("--traindir must be set")
	}

	if args.ValidateDir == "" {
		log.Fatalln("--validatedir must be set")
	}

	if args.TmpDir == "" {
		log.Fatalln("--tmpdir must be set")
	}

	if args.Steps < args.StepsPerFile {
		args.StepsPerFile = args.Steps
		log.Printf("steps %d < stepsperfile %d, setting stepsperfile to %d", args.Steps, args.StepsPerFile, args.Steps)
	}

	langGroup := lexicalv0.MustLangGroupFromName(args.Lang)

	trainDatasets := datasets(args.CacheRoot, langGroup, utils.TrainDataset)
	validateDatasets := datasets(args.CacheRoot, langGroup, utils.ValidateDataset)

	trainSampleWriter := utils.NewSampleWriter(args.TrainDir, args.TmpDir, args.BatchSize, args.StepsPerFile)
	defer trainSampleWriter.Flush()

	validateSampleWriter := utils.NewSampleWriter(args.ValidateDir, args.TmpDir, args.BatchSize, args.StepsPerFile)
	defer validateSampleWriter.Flush()

	enc, err := lexicalv0.NewFileEncoder(args.Vocab, langGroup)
	fail(err)

	filesToSkip := args.SkipSteps * args.BatchSize
	log.Printf("skipping %d steps, or %d files", args.SkipSteps, filesToSkip)

	trainDatagen := datagen(args.ContextSize, enc, trainSampleWriter)
	validateDatagen := datagen(args.ContextSize, enc, validateSampleWriter)

	go datasetCycler(filesToSkip, trainDatasets, trainDatagen)
	go datasetCycler(filesToSkip, validateDatasets, validateDatagen)

	log.Println("starting...")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	targetSteps := args.NumGPU * int(float32(args.Steps)*1.1)
	for {
		select {
		case <-ticker.C:
			// Assume if the training writer has written enough steps,
			// the validate writer *definitely* has written enough steps.
			// Inflated by 10% for a bit of a cushion
			if trainSampleWriter.StepsWritten() >= targetSteps &&
				validateSampleWriter.StepsWritten() >= targetSteps {
				log.Printf("reached steps target of %d, exiting. steps written: %d", args.Steps, trainSampleWriter.StepsWritten())
				return
			}
		}
	}
}

func datasets(cacheRoot string, langGroup lexicalv0.LangGroup, dt utils.DatasetType) []*source.Dataset {
	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = 1
	emrOpts.Epochs = 1e6
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.CacheRoot = cacheRoot
	emrOpts.PanicOnError = false

	var ds []*source.Dataset
	for idx, input := range utils.DatasetForLang(dt, langGroup) {
		files, err := aggregator.ListDir(input)
		fail(err)
		ds = append(ds, source.NewEMRDataset(
			fmt.Sprintf("%s-%d-%v-corpus", langGroup.Name(), idx, dt), emrOpts, files))
	}
	return ds
}

func datagen(contextSize int, enc *lexicalv0.FileEncoder, writer *utils.SampleWriter) func(pipeline.Record) {
	var m sync.Mutex
	concatByExt := make(map[string][]int)

	return func(record pipeline.Record) {
		kv := record.Value.(pipeline.Keyed)
		bs := []byte(kv.Sample.(sample.ByteSlice))

		context, err := extractSample(kv.Key, bs, &m, contextSize, enc, concatByExt)
		if err != nil {
			log.Println("encoding error:", err)
		}
		if len(context) == 0 {
			// This happens if the encoded file is smaller than the context size
			return
		}

		validateSample(context, contextSize, enc)

		sample := utils.Sample{
			Context: context,
			Lang:    enc.LangTagForPath(kv.Key),
		}
		fail(writer.WriteSample(sample))
	}
}

func datasetCycler(filesToSkip int, datasets []*source.Dataset, datagen func(pipeline.Record)) {
	var sources []pipeline.Source
	for _, dataset := range datasets {
		source, err := dataset.ForShard(0, 1)
		fail(err)
		sources = append(sources, source)
	}

	var skipped int
	for {
		for _, source := range sources {
			record := source.SourceOut()
			if record == (pipeline.Record{}) || record.Value == nil {
				log.Printf("skipping strange record: %+v", record)
				continue
			}
			if skipped < filesToSkip {
				skipped++
				continue
			}
			datagen(record)
		}
	}
}

func extractSample(key string, content []byte, m *sync.Mutex, contextSize int, enc *lexicalv0.FileEncoder, concatByExt map[string][]int) ([]int, error) {
	_, fn := utils.SplitKey(key)
	if utils.FilterFile(fn, content) {
		return nil, nil
	}

	toks, err := enc.Lexer.Lex(content)
	if err != nil {
		return nil, err
	}

	idx := rand.Intn(len(toks))

	encoded := encode(enc, key, toks, idx, contextSize)

	if len(encoded) >= contextSize {
		return encoded, nil
	}
	if contextSize < minContextToConcat {
		return nil, nil
	}

	ext := filepath.Ext(fn)
	m.Lock()
	defer m.Unlock()

	concatByExt[ext] = append(concatByExt[ext], encoded...)
	if encoded := concatByExt[ext]; len(encoded) >= contextSize*contextConcatMultiple {
		// +1 since we are getting an endpoint instead of an element
		idx = rand.Intn(1+len(encoded)-contextSize) + contextSize

		// TODO (juan): for now we just use the filename of the last entry
		encoded = enc.PrepareBeforeContext(encoded[:idx], contextSize, key)
		concatByExt[ext] = nil
		return encoded, nil
	}

	return nil, nil
}

// we pass in toks and randIdx to make unit testing easier
func encode(enc *lexicalv0.FileEncoder, filename string, toks []lexer.Token, randIdx, contextSize int) []int {
	var suffix []int
	for i := randIdx; i < len(toks); i++ {
		suffix = append(suffix, enc.EncodeTokens([]lexer.Token{toks[i]})...)
		if len(suffix) >= contextSize {
			return enc.PrepareBeforeContext(suffix, contextSize, filename)
		}
	}

	var prefix []int
	for i := randIdx - 1; i > -1; i-- {
		prefix = append(enc.EncodeTokens([]lexer.Token{toks[i]}), prefix...)
		if len(prefix)+len(suffix) >= contextSize {
			return enc.PrepareBeforeContext(append(prefix, suffix...), contextSize, filename)
		}
	}

	return enc.PrepareBeforeContext(append(prefix, suffix...), contextSize, filename)
}

func validateSample(sample []int, contextSize int, enc *lexicalv0.FileEncoder) {
	if !validate {
		return
	}

	if len(sample) != contextSize {
		log.Fatalf("expected sample to have len %d but got %d\n", contextSize, len(sample))
	}

	for _, id := range sample {
		if id < 0 || id >= enc.Size() {
			log.Fatalf("invalid vocab id %d\n", id)
		}
	}
}
"""
