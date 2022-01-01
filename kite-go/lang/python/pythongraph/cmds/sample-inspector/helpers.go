package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/dgryski/go-spooky"

	"github.com/kiteco/kiteco/kite-go/lang/python"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
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

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (w worker) getInputs(src string) (pythongraph.Inputs, error) {
	bSrc := []byte(src)
	words, err := pythonscanner.Lex(bSrc, scanOpts)

	if err != nil {
		return pythongraph.Inputs{}, fmt.Errorf("unable to lex file: %v", err)
	}

	ast, err := pythonparser.ParseWords(kitectx.Background(), bSrc, words, parseOpts)
	if ast == nil {
		return pythongraph.Inputs{}, fmt.Errorf("unable to parse ast: %v", err)
	}

	rast, err := pythonanalyzer.NewResolver(w.RM, pythonanalyzer.Options{
		Path: "/src.py",
	}).Resolve(ast)

	if err != nil {
		return pythongraph.Inputs{}, fmt.Errorf("unable to resolve ast: %v", err)
	}

	return pythongraph.Inputs{
		RM:     w.RM,
		RAST:   rast,
		Words:  words,
		Buffer: bSrc,
	}, nil
}

type srcCursor struct {
	Src    string
	Cursor int
}

func newSrcCursor(s string) (srcCursor, error) {
	parts := strings.Split(s, "$")
	switch len(parts) {
	case 1, 2:
		return srcCursor{
			Src:    strings.Join(parts, ""),
			Cursor: len(parts[0]),
		}, nil
	default:
		return srcCursor{}, fmt.Errorf("input src may contain 0 or 1 cursor ($), got: %d", len(parts)-1)
	}
}

func (w worker) defaultTrainParams(sc srcCursor, saver *saver) pythongraph.TrainParams {
	// hacky way to get different tasks instances based on the cursor position
	contents := strings.Join([]string{
		sc.Src[:sc.Cursor],
		"$",
		sc.Src[sc.Cursor:],
	}, "")

	seed := spooky.Hash64([]byte(contents))
	return pythongraph.TrainParams{
		ModelMeta: pythongraph.ModelMeta{
			NameSubtokenIndex: w.MI.NameSubtokenIndex,
			TypeSubtokenIndex: w.MI.TypeSubtokenIndex,
			ProductionIndex:   w.MI.ProductionIndex,
		},
		Rand:  rand.New(rand.NewSource(int64(seed))),
		Saver: saver,
	}
}

func anySym(rm pythonresource.Manager, val pythontype.Value) pythonresource.Symbol {
	for _, s := range python.GetExternalSymbols(kitectx.Background(), rm, val) {
		return s
	}
	return pythonresource.Symbol{}
}
