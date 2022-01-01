//go:generate go-bindata -pkg distranking -prefix ../.. ../../distscores.json

package distranking

import "bytes"

// DefaultRanking is the global rankings for Kite's distributions
var DefaultRanking Ranking

func init() {
	r, err := New(bytes.NewReader(MustAsset("distscores.json")))
	if err != nil {
		panic(err)
	}

	DefaultRanking = r
}
