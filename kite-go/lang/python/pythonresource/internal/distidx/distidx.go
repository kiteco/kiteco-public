package distidx

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/pkg/errors"
)

// Index maps top-level (dot-free) package names to Distributions
type Index map[string][]keytypes.Distribution

// New deserializes a new symbol index from an io.Reader
func New(r io.Reader) (Index, error) {
	var out Index
	err := json.NewDecoder(r).Decode(&out)
	if err != nil {
		return nil, err
	}

	// sort and verify per-top-level uniqueness of Distributions
	for tl, dists := range out {
		sort.Sort(keytypes.DistributionList(dists))

		prev := keytypes.Distribution{}
		for _, dist := range dists {
			if dist == prev {
				return nil, errors.Errorf("toplevel %s maps to distribution %s twice", tl, dist)
			}
			prev = dist
		}
	}

	return out, nil
}
