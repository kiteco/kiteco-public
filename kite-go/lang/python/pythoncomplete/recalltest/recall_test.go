// +build slow

package recalltest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	datasetPath = "samples_3347.json"
	testFolder  = "kite-go/lang/python/pythoncomplete/recalltest/"
)

func TestCompareRecall(t *testing.T) {
	t.SkipNow()
	recalls := tryLoadRecalls()
	maxSamples := 2000
	var api *api.API
	if recalls == nil {
		api = buildCompletionAPI()
	}

	t.Run("Compare Recalls for GGNN model", func(t *testing.T) {
		computeAndCompareRecall(t, true, api, recalls, maxSamples)
	})

	t.Run("Compare Recalls for Call model", func(t *testing.T) {
		computeAndCompareRecall(t, false, api, recalls, maxSamples)
	})
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		fail(errors.Wrapf(err, "Error while opening the file %s that contains expected result for recall", filename))
	}
	return !info.IsDir()
}

func tryLoadRecalls() map[string]RecallsInfo {
	basedir := "/tmp"
	if os.Getenv("TRAVIS_BUILD_DIR") != "" {
		basedir = os.Getenv("TRAVIS_BUILD_DIR")
	}

	filename := fmt.Sprintf("%s/recalls_%s.json", basedir, time.Now().Format("2006_01_02"))
	if fileExists(filename) {
		file, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Println("Can't read recalls file, recomputing them")
			return nil
		}

		var result map[string]RecallsInfo
		err = json.Unmarshal(file, &result)
		if err != nil {
			fmt.Println("Error while unmarshalling json of Recalls file, recomputing them")
			return nil
		}
		fmt.Printf("Precomputed recalls file found in %s, using these for comparison.\nPrecomputed recalls:\n%s\n", filename, result)
		return result
	}
	return nil
}

func computeAndCompareRecall(t *testing.T, useGGNN bool, api *api.API, precomputedRecalls map[string]RecallsInfo, maxSamples int) {
	model := "Call Model"
	baselineFilepath := "callmodel_baseline.json"
	precomputedRecallsKey := "CallModel"
	if useGGNN {
		baselineFilepath = "ggnn_baseline.json"
		model = "GGNN Model"
		precomputedRecallsKey = "GGNNModel"
	}
	var baseline RecallsInfo
	file, err := ioutil.ReadFile(baselineFilepath)
	require.NoError(t, err, "Error while reading the baseline file for the recall of the %s", model)

	err = json.Unmarshal(file, &baseline)
	require.NoError(t, err, "Error while unmarshalling json of baseline Recalls for %s", model)

	require.NotEmpty(t, baseline.ExactMatch.Recalls, "You should have at least 1 value for baseline recall for exact match for the %s", model)
	require.NotEmpty(t, baseline.PlaceholderMatch.Recalls, "You should have at least 1 value for baseline recall for placeholder match for the %s", model)

	var errors []string
	if precomputedRecalls == nil {
		errors = computeAndCompareRecalls(datasetPath, baseline, useGGNN, api, maxSamples)
	} else {

		recalls := precomputedRecalls[precomputedRecallsKey]
		errors = compareRecalls(baseline, recalls.ExactMatch.Recalls, recalls.PlaceholderMatch.Recalls)
	}
	if len(errors) > 0 {
		errorsStr := strings.Join(errors, "\n")
		fullFilepath := path.Join(testFolder, baselineFilepath)
		assert.Failf(t, fmt.Sprint("Recall change too much for ", model), "%s\nPlease fix the models or update the file %s with the following content:\n%s", errorsStr, fullFilepath, getNewBaselineContent(baseline))
	}
}

func getNewBaselineContent(baseline RecallsInfo) string {
	result, err := json.MarshalIndent(baseline, "", "  ")
	fail(err)
	return string(result)
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
