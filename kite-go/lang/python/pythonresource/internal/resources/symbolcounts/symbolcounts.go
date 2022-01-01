package symbolcounts

import (
	"compress/gzip"
	"encoding/json"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
)

// Entity is the counts for a given symbol
type Entity = symbolcounts.Counts

// Entities is a map of counts for each symbol
type Entities map[string]Entity

// serdes bundles a symbol name with its counts for (de)serialization
type serdes struct {
	Symbol string
	Data   symbolcounts.Counts
}

// Encode implements resources.Resource
func (rs Entities) Encode(w io.Writer) error {
	wd := gzip.NewWriter(w)
	defer wd.Close()

	enc := json.NewEncoder(wd)
	for sym, dat := range rs {
		err := enc.Encode(serdes{sym, dat})
		if err != nil {
			return err
		}
	}
	return nil
}

// Decode implements resources.Resource
func (rs Entities) Decode(r io.Reader) error {
	rd, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer rd.Close()

	dec := json.NewDecoder(rd)
	for {
		var out serdes
		err := dec.Decode(&out)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		rs[out.Symbol] = out.Data
	}
	return nil
}
