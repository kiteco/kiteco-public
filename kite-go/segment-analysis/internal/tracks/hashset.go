package tracks

import (
	"compress/gzip"
	"encoding/gob"
	"log"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// LoadContentHashSet loads the content hash set at the provided path.
func LoadContentHashSet(path string) (map[string]bool, error) {
	var hashes map[string]bool
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gunzip, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	err = gob.NewDecoder(gunzip).Decode(&hashes)
	if err != nil {
		return nil, err
	}

	log.Printf("loaded %d content hashes", len(hashes))
	return hashes, nil
}
