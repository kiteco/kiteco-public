package tests

import (
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	uid = 33
	mid = "machine-id"
)

func testFile(name string) string {
	return filepath.Join("./testdata", name)
}

func requireSourceTree(t *testing.T, manager pythonresource.Manager, files map[string]string) *pythonenv.SourceTree {
	st, err := pythontest.SourceTree(uid, mid, manager, files)
	require.NoError(t, err)

	// verify some basic properties of the source tree
	for file := range files {
		assert.NotNil(t, st.Files[file], "unable to find module for file '%s'", file)
		for dir := path.Dir(file); dir != path.Dir(file); dir = path.Dir(file) {
			assert.NotNil(t, st.Dirs[dir], "unable to find package for dir '%s'", dir)
		}

		// check package structure
		val, err := st.FindValue(kitectx.Background(), "/", nil)
		if err != nil {
			t.Errorf("unable to find root value: %v\n", err)
			continue
		}

		if val == nil {
			t.Errorf("got nil root value")
			continue
		}

		_, ok := val.(*pythontype.SourcePackage)
		if !ok {
			t.Errorf("expected root to be a source package but got type %T", val)
			continue
		}

		// ensure that for each file we have properly linked
		// all packages and modules
		parts := strings.Split(file, "/")
		for i, part := range parts {
			if part == "" {
				continue
			}
			res, err := pythontype.AttrNoCtx(val, strings.TrimSuffix(part, ".py"))
			if err != nil {
				t.Errorf("unable to find attr %s on %v", part, val)
				break
			}

			val = res.Value()
			if i == len(parts)-1 {
				// end in a module
				assert.IsType(t, &pythontype.SourceModule{}, val)
			} else {
				assert.IsType(t, &pythontype.SourcePackage{}, val)
			}
		}
	}

	return st
}

func requireValue(t *testing.T, st *pythonenv.SourceTree, addr pythontype.Address) pythontype.Value {
	addr.User = uid
	addr.Machine = mid
	val, err := st.Locate(kitectx.Background(), pythonenv.LocatorForAddress(addr))
	require.NoError(t, err)
	require.NotNil(t, val)
	return val
}

func requireSymbol(t *testing.T, st *pythonenv.SourceTree, nsAddr pythontype.Address, attr string) (pythontype.Value, pythontype.Value) {
	nsAddr.User = uid
	nsAddr.Machine = mid
	nsVal := requireValue(t, st, nsAddr)
	nsValTemp, attrVal, attrTemp, err := st.LocateSymbol(kitectx.Background(), pythonenv.SymbolLocator(nsVal, attr))
	require.NoError(t, err)
	require.NotNil(t, nsValTemp)
	require.NotNil(t, attrVal)
	require.Equal(t, attr, attrTemp, 0)
	require.True(t, pythontype.EqualNoCtx(nsVal, nsValTemp))
	return nsVal, nsValTemp
}

func TestClassSymbol(t *testing.T) {
	files := map[string]string{
		"/src/server.py": testFile("server.py"),
	}

	st := requireSourceTree(t, pythonresource.MockManager(t, nil), files)

	requireSymbol(t, st, pythontype.Address{File: "/src/server.py"}, "Server")
}

func TestClassMembers(t *testing.T) {
	files := map[string]string{
		"/src/server.py": testFile("server.py"),
		"/src/run.py":    testFile("run.py"),
	}

	st := requireSourceTree(t, pythonresource.MockManager(t, nil), files)

	addr := pythontype.Address{File: "/src/server.py"}

	requireSymbol(t, st, addr, "Server")

	addr.Path = pythonimports.NewDottedPath("Server")
	requireSymbol(t, st, addr, "start")

	requireSymbol(t, st, addr, "__init__")

	requireSymbol(t, st, addr, "running")

	requireSymbol(t, st, addr, "port")
}

func TestFunctionLocalVariable(t *testing.T) {
	files := map[string]string{
		"/src/server.py": testFile("server.py"),
		"/src/run.py":    testFile("run.py"),
	}

	st := requireSourceTree(t, pythonresource.MockManager(t, nil), files)

	addr := pythontype.Address{File: "/src/run.py"}

	requireSymbol(t, st, addr, "start")

	addr.Path = pythonimports.NewDottedPath("start")

	requireSymbol(t, st, addr, "newServer")

	requireSymbol(t, st, addr, "server")

	addr.Path = pythonimports.NewDottedPath("start.newServer")

	requireSymbol(t, st, addr, "s")
}

func TestPackage(t *testing.T) {
	files := map[string]string{
		"/src/server.py": testFile("server.py"),
		"/src/run.py":    testFile("run.py"),
	}

	st := requireSourceTree(t, pythonresource.MockManager(t, nil), files)

	pkg, err := st.Package("/")
	assert.NoError(t, err)
	assert.Equal(t, pkg, st.Dirs["/"], 0)
	pkg, err = st.Package("/src")
	assert.NoError(t, err)
	assert.Equal(t, pkg, st.Dirs["/"], 0)

	src, err := st.Package("/src/server.py")
	assert.NoError(t, err)
	assert.Equal(t, src, st.Dirs["/src"], 0)

	src, err = st.Package("/src/run.py")
	assert.NoError(t, err)
	assert.Equal(t, src, st.Dirs["/src"], 0)
}
