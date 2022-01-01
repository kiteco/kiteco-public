// produces a table of data for analyzing min p tuning.
// looks at the distribution of probabilities over correct predictions.
// for example, setting min p to the value of the 10th percentile
// corresponds to eliminating 10 percent of correct predictions.

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/minp"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

func main() {
	arg.MustParse(&minp.Args)

	language := lexicalv0.MustLangGroupFromName(minp.Args.Language)

	modelOptions, err := lexicalmodels.GetDefaultModelOptions(language)
	if err != nil {
		log.Fatal(err)
	}

	config, err := predict.NewSearchConfigFromModelPath(modelOptions.ModelPath)
	if err != nil {
		log.Fatal(err)
	}

	// Set these to no-op values for tuning
	config.MinP = 0.0
	config.PrefixRegularization = 1.0

	d, err := minp.Collect(modelOptions.ModelPath, config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("hit rate:")
	cols := []string{""}
	for depth := 1; depth <= minp.Args.MaxDepth; depth++ {
		cols = append(cols, fmt.Sprintf("%d", depth))
	}

	fmt.Println(strings.Join(cols, ","))
	for pct := 1; pct <= 25; pct++ {
		cols := []string{fmt.Sprintf("%.3f", float64(pct)/100)}
		for depth := 1; depth <= minp.Args.MaxDepth; depth++ {
			if len(d[depth]) == 0 {
				cols = append(cols, "na")
				continue
			}
			value := minp.GetPercentile(d[depth], float64(pct)/100)
			cols = append(cols, fmt.Sprintf("%.5f", value))
		}
		fmt.Println(strings.Join(cols, ","))
	}

	fmt.Println("\nsample size:")
	for depth := 1; depth <= minp.Args.MaxDepth; depth++ {
		fmt.Printf("%d,%d\n", depth, len(d[depth]))
	}
}
