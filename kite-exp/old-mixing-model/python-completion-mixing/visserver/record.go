package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmixing"
)

type locator struct {
	Hash   string
	Cursor int64
}

type record struct {
	Sample pythonmixing.TrainSample `json:"sample"`
	// Probs contains the softmax probabilities of the completions as inferred by the model
	Probs []float64 `json:"probs"`
}

// Locator which uniquely identifies the record
func (r record) Locator() locator {
	return locator{
		Hash:   r.Sample.Meta.Hash,
		Cursor: r.Sample.Meta.Cursor,
	}
}

type recordIndex struct {
	records []record
	indices map[locator]int
}

func newRecordIndex(filename string) (recordIndex, error) {
	log.Printf("creating record index from %s", filename)

	f, err := os.Open(filename)
	if err != nil {
		return recordIndex{}, err
	}

	r := bufio.NewReader(f)

	var records []record
	for {
		text, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return recordIndex{}, err
		}

		var rec record
		if err = json.NewDecoder(strings.NewReader(text)).Decode(&rec); err != nil {
			return recordIndex{}, err
		}
		records = append(records, rec)
	}

	indices := make(map[locator]int, len(records))
	for i, rec := range records {
		loc := rec.Locator()
		if _, found := indices[loc]; found {
			return recordIndex{}, fmt.Errorf("duplicate record for locator: %+v", loc)
		}
		indices[loc] = i
	}

	log.Printf("%d records read", len(records))

	return recordIndex{
		records: records,
		indices: indices,
	}, nil
}

// Count number of records
func (r recordIndex) Count() int {
	return len(r.records)
}

func (r recordIndex) Get(idx int) record {
	return r.records[idx]
}

func (r recordIndex) Index(loc locator) (int, error) {
	idx, found := r.indices[loc]
	if !found {
		return 0, fmt.Errorf("index not found for locator: %+v", loc)
	}
	return idx, nil
}
