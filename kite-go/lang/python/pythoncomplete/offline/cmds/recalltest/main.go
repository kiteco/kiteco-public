package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/recalltest"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	log.SetPrefix("")
	log.SetFlags(0)

	if len(os.Args) != 3 && len(os.Args) != 5 || len(os.Args) == 5 && os.Args[3] != "--json" {
		fmt.Println("usage: recalltest sample_file max_samples [--json output.json] # Pass 0 for max samples to use all samples")
		os.Exit(1)
	}
	filename := os.Args[1]
	maxSamples, err := strconv.Atoi(os.Args[2])
	fail(err)
	recallsData := make(map[string]recalltest.RecallsInfo, 2)
	api := buildCompletionAPI()

	ggnnRecalls := computeRecalls(filename, api, maxSamples, true)
	callModelRecalls := computeRecalls(filename, api, maxSamples, false)

	recallsData["GGNNModel"] = ggnnRecalls
	recallsData["CallModel"] = callModelRecalls

	var jsonOutputFile string
	if len(os.Args) == 5 {
		jsonOutputFile = os.Args[4]
	}

	// print to file, if set on the cmdline
	if jsonOutputFile != "" {
		jsonBytes, err := json.MarshalIndent(recallsData, "", "  ")
		fail(err)
		err = ioutil.WriteFile(jsonOutputFile, jsonBytes, 0600)
		fail(err)
	}

	fmt.Printf("GGNN recalls:\n%s\n\nCall Model recalls:\n%s\n", ggnnRecalls.String(), callModelRecalls.String())

}

func computeRecalls(sampleFile string, api *api.API, maxSamples int, useGGNN bool) recalltest.RecallsInfo {
	start := time.Now()
	completionMatches := recalltest.ComputeMatchingCompletions(sampleFile, "", false, useGGNN, api, maxSamples)
	exactRecalls, placeholderRecalls := recalltest.ComputeBothRecalls(completionMatches, 5)
	return recalltest.RecallsInfo{
		ExactMatch:       recalltest.RecallSet{Recalls: exactRecalls},
		PlaceholderMatch: recalltest.RecallSet{Recalls: placeholderRecalls},
		Timestamp:        time.Now(),
		Duration:         time.Now().Sub(start),
	}

}

func buildCompletionAPI() *api.API {
	fail(datadeps.Enable())
	tensorflow.SetTensorflowThreadpoolSize(4)
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)
	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	fail(err)

	completionAPITemp := api.New(context.Background(), api.Options{
		ResourceManager: rm,
		Models:          models,
	}, licensing.Pro)
	return &completionAPITemp
}
