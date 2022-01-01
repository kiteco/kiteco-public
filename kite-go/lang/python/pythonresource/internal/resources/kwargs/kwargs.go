package kwargs

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"math/rand"
	"reflect"
	"testing/quick"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// KeywordArgs contains keyword arguments for the named function
type KeywordArgs struct {
	Name   string
	Kwargs []KeywordArg
}

// KeywordArg represents a single keyword argument, with possible types
type KeywordArg struct {
	Name  string
	Types []string
}

// Entities indexes kwargs by symbol
type Entities map[pythonimports.Hash]KeywordArgs

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

// Generate implements quick.Generator
func (e Entities) Generate(rand *rand.Rand, size int) reflect.Value {
	v, ok := quick.Value(reflect.TypeOf(map[pythonimports.Hash]KeywordArgs{}), rand)
	if !ok {
		panic("failed to generate random value")
	}
	res := v.Interface().(map[pythonimports.Hash]KeywordArgs)

	// replace empty slices with nil slices because gob can't tell the difference
	// TODO(naman) we do this for several resources; unify using reflection?
	for h, kwargs := range res {
		if len(kwargs.Kwargs) == 0 {
			kwargs.Kwargs = nil
		}

		for i := range kwargs.Kwargs {
			if len(kwargs.Kwargs[i].Types) == 0 {
				kwargs.Kwargs[i].Types = nil
			}
		}

		res[h] = kwargs
	}
	if len(res) == 0 {
		res = nil
	}

	return reflect.ValueOf(res)
}
