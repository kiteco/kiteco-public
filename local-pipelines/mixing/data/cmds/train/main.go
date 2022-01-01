package main

import (
	"errors"
	"log"
	"math/rand"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/local-pipelines/mixing/data/normalize"
)

var config = struct {
	Cursor    string
	Limit     int
	Iters     int
	MaxErrors int
	Seed      int64
	Output    string
}{
	Cursor:    "$",
	Limit:     10,
	Iters:     1000,
	MaxErrors: 10,
	Seed:      0,
	Output:    "data/train.csv",
}

func main() {
	arg.MustParse(&config)

	resourceManager, models, lexicalmodels, err := normalize.Setup()
	if err != nil {
		log.Fatal(err)
	}
	global := pythonproviders.Global{
		ResourceManager: resourceManager,
		Models:          models,
		FilePath:        "/sample.py",
		Lexical: lexicalproviders.Global{
			Models:   lexicalmodels,
			FilePath: "/sample.py",
			Product:  licensing.Pro,
		},
		Product: licensing.Pro,
	}
	normalizedProviders := driver.NormalizedProviders()

	rand.Seed(config.Seed)
	codeGenerator, err := inspect.NewCodeGenerator(lexicalv0.NewLangGroup(lang.Python), false, config.Cursor)
	if err != nil {
		log.Fatal(err)
	}
	defer codeGenerator.Close()
	collector := normalize.NewProviderDataCollector(config.Cursor, config.Limit)

	for sampleID := 0; sampleID < config.Iters; sampleID++ {
		code, _, err := codeGenerator.Next()
		if err != nil {
			log.Fatal(err)
		}
		for _, provider := range normalizedProviders {
			collector.Collect(provider, global, code, sampleID)
		}
	}

	if collector.ErrorCount > config.MaxErrors {
		log.Fatal(errors.New("too many collection errors"))
	}
	err = collector.Write(config.Output)
	if err != nil {
		log.Fatal(err)
	}
}
