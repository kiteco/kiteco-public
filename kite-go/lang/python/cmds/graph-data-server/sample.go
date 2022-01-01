package main

import (
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	scanOpts = pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}

	parseOpts = pythonparser.Options{
		Approximate: true,
		ErrorMode:   pythonparser.Recover,
	}
)

const maxFileBytes = 30000
const maxSampleBuildDuration = 100 * time.Millisecond

// sampleSeed represents the necessary inputs to deterministically compute a sample.
type sampleSeed struct {
	// Symbol as given by the user
	Symbol pythonresource.Symbol
	// Hash for the source to try
	Hash string
	// deterministic pseudo-random integer that determines how the sample is created
	Random int
}

// sample represents a train/test sample returned to the client.
type sample struct {
	Data trainData `json:"data"`
}

type trainData struct {
	Expr *pythongraph.ExprTrainSample `json:"expr"`
}

// global resources needed to build graphs
type resources struct {
	rm    pythonresource.Manager
	store *codeStore
}

func getInputs(src []byte, res *resources) (pythongraph.Inputs, error) {
	defer getInputsDuration.DeferRecord(time.Now())

	words, err := pythonscanner.Lex(src, scanOpts)

	if err != nil {
		return pythongraph.Inputs{}, fmt.Errorf("unable to lex file: %v", err)
	}

	ast, err := pythonparser.ParseWords(kitectx.Background(), src, words, parseOpts)
	if ast == nil {
		return pythongraph.Inputs{}, fmt.Errorf("unable to parse ast: %v", err)
	}

	rast, err := pythonanalyzer.NewResolver(res.rm, pythonanalyzer.Options{
		Path: "/src.py",
	}).Resolve(ast)

	if err != nil {
		return pythongraph.Inputs{}, fmt.Errorf("unable to resolve ast: %v", err)
	}

	return pythongraph.Inputs{
		RM:     res.rm,
		RAST:   rast,
		Words:  words,
		Buffer: src,
	}, nil
}
