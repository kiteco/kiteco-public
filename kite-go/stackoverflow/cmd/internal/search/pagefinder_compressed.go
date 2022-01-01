package search

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"

	"github.com/golang/snappy"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
)

const (
	// pagesPerBlock controls block size. Higher number gives better compression, but lower latency
	pagesPerBlock = 50
)

// PageFinderCompressed is an in memory page store for SO pages.
type PageFinderCompressed map[int64][]byte

// Find satisfies the PageFinder interface.
func (pf PageFinderCompressed) Find(id int64) (*stackoverflow.StackOverflowPage, error) {
	block, ok := pf[id]
	if !ok {
		return nil, fmt.Errorf("id %d not found in block map", id)
	}

	decomp := snappy.NewReader(bytes.NewBuffer(block))
	decoder := gob.NewDecoder(decomp)
	for {
		var page stackoverflow.StackOverflowPage
		err := decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("error during block decoding:", err)
			continue
		}
		if id == page.GetQuestion().GetPost().GetId() {
			return &page, nil
		}
	}

	return nil, fmt.Errorf("id %d not found in block", id)
}

// LoadFromPagesDump loads the PageFinderMemory object from a dump
// of so pages (one per line)
func (pf PageFinderCompressed) LoadFromPagesDump(r io.Reader) error {
	var ids []int64
	buf := &bytes.Buffer{}
	comp := snappy.NewWriter(buf)
	encoder := gob.NewEncoder(comp)

	decoder := gob.NewDecoder(r)
	for {
		var page stackoverflow.StackOverflowPage
		err := decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("error reading gob:", err)
			continue
		}

		id := page.GetQuestion().GetPost().GetId()
		if id < 1 {
			continue
		}

		err = encoder.Encode(&page)
		if err != nil {
			log.Fatalln("error encoding page to gob block:", err)
		}

		ids = append(ids, id)
		if len(ids) > pagesPerBlock {
			block := buf.Bytes()
			for _, id := range ids {
				pf[id] = block
			}
			ids = ids[:0]
			buf = &bytes.Buffer{}
			comp = snappy.NewWriter(buf)
			encoder = gob.NewEncoder(comp)
		}
	}
	if len(ids) > 0 {
		block := buf.Bytes()
		for _, id := range ids {
			pf[id] = block
		}
	}
	return nil
}
