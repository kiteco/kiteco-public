package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/localtraining"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

// TODO:
//   - for python we should use a variant of the "select files" logic we use in `pythonbatch` so
//     we handle libraries in a reasonable way?
//   - for other languages it is unclear where this should live.
func getFiles(group lexicalv0.LangGroup, rootDir string) []string {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Errorf("error iterating file %s in %s with error: %s", path, rootDir, err)
		}
		if info.IsDir() {
			return nil
		}
		// 128kb filter
		if info.Size() > (1 << 17) {
			return nil
		}

		if !group.Contains(lang.FromFilename(path)) {
			return nil
		}

		// Go filters
		if strings.HasSuffix(path, "bindata.go") || strings.HasSuffix(path, "pb.go") || strings.Contains(path, "vendor") {
			return nil
		}
		// JS Filters
		if strings.Contains(path, "node_modules") || strings.HasSuffix(path, ".min.js") {
			return nil
		}
		// Python Filters
		if strings.Contains(path, "kite_ml/env") || strings.Contains(path, "site-packages") || strings.Contains(path, "dist-packages") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	fail(err)

	sort.Strings(files)

	return files
}

func main() {
	args := struct {
		TrainRatio        float64
		ValidateRatio     float64
		SplitType         localtraining.SplitType
		Seed              int64
		NumGo             int
		VocabInit         localtraining.VocabInit
		WeightedSampling  bool
		TrainSampleRate   float64
		TrainBatchSize    int
		ValidateBatchSize int
		StepsPerFile      int

		MaxTrainIters int
		VocabIters    int
		LocalDataRoot string
		Language      string
		GlobalModel   string

		OutVocabIdents     string
		OutVocabEmbeddings string
		OutValidateSamples string
		OutTrainSamples    string
	}{
		TrainRatio:        0.95,
		ValidateRatio:     0.05,
		SplitType:         localtraining.RandomSplit,
		Seed:              42,
		NumGo:             runtime.NumCPU(),
		VocabInit:         localtraining.AverageParents,
		WeightedSampling:  true,
		TrainSampleRate:   0.5,
		TrainBatchSize:    20,
		ValidateBatchSize: 100,
		StepsPerFile:      500,
	}
	arg.MustParse(&args)

	group := lexicalv0.MustLangGroupFromName(args.Language)

	params := localtraining.Params{
		TrainRatio:       args.TrainRatio,
		ValidateRatio:    args.ValidateRatio,
		SplitType:        args.SplitType,
		NumGo:            args.NumGo,
		VocabIters:       args.VocabIters,
		VocabInit:        args.VocabInit,
		WeightedSampling: args.WeightedSampling,
		TrainSampleRate:  args.TrainSampleRate,
	}

	files := getFiles(group, args.LocalDataRoot)

	log.Printf("training on %d files\n", len(files))
	if len(files) == 0 {
		log.Fatalf("no files to train on")
	}

	in := localtraining.Inputs{
		Language:        group,
		Seed:            args.Seed,
		Files:           files,
		GlobalModelPath: args.GlobalModel,
		ContextSize:     contextSizeForModel(args.GlobalModel),
	}

	trainer, err := localtraining.NewTrainer(params, in)
	fail(err)

	res, err := trainer.Train(kitectx.Background())
	fail(err)

	outVocab, err := os.Create(args.OutVocabIdents)
	fail(err)
	defer outVocab.Close()

	// TODO: kind of nasty, but works for this binary
	_, err = bpe.NewBuilderFromEncoder(res.NewEncoder.BPE).WriteTo(outVocab)
	fail(err)

	writeToJSON(res.NewVocab.NewEmbeddings, args.OutVocabEmbeddings)

	trainSamples, validationSamples := res.TrainSamples, res.ValidateSamples

	// If we end up with no validation samples, just re-split the training samples. This
	// happens when the number of files is so low that the validation sample rate ends up pulling
	// only 1 or 2 files. If we get unlucky with those files, we may end up with no samples
	// This happens when attempting to tune on requests-oauthlib, for example (22 python files)
	//
	// TODO(tarak): What we should do is generate all samples, then split them based on train/validate/test
	// ratios instead of splitting by files. That works for our large training runs but can cause edge cases here.
	// We can do this if we are OK with removing the LastModifiedTimeSplit option.
	if len(validationSamples) == 0 && len(trainSamples) != 0 {
		splitIdx := int(float64(len(res.TrainSamples)) * params.ValidateRatio)
		validationSamples = trainSamples[:splitIdx]
		trainSamples = trainSamples[splitIdx:]
	}

	log.Printf("training samples: %d, validation samples: %d", len(trainSamples), len(validationSamples))

	if len(trainSamples) == 0 && len(validationSamples) == 0 {
		log.Println("not enough training/validation samples, exiting")
		os.Exit(1)
	}

	maxNumSteps := 2 * args.MaxTrainIters

	trainWriter := utils.NewSampleWriter(args.OutTrainSamples, "", args.TrainBatchSize, args.StepsPerFile)
	defer trainWriter.Flush()

	// TODO: kind of nasty, we cycle through data multiple
	// times if needed to get enough samples
	for trainWriter.StepsWritten() < maxNumSteps {
		for _, s := range trainSamples {
			sample := utils.Sample{
				Lang:    s.LangTag,
				Context: s.Sample,
			}
			fail(trainWriter.WriteSample(sample))
			if trainWriter.StepsWritten() >= maxNumSteps {
				break
			}
		}
	}

	validateWriter := utils.NewSampleWriter(args.OutValidateSamples, "", args.ValidateBatchSize, args.StepsPerFile)
	defer validateWriter.Flush()

	for validateWriter.StepsWritten() < maxNumSteps {
		for _, s := range validationSamples {
			sample := utils.Sample{
				Lang:    s.LangTag,
				Context: s.Sample,
			}
			fail(validateWriter.WriteSample(sample))
			if validateWriter.StepsWritten() >= maxNumSteps {
				break
			}
		}
	}
}

func writeToJSON(content interface{}, filename string) {
	buf, err := json.Marshal(content)
	fail(err)

	fail(ioutil.WriteFile(filename, buf, os.ModePerm))
}

func contextSizeForModel(path string) int {
	hp, err := predict.NewHParams(fileutil.Join(path, "config.json"))
	fail(err)
	return hp.ContextSize
}
