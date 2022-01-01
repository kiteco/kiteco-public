package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondiffs"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	counts := make(map[pythondiffs.DecoratorStyle]int)

	for r.Next() {
		var decorator pythondiffs.DecoratorStyle
		err := json.Unmarshal(r.Value(), &decorator)
		if err != nil {
			continue
		}
		if r.Key() != lastKey && len(counts) > 0 {
			var total int
			for _, c := range counts {
				total += c
			}
			// Compute the probability of each style
			var styles []pythondiffs.DecoratorStyleProb
			for dec, c := range counts {
				styles = append(styles, pythondiffs.DecoratorStyleProb{
					Literal:     dec.Literal,
					LiteralRoot: dec.LiteralRoot,
					Prob:        float64(c) / float64(total),
				})
			}
			out, err := json.Marshal(styles)
			if err == nil {
				w.Emit(lastKey, out)
			}
			counts = make(map[pythondiffs.DecoratorStyle]int)
		}
		counts[decorator]++
		lastKey = r.Key()
	}

	if r.Key() != lastKey && len(counts) > 0 {
		var total int
		for _, c := range counts {
			total += c
		}
		// Compute the probability of each style
		var styles []pythondiffs.DecoratorStyleProb
		for dec, c := range counts {
			styles = append(styles, pythondiffs.DecoratorStyleProb{
				Literal:     dec.Literal,
				LiteralRoot: dec.LiteralRoot,
				Prob:        float64(c) / float64(total),
			})
		}
		out, err := json.Marshal(styles)
		if err != nil {
			log.Println(err)
		} else {
			w.Emit(lastKey, out)
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}
