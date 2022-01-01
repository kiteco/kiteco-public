package python

import (
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var manager pythonresource.Manager

func defaultManager(t *testing.T) pythonresource.Manager {
	if manager == nil {
		manager = pythonresource.MockManager(t, nil)
	}
	return manager
}

type ancestorTestData struct {
	Path string
	File string
	Name string
}

func ancestorsTestFile(path string) string {
	return filepath.Join("./pythontest/testfiles/ancestors", path)
}

func assertAncestors(t *testing.T, graph *pythonimports.Graph, manager pythonresource.Manager, files map[string]string, addr pythontype.Address, expected []ancestorTestData) {
	uid, mid := int64(1), "machine-id"
	st, err := pythontest.SourceTree(uid, mid, manager, files)
	require.NoError(t, err)

	val, err := st.FindValue(kitectx.Background(), addr.File, addr.Path.Parts)
	require.NoError(t, err)
	require.NotNil(t, val)

	vb := valueBundle{
		val: val,
		indexBundle: indexBundle{
			idx: &pythonlocal.SymbolIndex{SourceTree: st},
		},
	}

	editor := newEditorServices(&Services{ImportGraph: graph, ResourceManager: manager})

	actual := editor.renderValueAncestors(kitectx.Background(), vb)
	if len(actual) != len(expected) {
		t.Errorf("expected actual ancestors length %d got %d: %v", len(expected), len(actual), actual)
		return
	}

	for i, aa := range actual {
		td := expected[i]
		ea := editorapi.Ancestor{
			Name: td.Name,
			ID: editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(pythontype.Address{
				User:    uid,
				Machine: mid,
				File:    td.File,
				Path:    pythonimports.NewDottedPath(td.Path),
			})),
		}
		assert.Equal(t, ea, aa)
	}
}

func TestAncestorsEdgeCases(t *testing.T) {
	graph := pythonimports.MockGraph()
	manager := defaultManager(t)

	// nil value
	editor := newEditorServices(&Services{ImportGraph: graph, ResourceManager: manager})

	assert.Nil(t, editor.renderValueAncestors(kitectx.Background(), valueBundle{}))

	// nil address
	fakePath := "/Users/juan/src/server/server/server.py"
	files := map[string]string{
		fakePath: ancestorsTestFile("server/server.py"),
	}

	addr := pythontype.Address{
		File: fakePath,
		Path: pythonimports.NewDottedPath("s"),
	}

	assertAncestors(t, graph, manager, files, addr, nil)
}

func TestAncestorsGlobal(t *testing.T) {
	fakePath := "/Users/juan/src/server/server/server.py"
	files := map[string]string{
		fakePath: ancestorsTestFile("server/server.py"),
	}

	expected := []ancestorTestData{
		{Path: "django", Name: "django"},
		{Path: "django.db", Name: "db"},
	}

	addr := pythontype.Address{
		File: fakePath,
		Path: pythonimports.NewDottedPath("models"),
	}

	graph := pythonimports.MockGraph("django.db.models")
	manager := pythonresource.MockManager(t, nil, "django.db.models")
	assertAncestors(t, graph, manager, files, addr, expected)
}

func TestAncestorsUnixLike(t *testing.T) {
	files := map[string]string{
		"/Users/juan/src/server/server/server.py": ancestorsTestFile("server/server.py"),
		"/Users/juan/src/server/server/logger.py": ancestorsTestFile("server/logger.py"),
		"/Users/juan/src/server/run.py":           ancestorsTestFile("run.py"),
	}

	expected := []ancestorTestData{
		{File: "/Users/juan/src/server", Name: "server"},
		{File: "/Users/juan/src/server/server", Name: "server"},
		{File: "/Users/juan/src/server/server/server.py", Name: "server"},
		{File: "/Users/juan/src/server/server/server.py", Path: "Server", Name: "Server"},
	}

	addr := pythontype.Address{
		File: "/Users/juan/src/server/server/server.py",
		Path: pythonimports.NewDottedPath("Server.start"),
	}

	graph := pythonimports.MockGraph()
	manager := defaultManager(t)
	assertAncestors(t, graph, manager, files, addr, expected)
}

