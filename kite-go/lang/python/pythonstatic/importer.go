package pythonstatic

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Importer for packages and modules.
type Importer struct {
	// PythonPaths is an ordered list of import "root"s to consider for the purposes of importing
	PythonPaths map[string]struct{}
	// Path is the location of the current file in the user's filesystem.
	Path string
	// Global is the global symbol graph
	Global pythonresource.Manager
	// Local is the local symbol tree.
	Local *pythonenv.SourceTree
}

// ImportAbs finds the value for the given top level package
// e.g `kite` in `import kite.foo`.
func (i Importer) ImportAbs(ctx kitectx.Context, pkg string) (pythontype.Value, bool) {
	ctx.CheckAbort()

	var vals []pythontype.Value
	if i.Local != nil {
		if sym := i.Local.ImportAbs(i.Path, pkg); sym != nil {
			vals = append(vals, sym.Value)
		}
	}

	for path := range i.PythonPaths {
		// TODO(naman) i.Local.ImportRooted(path, pkg)?
		if sym := i.Local.ImportAbs(path, pkg); sym != nil {
			vals = append(vals, sym.Value)
		}
	}

	if i.Global != nil {
		pkgPath := pythonimports.NewDottedPath(pkg)
		for _, dist := range i.Global.DistsForPkg(pkg) {
			if sym, err := i.Global.NewSymbol(dist, pkgPath); err == nil {
				vals = append(vals, pythontype.TranslateExternal(sym, i.Global))
			}
		}
	}

	if len(vals) == 0 {
		return nil, false
	}
	return pythontype.Unite(ctx, vals...), true
}

// ImportRel finds the value corresponding to the given sequence of dots in a relative
// import such as "from ...foo import bar". SRCPATH should be a path to the python source
func (i Importer) ImportRel(dots int) (pythontype.Value, bool) {
	if i.Local != nil {
		if pkg := i.Local.ImportRel(i.Path, dots); pkg != nil {
			return pkg, true
		}
	}
	return nil, false
}

// Navigate finds a node in the global symbol graph
func (i Importer) Navigate(path pythonimports.DottedPath) (pythonresource.Symbol, error) {
	return i.Global.PathSymbol(path)
}
