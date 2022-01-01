package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

func main() {
	var output string
	flag.StringVar(&output, "output", "", "json file to write distribution scores to")
	flag.Parse()

	manager, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatalln(err)
	}

	distScores := make(map[keytypes.Distribution]int)
	for _, dist := range pythonresource.DefaultOptions.Manifest.Distributions() {
		if dist == keytypes.BuiltinDistribution3 {
			continue
		}

		toplevels, err := manager.TopLevels(dist)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, tl := range toplevels {
			sym, err := manager.NewSymbol(dist, pythonimports.NewDottedPath(tl))
			if err != nil {
				log.Println(err)
				continue
			}

			counts := manager.SymbolCounts(sym)
			if counts == nil {
				continue
			}

			// Use Import counts to determine score, since we care about how often the package
			// is used, not necessarily how often its members are used.
			distScores[dist] += counts.Import
		}
	}

	type distAndScore struct {
		Distribution keytypes.Distribution
		Score        int
		Rank         int
	}

	var das []distAndScore
	for dist, score := range distScores {
		das = append(das, distAndScore{
			Distribution: dist,
			Score:        score,
		})
	}

	sort.Slice(das, func(i, j int) bool {
		return das[i].Score > das[j].Score
	})

	for idx := range das {
		das[idx].Rank = idx
	}

	buf, err := json.MarshalIndent(das, "", " ")
	if err != nil {
		log.Fatalln(err)
	}

	err = ioutil.WriteFile(output, buf, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
}
