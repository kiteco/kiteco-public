package search

import (
	"encoding/json"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/bufutil"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// PageFinderLevelDB encapsulates the functionality of fetching
// StackOverFlowPages from a document store.
type PageFinderLevelDB struct {
	db *leveldb.DB
}

// NewPageFinderLevelDB initializes a new level db page finder or returns an error.
func NewPageFinderLevelDB(dbfile string) (*PageFinderLevelDB, error) {
	db, err := leveldb.OpenFile(dbfile, &opt.Options{
		ReadOnly:               true,
		OpenFilesCacheCapacity: 100, // hack for running locally
	})
	if err != nil {
		return nil, err
	}
	return &PageFinderLevelDB{
		db: db,
	}, nil
}

// Find retrieves the page with the specified id if it exists, otherwise returns nil and error.
func (pf *PageFinderLevelDB) Find(id int64) (*stackoverflow.StackOverflowPage, error) {
	key := bufutil.IntToBytes(id)
	val, err := pf.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	var page stackoverflow.StackOverflowPage
	err = json.Unmarshal(val, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}
