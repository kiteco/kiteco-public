package docs

import (
	"compress/gzip"
	"encoding/gob"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// Entity encapsulates html/text python documentation as returned by the API
type Entity struct {
	HTML string
	Text string
}

// Entities contains Entity keyed by canonical symbol
type Entities map[pythonimports.Hash]Entity

// Encode encodes python documentation
func (rs Entities) Encode(w io.Writer) error {
	wd := gzip.NewWriter(w)
	defer wd.Close()

	return gob.NewEncoder(wd).Encode(rs)
}

// Decode decodes python documentation
func (rs Entities) Decode(r io.Reader) error {
	rd, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer rd.Close()

	return gob.NewDecoder(rd).Decode(&rs)
}
