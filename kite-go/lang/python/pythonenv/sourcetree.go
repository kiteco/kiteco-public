package pythonenv

import (
	"fmt"
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// SourceTree contains Modules for each module and package
type SourceTree struct {
	Files map[string]*pythontype.SourceModule  // Files maps filenames to modules
	Dirs  map[string]*pythontype.SourcePackage // Dirs maps directory paths to packages
}

// NewSourceTree construct an empty source tree
func NewSourceTree() *SourceTree {
	return &SourceTree{
		Files: make(map[string]*pythontype.SourceModule),
		Dirs:  make(map[string]*pythontype.SourcePackage),
	}
}

// AddFile inserts a new module into the source tree, which also
// creates parent directories as required
func (t *SourceTree) AddFile(srcpath string, mod *pythontype.SourceModule, windows bool) {
	parent := t.AddDir(path.Dir(srcpath), mod.Members.Name.User, mod.Members.Name.Machine, windows)
	base := path.Base(srcpath)
	if base == "__init__.py" {
		parent.Init = mod
	} else {
		attr := strings.TrimSuffix(base, ".py")
		sym := parent.DirEntries.Create(attr)
		sym.Value = mod
	}

	t.Files[srcpath] = mod
}

// AddDir inserts a new package into the source tree
func (t *SourceTree) AddDir(srcpath string, uid int64, mid string, windows bool) *pythontype.SourcePackage {
	// get or create package for this dir
	pkg, found := t.Dirs[srcpath]
	if !found {
		name := pythontype.Address{User: uid, Machine: mid, File: srcpath}
		// this symbol table has no parent because packages do not inherit any symbols from
		// any "parent" scope.
		pkg = &pythontype.SourcePackage{LowerCase: windows, DirEntries: pythontype.NewSymbolTable(name, nil)}
		t.Dirs[srcpath] = pkg
	}

	// create an attribute on the parent package that points to this package
	if path.Dir(srcpath) != srcpath {
		parent := t.AddDir(path.Dir(srcpath), uid, mid, windows)
		attr := strings.TrimSuffix(path.Base(srcpath), ".py")
		sym := parent.DirEntries.LocalOrCreate(attr)
		sym.Value = pkg
	}

	return pkg
}

// ImportAbs finds the node for the given top-level package (e.g. "kite" in "import kite.foo.bar")
// SRCPATH should be a path to the python source file that contained the import statement.
func (t *SourceTree) ImportAbs(srcpath string, name string) *pythontype.Symbol {
	var found *pythontype.Symbol
	t.srcPkgSearch(srcpath, func(pkg *pythontype.SourcePackage) bool {
		if sym, ok := pkg.DirAttr(name); ok {
			found = sym
			return false // stop
		}
		return true
	})
	return found
}

// srcPkgSearch calls callback once for each package in the (approximate) absolute import search path for srcpath,
// which may be a file or a directory.
// Earlier calls to callback take priority over later ones (i.e. the packages are emitted in order for search).
// If callback return false, no more packages are emitted.
func (t *SourceTree) srcPkgSearch(srcpath string, callback func(*pythontype.SourcePackage) bool) {
	if !path.IsAbs(srcpath) {
		panic(fmt.Sprintf("SourceTree received non-absolute path: `%s`", srcpath))
	}

	for {
		if pkg, exists := t.Dirs[srcpath]; exists {
			if !callback(pkg) {
				break
			}
		}
		if parent := path.Dir(srcpath); parent != srcpath {
			srcpath = parent
		} else {
			break
		}
	}
}

// ListAbs returns a list (map from name to pythontype.Symbol) of packages importable via ImportAbs
func (t *SourceTree) ListAbs(srcpath string) map[string]*pythontype.Symbol {
	listing := make(map[string]*pythontype.Symbol)
	t.srcPkgSearch(srcpath, func(pkg *pythontype.SourcePackage) bool {
		for name, sym := range pkg.DirEntries.Table {
			if _, ok := listing[name]; !ok {
				listing[name] = sym
			}
		}
		return true
	})
	return listing
}

// ImportRel finds the node corresponding to the given sequence of dots in a relative
// import such as "from ...foo import bar". SRCPATH should be a path to the python source
// file that contained the import statement.
func (t *SourceTree) ImportRel(srcpath string, dots int) *pythontype.SourcePackage {
	if !path.IsAbs(srcpath) {
		panic(fmt.Sprintf("SourceTree received non-absolute path: %s", srcpath))
	}
	for i := 0; i < dots; i++ {
		srcpath = path.Dir(srcpath)
	}
	return t.Dirs[srcpath]
}

// Flatten creates a flat representation of this source tree suitable for serialization
func (t *SourceTree) Flatten(ctx kitectx.Context) (*FlatSourceTree, error) {
	ctx.CheckAbort()

	var vs []pythontype.Value
	for _, mod := range t.Files {
		if mod != nil {
			vs = append(vs, mod)
		}
	}
	for _, pkg := range t.Dirs {
		if pkg != nil {
			vs = append(vs, pkg)
		}
	}

	flat, err := pythontype.FlattenValues(ctx, vs)
	if err != nil {
		return nil, err
	}

	ft := FlatSourceTree{
		Values: flat,
	}
	for path, mod := range t.Files {
		hsh, err := pythontype.Hash(ctx, mod)
		if err != nil {
			return nil, err
		}
		ft.Files = append(ft.Files, FlatItem{
			Name:    path,
			ValueID: hsh,
		})
	}
	for path, pkg := range t.Dirs {
		hsh, err := pythontype.Hash(ctx, pkg)
		if err != nil {
			return nil, err
		}
		ft.Dirs = append(ft.Dirs, FlatItem{
			Name:    path,
			ValueID: hsh,
		})
	}
	return &ft, nil
}

// Package returns the parent package that contains the specified path.
// If the path is to a module this returns the package for the directory
// containing the module, if the path is to a directory this returns the package
// for the parent directory.
func (t *SourceTree) Package(p string) (*pythontype.SourcePackage, error) {
	pkg := t.Dirs[path.Dir(p)]
	if pkg == nil {
		return nil, fmt.Errorf("unable to find package for path `%s`", p)
	}
	return pkg, nil
}

// Locate finds a value in the source tree from a string locator.
// It uses the file name to find the SourceModule and then iteratively
// searches down the attribute path to find the value. This function
// can return an error if the input string is not a locator or if
// no value could be found.
func (t *SourceTree) Locate(ctx kitectx.Context, loc string) (pythontype.Value, error) {
	ctx.CheckAbort()
	addr, err := ParseValueLocator(loc)
	if err != nil {
		return nil, err
	}
	val, err := t.FindValue(ctx, addr.File, addr.Path.Parts)
	if err == nil || strings.HasSuffix(addr.File, ".py") {
		return val, err
	}
	return t.FindValue(ctx, addr.File+".py", addr.Path.Parts)
}

// LocateSymbol finds the value corresponding to a symbol from a string
// locator. It uses the method FindValue to first find the namespace and then
// uses the symbol name to find the appropriate value. It returns the value,
// the namespace of the value and the name of the symbol. This function
// can return an error if the input string is not a locator or if no value
// could be found.
func (t *SourceTree) LocateSymbol(ctx kitectx.Context, loc string) (pythontype.Value, pythontype.Value, string, error) {
	ctx.CheckAbort()
	addr, name, err := ParseSymbolLocator(loc)
	if err != nil {
		return nil, nil, "", err
	}
	ns, err := t.FindValue(ctx, addr.File, addr.Path.Parts)
	if err != nil {
		return nil, nil, "", err
	}

	ns, attr, err := t.FindSymbol(ctx, addr.File, addr.Path.Parts, name)
	if err != nil {
		return nil, nil, "", err
	}
	return ns, attr, name, nil
}

// FindSymbol finds the symbol associated with the filename, path, and attr.
func (t *SourceTree) FindSymbol(ctx kitectx.Context, file string, path []string, attr string) (pythontype.Value, pythontype.Value, error) {
	ns, err := t.FindValue(ctx, file, path)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to find value %s %s: %v", file, strings.Join(path, "."), err)
	}

	av := findAttr(ctx, ns, attr, 0)
	if av == nil {
		return nil, nil, fmt.Errorf("unable to find attr %s on namespace %v", attr, ns)
	}

	return ns, av, nil
}

