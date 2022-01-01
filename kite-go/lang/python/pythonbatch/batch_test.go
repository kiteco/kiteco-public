package pythonbatch

import (
	"io/ioutil"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	uid = 3
	mid = "machine-id"
)

func mustReadFile(path string) []byte {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return buf
}

func managerAdd(t testing.TB, m *BatchManager, path, hash string, contents []byte) {
	opts := m.opts.PathSelection.Parse
	opts.ScanOptions.Label = path
	mod, _ := pythonparser.Parse(kitectx.Background(), contents, opts)
	require.NotNil(t, mod)
	m.Add(&SourceUnit{
		ASTBundle: pythonstatic.ASTBundle{AST: mod, Path: path, Imports: pythonstatic.FindImports(kitectx.Background(), path, mod)},
		Contents:  contents,
		Hash:      hash,
	})
}

func TestDefinitions(t *testing.T) {
	graph := pythonresource.MockManager(t, nil)
	bi := BatchInputs{
		User:    uid,
		Machine: mid,
		Graph:   graph,
	}
	manager := NewBatchManager(kitectx.Background(), bi, DefaultOptions, nil)
	managerAdd(t, manager, "/kite.py", "hash", mustReadFile("testdata/kite.py"))
	managerAdd(t, manager, "/man.py", "hash", mustReadFile("testdata/man.py"))

	results, err := manager.Build(kitectx.Background())
	require.NoError(t, err)

	t.Log("definitions:")
	for key := range results.Definitions {
		name := requireLocatorFromHash(t, key, manager.delegate.Exprs)
		t.Log(name)
	}

	names := []string{
		locator("/kite.py", "Kite"),
		locator("/kite.py", "Kite.foo"),
		locator("/kite.py", "Kite.__init__"),
		locator("/man.py", "Man"),
		locator("/man.py", "Man.car"),
		locator("/man.py", "Man.__init__"),
		locator("/man.py", "print_code"),
		locator("/man.py", "some_kite"),
	}

	// here we use require.Equal to avoid
	// having the logs print out the entire structs
	// if this fails
	require.Equal(t, len(names), len(results.Definitions), "expected %d defs, got %d", len(names), len(results.Definitions))

	for _, name := range names {
		val := requireValueFromLocator(t, name, manager.delegate.Exprs)
		_, found := results.Definitions[pythonlocal.LookupID(val)]
		assert.True(t, found, "missing definition for %s\n", name)
	}
}

func requireValueFromLocator(t *testing.T, loc string, vals map[pythonast.Expr]pythontype.Value) pythontype.Value {
	var val pythontype.Value
	for _, v := range vals {
		if pythonenv.Locator(v) == loc {
			val = v
			break
		}
	}
	require.NotNil(t, val, "unable to find value from locator `%s`", loc)
	return val
}

func requireLocatorFromHash(t *testing.T, hash string, vals map[pythonast.Expr]pythontype.Value) string {
	var loc string
	for _, val := range vals {
		if val == nil {
			continue
		}

		if hash == pythonlocal.LookupID(val) {
			loc = pythonenv.Locator(val)
			break
		}
	}
	require.NotEmpty(t, loc, "unable to find locator for hash %d", hash)
	return loc
}

func locator(file, path string) string {
	return pythonenv.LocatorForAddress(pythontype.Address{
		User:    uid,
		Machine: mid,
		File:    file,
		Path:    pythonimports.NewDottedPath(path),
	})
}

func TestNameOverload(t *testing.T) {
	graph := pythonresource.MockManager(t, nil)
	bi := BatchInputs{
		Graph: graph,
	}
	manager := NewBatchManager(kitectx.Background(), bi, DefaultOptions, nil)
	managerAdd(t, manager, "/testdata/__init__.py", "hash", mustReadFile("testdata/__init__.py"))
	managerAdd(t, manager, "/testdata/kite.py", "hash", mustReadFile("testdata/kite.py"))

	results, err := manager.Build(kitectx.Background())
	require.NoError(t, err)

	for key, mod := range results.Assembly.Sources.Files {
		t.Log(key)
		for attr, sym := range mod.Members.Table {
			t.Logf("  .%s -> %v", attr, sym.Value)
		}
	}

	pkg := results.Assembly.Sources.Dirs["/testdata"]
	require.NotNil(t, pkg)

	res, _ := pythontype.AttrNoCtx(pkg, "kite")
	require.NotNil(t, res.Value())
	kite := res.Value()
	require.NotNil(t, kite)
	require.NotNil(t, kite)
	assert.Equal(t, kite.Kind(), pythontype.FunctionKind)
}

func TestPackageStructure(t *testing.T) {
	graph := pythonresource.MockManager(t, nil)
	bi := BatchInputs{
		Graph: graph,
	}
	manager := NewBatchManager(kitectx.Background(), bi, DefaultOptions, nil)
	managerAdd(t, manager, "/code/lib/x.py", "hash", []byte("import y"))
	managerAdd(t, manager, "/code/lib/y.py", "hash", []byte("value = 123"))
	managerAdd(t, manager, "/code/cmds/burried/deep.py", "hash", []byte("from lib import y"))
	managerAdd(t, manager, "/code/cmds/burried/__init__.py", "hash", []byte("foo = 'bar'"))

	results, err := manager.Build(kitectx.Background())
	require.NoError(t, err)

	code := results.Assembly.Sources.Dirs["/code"]
	lib := results.Assembly.Sources.Dirs["/code/lib"]
	x := results.Assembly.Sources.Files["/code/lib/x.py"]
	y := results.Assembly.Sources.Files["/code/lib/y.py"]
	cmds := results.Assembly.Sources.Dirs["/code/cmds"]
	burried := results.Assembly.Sources.Dirs["/code/cmds/burried"]
	deep := results.Assembly.Sources.Files["/code/cmds/burried/deep.py"]

	require.NotNil(t, code)
	require.NotNil(t, lib)
	require.NotNil(t, x)
	require.NotNil(t, y)
	require.NotNil(t, cmds)
	require.NotNil(t, burried)
	require.NotNil(t, deep)

	// do not use assert.Equal here becase we want to test pointer equality
	assert.True(t, lib == code.DirEntries.Table["lib"].Value)
	assert.True(t, cmds == code.DirEntries.Table["cmds"].Value)
	assert.True(t, x == lib.DirEntries.Table["x"].Value)
	assert.True(t, y == lib.DirEntries.Table["y"].Value)
	assert.True(t, burried == cmds.DirEntries.Table["burried"].Value)
	assert.True(t, deep == burried.DirEntries.Table["deep"].Value)
	assert.True(t, y == x.Members.Table["y"].Value)
	assert.True(t, y == deep.Members.Table["y"].Value)

	assert.NotNil(t, y.Members.Table["value"])
	assert.NotNil(t, burried.Init.Members.Table["foo"])
}
