//go:generate go-bindata -pkg distidx -prefix ../.. ../../index.json

package distidx

import "bytes"

// KiteIndex is the global Kite distribution index
var KiteIndex Index

func init() {
	i, err := New(bytes.NewReader(MustAsset("index.json")))
	if err != nil {
		panic(err)
	}

	KiteIndex = i
}
