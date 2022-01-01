package main

import (
	"fmt"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

const (
	absoluteEpsilon = 0.1 // absoluteEpsilon is the maximal absolute difference between a old and a new value, any bigger diff will trigger a panic
	relativeEpsilon = 1   // relativeEpsilon is the same but for relative differences
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		OutputPath string
		Language   string
	}{
		OutputPath: "",
		Language:   "javascript",
	}
	arg.MustParse(&args)

	fail(datadeps.Enable())
	l := lexicalv0.MustLangGroupFromName(args.Language)
	modelOptions, err := lexicalmodels.GetDefaultModelOptions(l)
	fail(err)

	baseline, err := predict.NewSearchConfigFromModelPath(modelOptions.ModelPath)
	fail(err)
	configPath := "searchconfig.json"
	if args.OutputPath != "" {
		configPath = filepath.Join(args.OutputPath, configPath)
	}
	newConfig, err := predict.NewSearchConfig(configPath)
	fail(err)

	fail(compareConfig(baseline, newConfig))
}

func compareNumber(baseline, newValue float32, field string) error {
	if baseline-newValue > absoluteEpsilon || newValue-baseline > absoluteEpsilon {
		return errors.Errorf("The absolute difference for %s is too big between the baseline (%v) and the value obtained in the test run (%v)", field, baseline, newValue)
	}
	if baseline == 0 {
		return errors.Errorf("The baseline for %s is 0", field)
	}
	if newValue == 0 {
		return errors.Errorf("The new value for %s is 0", field)
	}
	if (baseline-newValue)/newValue > relativeEpsilon || (newValue-baseline)/baseline > relativeEpsilon {
		return errors.Errorf("The relative difference for %s is too big between the baseline (%v) and the value obtained in the test run (%v)", field, baseline, newValue)
	}

	fmt.Printf("For %s baseline value = %v new value - %v\n", field, baseline, newValue)
	return nil
}

func compareConfig(baseline predict.SearchConfig, newConfig predict.SearchConfig) error {
	var errAcc error
	if err := compareNumber(baseline.MinP, newConfig.MinP, "MinP"); err != nil {
		errAcc = errors.Combine(errAcc, err)
	}
	if err := compareNumber(baseline.IdentTemperature, newConfig.IdentTemperature, "IdentTemperature"); err != nil {
		errAcc = errors.Combine(errAcc, err)
	}
	if err := compareNumber(baseline.LexicalTemperature, newConfig.LexicalTemperature, "LexicalTemperature"); err != nil {
		errAcc = errors.Combine(errAcc, err)
	}

	return errAcc
}
