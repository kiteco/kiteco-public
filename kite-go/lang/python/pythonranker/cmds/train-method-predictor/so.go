package main

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// load stackoverflow data
func loadSOData(path string, data map[string]map[string]*pythonranker.MethodTrainingData) {
	in, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	decomp, err := gzip.NewReader(in)
	if err != nil {
		log.Fatal(err)
	}

	sodata := make(map[string]map[string]*pythonranker.MethodTrainingData)

	decoder := json.NewDecoder(decomp)
	err = decoder.Decode(&sodata)
	if err != nil {
		log.Fatal(err)
	}

	for p, packageData := range sodata {
		if _, exists := data[p]; !exists {
			log.Println(p, "doesn't exist in doc corpus. Skipping.")
		}
		for m, mdata := range packageData {
			if _, exists := data[p][m]; !exists {
				continue
			}
			input := strings.Join(mdata.Data, " ")
			data[p][m].Data = append(data[p][m].Data, processor.Apply(tokenizer(input))...)
		}
	}
}
