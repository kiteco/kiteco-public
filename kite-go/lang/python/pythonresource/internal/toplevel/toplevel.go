package toplevel

import (
	"compress/gzip"
	"encoding/gob"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Entity contains the documentation & symbol counts for a top-level
type Entity struct {
	Docs   *docs.Entity
	Counts *symbolcounts.Entity
}

// DistributionTopLevel pairs a Distribution with a top-level name
type DistributionTopLevel struct {
	Distribution keytypes.Distribution
	TopLevel     string
}

// Entities is a slice of entry objects
type Entities map[DistributionTopLevel]Entity

// Load loads toplevel entries
func Load(fpath string) (Entities, error) {
	r, err := fileutil.NewCachedReader(fpath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	gzR, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzR.Close()

	entries := make(Entities)
	if err := gob.NewDecoder(gzR).Decode(&entries); err != nil {
		return nil, err
	}

	return entries, nil
}
