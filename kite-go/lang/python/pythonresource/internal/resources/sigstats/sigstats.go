package sigstats

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// TypeInfo contains stats about the types used for an argument (5 most frequent types)
type TypeInfo struct {
	Path  string
	Dist  keytypes.Distribution
	Count int
}

// GetSymKey return a string that can be used as a key to represent this symbol
func (t TypeInfo) GetSymKey() string {
	return fmt.Sprintf("%s_%s", t.Dist.String(), t.Path)
}

// ArgStat contains stats about argument usage (positional or keyword)
type ArgStat struct {
	Name  string
	Count int
	Types map[pythonimports.Hash]TypeInfo
}

// Entity contains statistics for a given symbol's signature patterns
type Entity struct {
	Positional []ArgStat
	ArgsByName map[string]ArgStat
	Count      int
}

// Entities indexes kwargs by symbol
type Entities map[pythonimports.Hash]Entity

// Encode implements resources.Resource
func (e Entities) Encode(w io.Writer) error {
	wd := gzip.NewWriter(w)
	defer wd.Close()

	return gob.NewEncoder(wd).Encode(e)
}

// Decode implements resources.Resource
func (e Entities) Decode(r io.Reader) error {
	rd, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer rd.Close()

	return gob.NewDecoder(rd).Decode(&e)
}
