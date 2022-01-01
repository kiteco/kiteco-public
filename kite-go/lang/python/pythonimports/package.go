package pythonimports

import "github.com/kiteco/kiteco/kite-golib/serialization"

// Package represents the set of packages that were imported as a result of importing
// a single package.
type Package struct {
	Name         string
	Dependencies []string
}

// LoadDependencies loads information about dependencies between packages. A package X is
// said to depend on Y if, during import exploration, importing X caused Y to be imported.
func LoadDependencies(path string) (map[string]*Package, error) {
	pkgs := make(map[string]*Package)
	err := serialization.Decode(path, func(p *Package) {
		pkgs[p.Name] = p
	})
	return pkgs, err
}
