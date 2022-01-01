package search

import (
	"encoding/gob"
	"errors"
	"io"
	"log"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
)

// PageFinderMemory is an in memory page store for SO pages.
type PageFinderMemory map[int64]*stackoverflow.StackOverflowPage

// NewPageFinderMemory returns an initialized, in memory, PageFinder.
func NewPageFinderMemory(capacity int) PageFinderMemory {
	return make(PageFinderMemory, capacity)
}

// Find satisfies the PageFinder interface.
func (pf PageFinderMemory) Find(id int64) (*stackoverflow.StackOverflowPage, error) {
	if page, found := pf[id]; found {
		return page, nil
	}
	return nil, errors.New("page not found with id: " + strconv.FormatInt(id, 10))
}

// Add adds page to the store with key id.
func (pf PageFinderMemory) Add(id int64, page *stackoverflow.StackOverflowPage) error {
	pf[id] = page
	return nil
}

// Len returns the number of pages currently in the PageFinder.
func (pf PageFinderMemory) Len() int {
	return len(pf)
}

// LoadFromPagesDump loads the PageFinderMemory object from a dump
// of so pages (one per line)
func (pf PageFinderMemory) LoadFromPagesDump(r io.Reader) error {
	decoder := gob.NewDecoder(r)
	for {
		var page stackoverflow.StackOverflowPage
		err := decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err.Error())
			continue
		}
		id := page.GetQuestion().GetPost().GetId()
		if id < 1 {
			continue
		}
		pf[id] = &page
	}
	return nil
}