// FindValue finds the value associated with the filename and path.
// It is used by by Locate and LocateSymbol.
func (t *SourceTree) FindValue(ctx kitectx.Context, file string, path []string) (pythontype.Value, error) {
	ctx.CheckAbort()

	var val pythontype.Value
	if !strings.HasSuffix(file, ".py") {
		// Lookup package by directory name
		pkg, exists := t.Dirs[file]
		if !exists {
			return nil, fmt.Errorf("package does not exist for %s", file)
		}
		val = pkg
	} else {
		mod, exists := t.Files[file]
		if !exists {
			return nil, fmt.Errorf("module does not exist for %s", file)
		}
		val = mod
	}
	for _, name := range path {
		attr := findAttr(ctx, val, name, 0)
		if attr == nil {
			return nil, fmt.Errorf("could not find value for attr %s of %v", name, val)
		}
		val = attr
	}
	return val, nil
}

func findAttr(ctx kitectx.Context, val pythontype.Value, name string, steps int) pythontype.Value {
	var res []pythontype.Value
	for _, val := range pythontype.Disjuncts(ctx, val) {
		attr, _ := pythontype.Attr(ctx, val, name)
		if attr.Found() {
			res = append(res, attr.Value())
		}

		// SourceFunctions also store information about locally defined variables so
		// treat these as "attributes" as well.
		if val, ok := val.(*pythontype.SourceFunction); ok {
			if sym, exists := val.Locals.Table[name]; exists {
				res = append(res, sym.Value)
			}
		}
	}
	return pythontype.Unite(ctx, res...)
}
