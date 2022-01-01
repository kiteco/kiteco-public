package manifest

import (
	"encoding/json"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/pkg/errors"
)

// Manifest locates and loads resource data for a given distribution
type Manifest map[keytypes.Distribution]resources.LocatorGroup

// NamedReader represents an io.Reader with an associated name (as in os.File)
type NamedReader interface {
	io.Reader
	Name() string
}

// New loads the manifest data from json
func New(r NamedReader) (Manifest, error) {
	var m Manifest
	err := m.Decode(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode json")
	}

	// attempt to fix any relative paths in the manifest
	absName, err := filepath.Abs(r.Name())
	// don't check the err until we know we're gonna need a name in the loop below
	root := filepath.Dir(absName)
	for _, lg := range m {
		for key := range lg {
			fpath := string(lg[key])
			if strings.HasPrefix(fpath, "s3://") || filepath.IsAbs(fpath) {
				continue
			}
			if !filepath.IsAbs(root) {
				return nil, errors.Wrap(err, "could not deduce root for relative path in manifest")
			}
			lg[key] = resources.Locator(filepath.Join(root, fpath))
		}
	}

	return m, nil
}

// serdesManifest is for encoding/decoding a Manifest into JSON
type serdesManifest struct {
	Dist  keytypes.Distribution
	Paths resources.LocatorGroup
}

// Decode deserializes the manifest from an io.Reader
func (m *Manifest) Decode(r io.Reader) error {
	*m = make(map[keytypes.Distribution]resources.LocatorGroup)
	dec := json.NewDecoder(r)
	for {
		var out serdesManifest
		err := dec.Decode(&out)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		(*m)[out.Dist] = out.Paths
	}
	return nil
}

// Encode serializes the manifest to an io.Writer
func (m *Manifest) Encode(w io.Writer) error {
	dists := m.Distributions()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	for _, dist := range dists {
		paths := (*m)[dist]
		err := enc.Encode(serdesManifest{dist, paths})
		if err != nil {
			return err
		}
	}
	return nil
}

// NumDistributions returns the number of distributions mentioned in the Manifest
func (m Manifest) NumDistributions() int {
	return len(m)
}

// Distributions lists the distribution mentioned in the Manifest
func (m Manifest) Distributions() []keytypes.Distribution {
	dists := make([]keytypes.Distribution, 0, len(m))
	for dist := range m {
		dists = append(dists, dist)
	}
	sort.Sort(keytypes.DistributionList(dists))
	return dists
}

// Load loads a ResourceGroup for a given distribution
func (m Manifest) Load(dist keytypes.Distribution) (*resources.Group, error) {
	lg, exists := m[dist]
	if !exists {
		return nil, errors.New("distribution not found in manifest")
	}

	rg, err := resources.NewGroup(lg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load resource group")
	}

	return rg, nil
}

// Filter filters m to only contain the datafiles corresponding to types
func (m Manifest) Filter(types ...string) Manifest {
	filtered := make(Manifest)
	for dist, lg := range m {
		for _, t := range types {
			if l, ok := lg[t]; ok {
				filtLG := filtered[dist]
				if filtLG == nil {
					filtLG = make(resources.LocatorGroup)
				}
				filtLG[t] = l
				filtered[dist] = filtLG
			}
		}
	}
	return filtered
}

// FilterDistributions filters m to only contain the distributions provided
func (m Manifest) FilterDistributions(dists []keytypes.Distribution) Manifest {
	filtered := make(Manifest)
	toFilter := make(map[keytypes.Distribution]bool)
	for _, dist := range dists {
		toFilter[dist] = true
	}
	for dist, lg := range m {
		if _, ok := toFilter[dist]; ok {
			filtered[dist] = lg
		}
	}
	return filtered
}

// SymbolOnly filters m to contain only symbol graph/info datafiles
func (m Manifest) SymbolOnly() Manifest {
	return m.Filter("SymbolGraph")
}
