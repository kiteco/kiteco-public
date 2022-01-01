//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"

	assetfs "github.com/elazarl/go-bindata-assetfs"

	"github.com/kiteco/kiteco/kite-golib/templateset"

	callprobutils "github.com/kiteco/kiteco/local-pipelines/python-call-filtering/python-call-prob/call-prob-utils"

	"github.com/kiteco/kiteco/kite-golib/tensorflow"
	"github.com/kiteco/kiteco/local-pipelines/python-call-filtering/internal/utils"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

const (
	maxFileSize    = 50000
	predictTimeout = 10 * time.Second
)

const defaultSnippet = `
import pandas as pd

my_df = pd.read_csv("test.csv")

my_df.from_records("test")
`

var (
	datasetPath = pythoncode.DedupedCodeDumpPath
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type app struct {
	templates *templateset.Set
	res       utils.Resources
}

func toHTML(s interface{}) template.HTML {
	return template.HTML(fmt.Sprintf("%v", s))
}

func main() {
	datadeps.Enable()
	args := struct {
		MaxFiles             int
		NumTensorflowThreads int
		Port                 string
		Endpoints            []string
		ExprShards           string
	}{
		MaxFiles:             10,
		NumTensorflowThreads: 8,
		Port:                 ":5678",
	}
	tensorflow.SetTensorflowThreadpoolSize(args.NumTensorflowThreads)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	modelOpts := pythonmodels.DefaultOptions
	if args.ExprShards != "" {
		shards, err := pythonexpr.ShardsFromFile(args.ExprShards)
		fail(err)
		modelOpts.ExprModelShards = shards
	}

	expr, err := pythonexpr.NewShardedModel(context.Background(), modelOpts.ExprModelShards, modelOpts.ExprModelOpts)
	if err != nil {
		panic(err)
	}

	models := pythonmodels.Models{
		Expr: expr,
	}

	res := utils.Resources{RM: rm, Models: &models}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	theApp := app{
		templates: templateset.NewSet(staticfs, "templates", template.FuncMap{"toHTML": toHTML}),
		res:       res,
	}
	simPipeline(res)

	r := mux.NewRouter()
	r.HandleFunc("/", theApp.handleRequest)
	r.HandleFunc("/favicon.ico", faviconHandler)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	log.Printf("listening on http://localhost%s\n", args.Port)
	fail(http.ListenAndServe(args.Port, neg))
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func inSlice(s []int, i int) bool {
	for _, ii := range s {
		if i == ii {
			return true
		}
	}
	return false
}

func getRootData(samples []callprobutils.InspectableSample) RootData {
	var result []SampleInfo
	for _, s := range samples {
		var completions []CompletionInfo
		for i, c := range s.Sample.Meta.CompIdentifiers {
			features := s.Sample.Features.Comp[i]
			weights := features.Weights()
			label := 0
			if inSlice(s.Sample.Labels, i) {
				label = 1
			}
			weights = append(weights, s.Sample.Features.Contextual.Weights()...)
			completions = append(completions, CompletionInfo{
				Completion: c,
				Label:      label,
				Features:   weights,
			})
		}
		sort.Slice(completions, func(i, j int) bool {
			if completions[i].Label != completions[j].Label {
				return completions[i].Label > completions[j].Label
			}
			return completions[i].Completion < completions[j].Completion
		})

		result = append(result, SampleInfo{
			Source:      s.Source,
			UserTyped:   s.UserTyped,
			Completions: completions,
			Truncated:   s.Truncated,
		})
	}
	return RootData{Samples: result}
}

func (a app) handleRequest(w http.ResponseWriter, r *http.Request) {

	var data RootData
	switch r.Method {
	case "GET":
		inputTxt := defaultSnippet

		result, err := callprobutils.SimulatePipeline(inputTxt, a.res)
		if err != nil {
			fmt.Println("Error: ", err)
			panic(err)
		} else {
			fmt.Println(result)
		}
		data = getRootData(result)
	}
	err := a.templates.Render(w, "root.html", data)
	if err != nil {
		panic(err)
	}
}

func simPipeline(res utils.Resources) {
	inputTxt := defaultSnippet

	result, _ := callprobutils.SimulatePipeline(inputTxt, res)
	print(result)
}
