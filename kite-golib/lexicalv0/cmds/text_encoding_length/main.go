package main

import (
	"math/rand"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type encoderBundle struct {
	Vocab string
	*lexicalv0.FileEncoder
}

func main() {
	args := struct {
		Lang     string
		Vocab    string
		Out      string
		NumSites int
		Seed     int64
	}{
		NumSites: 1000,
		Out:      "results.tsv",
		Seed:     42,
	}
	arg.MustParse(&args)

	rand.Seed(args.Seed)

	langGroup := lexicalv0.MustLangGroupFromName(args.Lang)

	gen, err := inspect.NewCodeGenerator(langGroup, false, "")
	fail(err)
	defer gen.Close()

	enc, err := lexicalv0.NewFileEncoder(args.Vocab, langGroup)
	fail(err)

	eb := encoderBundle{
		Vocab:       args.Vocab,
		FileEncoder: enc,
	}

	sampleRates := sampleRates{
		atleast1Ident: .25,
	}

	windows := []int{1}

	computeMetrics(args.Out, sampleRates, gen, eb, args.NumSites, windows)
}
