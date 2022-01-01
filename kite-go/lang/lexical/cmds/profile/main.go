package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

const cursor = "$"

type arguments struct {
	Batches    int
	BatchSize  int
	Language   string
	Local      bool
	MinLines   int
	CPUProfile string
	MemProfile string
}

var (
	args = arguments{
		Batches:   10,
		BatchSize: 1,
		Language:  "",
		Local:     false,
		MinLines:  0,
	}
	language lexicalv0.LangGroup
)

type sample struct {
	Path string
	Code string
}

func getSamples() [][]sample {
	codeGenerator, err := inspect.NewCodeGenerator(language, args.Local, cursor)
	if err != nil {
		log.Fatal(err)
	}
	defer codeGenerator.Close()

	var batches [][]sample
	for len(batches) < args.Batches+1 {
		code, path, err := codeGenerator.Next()
		if err != nil {
			log.Fatal(err)
		}
		code = strings.Split(code, cursor)[0]
		if strings.Count(code, "\n") < args.MinLines {
			continue
		}
		if args.BatchSize == 1 {
			batches = append(batches, []sample{{
				Path: path,
				Code: code,
			}})
			continue
		}
		if len(code) <= args.BatchSize {
			continue
		}
		var batch []sample
		j := rand.Intn(len(code) - args.BatchSize)
		for k := 0; k < args.BatchSize; k++ {
			batch = append(batch, sample{
				Path: path,
				Code: code[:j+k],
			})
		}
		batches = append(batches, batch)
	}
	return batches
}

func main() {
	arg.MustParse(&args)

	language = lexicalv0.MustLangGroupFromName(args.Language)
	batches := getSamples()

	go http.ListenAndServe(":6060", nil)
	rand.Seed(time.Now().UnixNano())

	tensorflow.SetTensorflowThreadpoolSize(1)

	var err error
	var cpuw io.WriteCloser
	var once sync.Once
	if args.CPUProfile != "" {
		cpuw, err = os.Create(args.CPUProfile)
		if err != nil {
			log.Fatalln(err)
		}
		defer cpuw.Close()
	}

	models, err := lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions)
	if err != nil {
		log.Fatal(err)
	}

	completer := api.New(context.Background(), api.Options{Models: models}, licensing.Pro)
	opts := api.CompleteOptions{
		BlockDebug: true,
	}
	var start time.Time
	for i, batch := range batches {
		if i == 1 {
			fmt.Println("warmed up!")
			start = time.Now()
			if args.CPUProfile != "" {
				once.Do(func() {
					pprof.StartCPUProfile(cpuw)
				})
			}
		}
		for _, sample := range batch {
			buf := data.NewBuffer(sample.Code)
			sel := data.Cursor(len(sample.Code))
			request := data.APIRequest{
				UMF: data.UMF{
					Filename: sample.Path,
				},
				SelectedBuffer: buf.Select(sel),
			}
			completer.Complete(kitectx.Background(), opts, request, nil, nil)
		}
	}

	fmt.Println(time.Since(start))
	show("EmbedInitialContext", status.Stats.Percentiles(), status.EmbedInitialContextDuration.Values(), showDuration)
	show("PartialRunQuery", status.Stats.Percentiles(), status.PartialRunQueryDuration.Values(), showDuration)
	show("PartialRunOverlapDist", status.Stats.Percentiles(), status.PartialRunOverlapDist.Values(), showInt)
	show("CloseState", status.Stats.Percentiles(), status.ClosePartialRunDuration.Values(), showDuration)
	show("NewPredictState", status.Stats.Percentiles(), status.NewPredictStateDuration.Values(), showDuration)
	show("Search", status.Stats.Percentiles(), status.SearchDuration.Values(), showDuration)
	show("Prettify", status.Stats.Percentiles(), status.PrettifyDuration.Values(), showDuration)
	show("FormatCompletion", status.Stats.Percentiles(), status.FormatCompletionDuration.Values(), showDuration)
	show("FormatBytes", status.Stats.Percentiles(), status.FormatBytes.Values(), showBytes)
	show("Completion API", api.Stats.Percentiles(), api.CompletionDuration.Values(), showDuration)
	fmt.Printf("\nPRM reuse rate: %.1f%%\n", status.PartialRunReuseRate.Value())
	fmt.Printf("Prediction reuse rate: %.1f%%\n", status.PredictionReuseRate.Value())

	if args.CPUProfile != "" {
		pprof.StopCPUProfile()
	}

	if args.MemProfile != "" {
		memw, err := os.Create(args.MemProfile)
		if err != nil {
			log.Fatalln(err)
		}
		defer memw.Close()

		if err := pprof.WriteHeapProfile(memw); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

type showType string

var (
	showDuration showType = "duration"
	showBytes    showType = "bytes"
	showInt      showType = "int"
)

func show(label string, percentiles []float64, values []int64, t showType) {
	zeros := make([]int64, len(values))
	if reflect.DeepEqual(values, zeros) {
		fmt.Printf("=== %s: no values, skipping\n\n", label)
		return
	}
	fmt.Printf("=== %s\n", label)
	for i, percentile := range percentiles {
		switch t {
		case showBytes:
			fmt.Printf("%dth percentile: %v\n", int(100*percentile), humanize.Bytes(uint64(values[i])))
		case showDuration:
			fmt.Printf("%dth percentile: %v\n", int(100*percentile), time.Duration(values[i]))
		case showInt:
			fmt.Printf("%dth percentile: %v\n", int(100*percentile), values[i])
		}
	}
	fmt.Println()
}