func TestAncestorsWindows(t *testing.T) {
	files := map[string]string{
		"/windows/c/users/juan/src/server/server/server.py": ancestorsTestFile("server/server.py"),
		"/windows/c/users/juan/src/server/server/logger.py": ancestorsTestFile("server/logger.py"),
		"/windows/c/users/juan/src/server/run.py":           ancestorsTestFile("run.py"),
	}

	expected := []ancestorTestData{
		{File: "/windows/c/users/juan/src/server", Name: "server"},
		{File: "/windows/c/users/juan/src/server/server", Name: "server"},
		{File: "/windows/c/users/juan/src/server/server/server.py", Name: "server"},
		{File: "/windows/c/users/juan/src/server/server/server.py", Path: "Server", Name: "Server"},
	}

	addr := pythontype.Address{
		File: "/windows/c/users/juan/src/server/server/server.py",
		Path: pythonimports.NewDottedPath("Server.start"),
	}

	graph := pythonimports.MockGraph()
	manager := defaultManager(t)
	assertAncestors(t, graph, manager, files, addr, expected)
}

func TestAncestorsUnixLikeSingleModuleMember(t *testing.T) {
	fakePath := "/Users/juan/src/server/server.py"
	files := map[string]string{
		fakePath: ancestorsTestFile("/server/server.py"),
	}

	expected := []ancestorTestData{
		{File: fakePath, Name: "server"},
		{File: fakePath, Path: "Server", Name: "Server"},
	}

	addr := pythontype.Address{
		File: fakePath,
		Path: pythonimports.NewDottedPath("Server.start"),
	}

	graph := pythonimports.MockGraph()
	manager := defaultManager(t)
	assertAncestors(t, graph, manager, files, addr, expected)
}

func TestAncestorsUnixLikeSingleModule(t *testing.T) {
	fakePath := "/Users/juan/src/server/server.py"
	files := map[string]string{
		fakePath: ancestorsTestFile("/server/server.py"),
	}

	// test top level module
	expected := []ancestorTestData{
		{File: "/Users/juan/src/server", Name: "server"},
	}

	addr := pythontype.Address{
		File: "/Users/juan/src/server/server.py",
	}

	graph := pythonimports.MockGraph()
	manager := defaultManager(t)
	assertAncestors(t, graph, manager, files, addr, expected)
}

func TestAncestorsWindowsSingleModuleMember(t *testing.T) {
	fakePath := "/windows/c/users/juan/src/server/server.py"
	files := map[string]string{
		fakePath: ancestorsTestFile("/server/server.py"),
	}

	expected := []ancestorTestData{
		{File: fakePath, Name: "server"},
		{File: fakePath, Path: "Server", Name: "Server"},
	}

	addr := pythontype.Address{
		File: fakePath,
		Path: pythonimports.NewDottedPath("Server.start"),
	}

	graph := pythonimports.MockGraph()
	manager := defaultManager(t)
	assertAncestors(t, graph, manager, files, addr, expected)
}

func TestAncestorsWindowsSingleModule(t *testing.T) {
	fakePath := "/windows/c/users/juan/src/server/server.py"
	files := map[string]string{
		fakePath: ancestorsTestFile("/server/server.py"),
	}

	// test top level module
	expected := []ancestorTestData{
		{File: "/windows/c/users/juan/src/server", Name: "server"},
	}

	addr := pythontype.Address{
		File: "/windows/c/users/juan/src/server/server.py",
	}

	graph := pythonimports.MockGraph()
	manager := defaultManager(t)
	assertAncestors(t, graph, manager, files, addr, expected)
}
