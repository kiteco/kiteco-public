package pythonindex

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
)

type index struct {
	useStemmer    bool
	invertedIndex map[string][]*IdentCount
	// TODO(naman) unused: rm unless we decide to turn local code search back on
	diskIndex *diskmapIndex
}

func (i *index) find(s string) ([]*IdentCount, bool) {
	if i.invertedIndex != nil {
		if cnts, exists := i.invertedIndex[s]; exists {
			return cnts, exists
		}
	}
	if i.diskIndex != nil {
		cnts, err := i.diskIndex.find(s)
		return cnts, err == nil
	}
	return nil, false
}

type diskmapIndex struct {
	index *diskmap.Map
	cache *lru.Cache
}

func (d *diskmapIndex) find(s string) ([]*IdentCount, error) {
	if d.cache != nil {
		if cnts, ok := d.cache.Get(s); ok {
			return cnts.([]*IdentCount), nil
		}
	}
	var cnts []*IdentCount
	err := diskmap.JSON.Get(d.index, s, &cnts)
	if err == nil {
		d.cache.Add(s, cnts)
		return cnts, nil
	}
	return nil, fmt.Errorf("could not find identifier")
}
