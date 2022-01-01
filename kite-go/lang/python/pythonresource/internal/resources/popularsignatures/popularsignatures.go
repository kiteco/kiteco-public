package popularsignatures

import (
	"compress/gzip"
	"io"
	"math/rand"
	"reflect"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/tinylib/msgp/msgp"
)

// Encode implements resources.Resource
func (rs Entities) Encode(w io.Writer) error {
	wd := gzip.NewWriter(w)
	defer wd.Close()

	return msgp.Encode(wd, rs)
}

// Decode implements resources.Resource
func (rs Entities) Decode(r io.Reader) error {
	rd, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer rd.Close()

	return msgp.Decode(rd, &rs)
}

// Generate implements quick.Generator for generating random Entities for testing
func (rs Entities) Generate(rand *rand.Rand, size int) reflect.Value {
	e := make(Entities)
	for i := 0; i < size; i++ {
		key := pythonimports.Hash(rand.Int63())
		e[key] = append(e[key], &Signature{})
	}

	return reflect.ValueOf(e)
}
