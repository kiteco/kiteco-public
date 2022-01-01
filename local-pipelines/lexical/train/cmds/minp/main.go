package main

import (
	"encoding/json"
	"log"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/minp"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

func main() {
	args := struct {
		InSearch   string
		OutSearch  string
		Depth      int
		Percentile float64
		ModelPath  string
		Language   string
	}{
		Depth:      1,
		Percentile: 0.125,
	}
	arg.MustParse(&args)
	minp.Args.Language = args.Language

	config, err := predict.NewSearchConfig(args.InSearch)
	if err != nil {
		log.Fatal(err)
	}

	// Changes these values to correctly compute minp
	origPrefixRegularization := config.PrefixRegularization

	config.MinP = 0.0
	config.PrefixRegularization = 1.0

	d, err := minp.Collect(args.ModelPath, config)
	if err != nil {
		log.Fatal(err)
	}
	config.MinP = minp.GetPercentile(d[args.Depth], args.Percentile)
	config.PrefixRegularization = origPrefixRegularization

	f, err := fileutil.NewBufferedWriter(args.OutSearch)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(config)
	if err != nil {
		log.Fatal(err)
	}
}
