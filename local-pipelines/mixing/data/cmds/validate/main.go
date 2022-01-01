package main

import (
	"context"
	"errors"
	"log"
	"math/rand"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/local-pipelines/mixing/data/normalize"
)

var config = struct {
	Cursor    string
	TopN      int
	Iters     int
	MaxErrors int
	Seed      int64
	Output    string
}{
	Cursor:    "$",
	TopN:      5,
	Iters:     1000,
	MaxErrors: 10,
	Seed:      1,
	Output:    "data/validate.csv",
}

func main() {
	arg.MustParse(&config)

	resourceManager, models, lexicalmodels, err := normalize.Setup()
	if err != nil {
		log.Fatal(err)
	}
	apiOpts := api.Options{
		ResourceManager: resourceManager,
		Models:          models,
		LexicalModels:   lexicalmodels,
	}
	completer := api.New(context.Background(), apiOpts, licensing.Pro)
	control := api.NewCompleteOptions(data.APIOptions{})
	treatment := api.NewCompleteOptions(data.APIOptions{})

	control.BlockDebug = true
	treatment.BlockDebug = true

	treatment.MixOptions.UseExperimentalScoring = true

	rand.Seed(config.Seed)
	codeGenerator, err := inspect.NewCodeGenerator(lexicalv0.NewLangGroup(lang.Python), false, config.Cursor)
	if err != nil {
		log.Fatal(err)
	}
	defer codeGenerator.Close()
	collector := normalize.NewAPIDataCollector(config.Cursor, config.TopN)

	for sampleID := 0; sampleID < config.Iters; sampleID++ {
		code, _, err := codeGenerator.Next()
		if err != nil {
			log.Fatal(err)
		}
		collector.Collect(completer, control, sampleID, code, "control")
		collector.Collect(completer, treatment, sampleID, code, "treatment")
	}

	if collector.ErrorCount > config.MaxErrors {
		log.Fatal(errors.New("too many collection errors"))
	}
	err = collector.Write(config.Output)
	if err != nil {
		log.Fatal(err)
	}
}
