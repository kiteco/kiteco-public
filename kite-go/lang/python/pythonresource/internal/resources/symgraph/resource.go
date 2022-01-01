package symgraph

import (
	"compress/gzip"
	"io"

	"github.com/tinylib/msgp/msgp"
)

// Encode implements resource.Resource
func (g *Graph) Encode(w io.Writer) error {
	wd := gzip.NewWriter(w)
	defer wd.Close()
	return msgp.Encode(wd, *g)
}

// Decode implements resource.Resource
func (g *Graph) Decode(r io.Reader) error {
	rd, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer rd.Close()
	return msgp.Decode(rd, g)
}
