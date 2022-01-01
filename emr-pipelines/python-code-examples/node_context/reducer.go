package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	contextCount := make(map[string]int)
	var lastKey string

	for r.Next() {
		if r.Key() != lastKey && lastKey != "" {
			model, err := composeModel(lastKey, contextCount)
			if err == nil {
				buf, err := json.Marshal(model)
				if err != nil {
					log.Fatal(err)
				}
				w.Emit(lastKey, buf)
			}
			contextCount = make(map[string]int)
		}
		contextCount[string(r.Value())]++
		lastKey = r.Key()
	}

	if len(contextCount) > 0 {
		model, err := composeModel(lastKey, contextCount)
		if err == nil {
			buf, err := json.Marshal(model)
			if err != nil {
				log.Fatal(err)
			}
			w.Emit(lastKey, buf)
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}

func composeModel(lastKey string, contextCount map[string]int) (*pythonanalyzer.ContextStats, error) {
	var total int
	for _, count := range contextCount {
		total += count
	}

	if total == 0 {
		return nil, fmt.Errorf("total is 0.\n")
	}

	model := &pythonanalyzer.ContextStats{
		Ident: lastKey,
		Prob:  make(map[string]float64),
	}

	for context, count := range contextCount {
		model.Prob[context] = float64(count) / float64(total)
	}
	return model, nil
}
