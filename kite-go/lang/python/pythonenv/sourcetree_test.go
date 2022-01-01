package pythonenv

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addFile(sources *SourceTree, srcpath string, windows bool) *pythontype.SourceModule {
	name := pythontype.Address{File: srcpath}
	mod := &pythontype.SourceModule{Members: pythontype.NewSymbolTable(name, nil)}
	sources.AddFile(srcpath, mod, windows)
	return mod
}

func TestSourceTree_Simple(t *testing.T) {
	sources := NewSourceTree()
	mod := addFile(sources, "/src/foo.py", false)
	assert.Len(t, sources.Dirs, 2)
	assert.Len(t, sources.Files, 1)

	root := sources.Dirs["/"]
	require.NotNil(t, root)

	res, _ := pythontype.AttrNoCtx(root, "src")
	require.True(t, res.Found())
	src := res.Value().(*pythontype.SourcePackage)
	require.NotNil(t, src)

	res2, _ := pythontype.AttrNoCtx(src, "foo")
	require.True(t, res2.Found())
	foo := res2.Value().(*pythontype.SourceModule)
	require.True(t, foo == mod)
}

func TestSourceTree_Init(t *testing.T) {
	sources := NewSourceTree()
	mod := addFile(sources, "/src/bar/foo.py", false)
	init := addFile(sources, "/src/bar/__init__.py", false)
	assert.Len(t, sources.Dirs, 3)
	assert.Len(t, sources.Files, 2)

	root := sources.Dirs["/"]
	require.NotNil(t, root)

	res, _ := pythontype.AttrNoCtx(root, "src")
	assert.True(t, res.Found())
	src := res.Value().(*pythontype.SourcePackage)
	require.NotNil(t, src)

	res, _ = pythontype.AttrNoCtx(src, "bar")
	assert.True(t, res.Found())
	bar := res.Value().(*pythontype.SourcePackage)
	require.NotNil(t, bar)

	require.NotNil(t, bar.Init)
	require.True(t, bar.Init == init)

	res, _ = pythontype.AttrNoCtx(bar, "foo")
	require.True(t, res.Found())
	foo := res.Value().(*pythontype.SourceModule)
	require.NotNil(t, foo)
	require.True(t, foo == mod)

	initfile := sources.Files["/src/bar/__init__.py"]
	assert.True(t, initfile == init)
}

func TestSourceTree_Windows(t *testing.T) {
	sources := NewSourceTree()
	mod := addFile(sources, "/src/foo.py", true)
	assert.Len(t, sources.Dirs, 2)
	assert.Len(t, sources.Files, 1)

	root := sources.Dirs["/"]
	require.NotNil(t, root)

	res, _ := pythontype.AttrNoCtx(root, "Src")
	require.True(t, res.Found())
	src := res.Value().(*pythontype.SourcePackage)
	require.NotNil(t, src)

	res2, _ := pythontype.AttrNoCtx(src, "foO")
	require.True(t, res2.Found())
	foo := res2.Value().(*pythontype.SourceModule)
	require.True(t, foo == mod)
}
