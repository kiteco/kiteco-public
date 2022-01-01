package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		InputFile string `arg:"positional,required"`
	}{}
	arg.MustParse(&args)

	maybeQuit(datadeps.Enable())
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	maybeQuit(<-errc)

	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	maybeQuit(err)

	b, err := ioutil.ReadFile(args.InputFile)
	maybeQuit(err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}

	key := data.UMF{
		Filename: args.InputFile,
	}
	req := data.APIRequest{
		SelectedBuffer: sb,
		UMF:            key,
	}
	opts := api.IDCCCompleteOptions
	opts.BlockDebug = true
	a := api.New(context.Background(), api.Options{
		ResourceManager: rm,
		Models:          models,
	}, licensing.Pro)

	var resp data.APIResponse
	err = kitectx.Background().WithTimeout(5*time.Second, func(ctx kitectx.Context) error {
		resp = a.Complete(ctx, opts, req, nil, nil)
		return resp.ToError()
	})
	maybeQuit(err)

	// dump completions
	completionsFile := strings.Replace(args.InputFile, "inputs", "completion-fixtures", -1)
	completionsFile = strings.Replace(completionsFile, ".py", ".json", -1)
	outputFile, err := os.Create(completionsFile)
	maybeQuit(err)
	defer outputFile.Close()
	enc := json.NewEncoder(outputFile)
	enc.SetIndent("", "    ")
	err = enc.Encode(resp)
	maybeQuit(err)

	// dump scheduler cache
	fixture, err := a.CreateSchedulerFixture(key)
	maybeQuit(err)
	cacheFile := strings.Replace(args.InputFile, "inputs", "cache-fixtures", -1)
	cacheFile = strings.Replace(cacheFile, ".py", ".json", -1)
	outputFile, err = os.Create(cacheFile)
	maybeQuit(err)
	defer outputFile.Close()
	enc = json.NewEncoder(outputFile)
	enc.SetIndent("", "    ")
	err = enc.Encode(fixture)
	maybeQuit(err)
}
