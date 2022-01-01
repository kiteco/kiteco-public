package argspec

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"math/rand"
	"reflect"
	"testing/quick"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// Entity represents a single Python ArgSpec
type Entity = pythonimports.ArgSpec

// Entities indexes Python argspecs
type Entities map[pythonimports.Hash]Entity

// Encode implements resources.Resource
func (d Entities) Encode(w io.Writer) error {
	wd := gzip.NewWriter(w)
	defer wd.Close()

	return gob.NewEncoder(wd).Encode(d)
}

// Decode implements resources.Resource
func (d Entities) Decode(r io.Reader) error {
	rd, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer rd.Close()

	return gob.NewDecoder(rd).Decode(&d)
}

// Generate implements quick.Generator
func (d Entities) Generate(rand *rand.Rand, size int) reflect.Value {
	v, ok := quick.Value(reflect.TypeOf(map[pythonimports.Hash]Entity{}), rand)
	if !ok {
		panic("failed to generate random value")
	}
	res := v.Interface().(map[pythonimports.Hash]Entity)

	// replace empty slices with nil slices because gob can't tell the difference
	// TODO(naman) we do this for several resources; unify using reflection?
	for h, spec := range res {
		if len(spec.Args) == 0 {
			spec.Args = nil
		}

		for i := range spec.Args {
			if len(spec.Args[i].Types) == 0 {
				spec.Args[i].Types = nil
			}
		}

		res[h] = spec
	}
	if len(res) == 0 {
		res = nil
	}

	return reflect.ValueOf(res)
}
