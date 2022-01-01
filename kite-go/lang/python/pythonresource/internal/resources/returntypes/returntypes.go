package returntypes

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"math/rand"
	"reflect"
	"testing/quick"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// Entities represents the return types of functions & methods for a given distribution
// it maps path hashes to slices of path strings
type Entities map[uint64]Entity

// Entity is a set of path strings representing the return types for a given symbol
type Entity = map[string]keytypes.Truthiness

// Encode encodes Entities
func (rs Entities) Encode(w io.Writer) error {
	wd := gzip.NewWriter(w)
	defer wd.Close()

	return gob.NewEncoder(wd).Encode(rs)
}

// Decode decodes Entities
func (rs Entities) Decode(r io.Reader) error {
	rd, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer rd.Close()

	return gob.NewDecoder(rd).Decode(&rs)
}

// Generate implements quick.Generator
func (rs Entities) Generate(rand *rand.Rand, size int) reflect.Value {
	v, ok := quick.Value(reflect.TypeOf((map[uint64]map[string]keytypes.Truthiness)(nil)), rand)
	if !ok {
		panic("failed to generate random value")
	}
	res := v.Interface().(map[uint64]map[string]keytypes.Truthiness)

	// replace empty slices with nil slices because gob can't tell the difference
	// TODO(naman) we do this for several resources; unify using reflection?
	for h := range res {
		if len(res[h]) == 0 {
			res[h] = nil
		}
	}
	if len(res) == 0 {
		res = nil
	}

	return reflect.ValueOf(res)
}
