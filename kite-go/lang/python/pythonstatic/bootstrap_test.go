package pythonstatic

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bootstrap(t *testing.T, sources map[string]string) []string {
	var opts pythonparser.Options
	imports := make(map[string][]ImportPath)
	for srcpath, src := range sources {
		ast, err := pythonparser.Parse(kitectx.Background(), []byte(src), opts)
		require.NoError(t, err)
		imports[srcpath] = FindImports(kitectx.Background(), srcpath, ast)
	}
	deps := ComputeDependencies(imports)

	t.Logf("%# v", pretty.Formatter(imports))
	t.Logf("%# v", pretty.Formatter(deps))

	return ComputeBootstrapSequence(deps)
}

func TestBootstrapTwo(t *testing.T) {
	files := make(map[string]string)
	files["/src/foo.py"] = "import bar"
	files["/src/bar.py"] = ""

	order := bootstrap(t, files)
	assert.Len(t, order, 2)
	assert.Equal(t, "/src/bar.py", order[0])
	assert.Equal(t, "/src/foo.py", order[1])
}

func TestBootstrapDag(t *testing.T) {
	files := make(map[string]string)
	files["/src/a.py"] = "import b, c, d"
	files["/src/b.py"] = "import c, d"
	files["/src/c.py"] = "import d"
	files["/src/d.py"] = ""

	order := bootstrap(t, files)
	assert.Len(t, order, 4)
	assert.Equal(t, "/src/d.py", order[0])
	assert.Equal(t, "/src/c.py", order[1])
	assert.Equal(t, "/src/b.py", order[2])
	assert.Equal(t, "/src/a.py", order[3])
}

func TestBootstrapRelative(t *testing.T) {
	files := make(map[string]string)
	files["/src/a/b/src.py"] = "from ...x import foo"
	files["/src/a/x.py"] = ""
	files["/src/x.py"] = "import a.x"

	order := bootstrap(t, files)
	assert.Len(t, order, 3)
	assert.Equal(t, "/src/a/x.py", order[0])
	assert.Equal(t, "/src/x.py", order[1])
	assert.Equal(t, "/src/a/b/src.py", order[2])
}

func TestFindImports_ImportFromStar(t *testing.T) {
	src := `
from foo import *
	`

	ast, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err)

	imps := FindImports(kitectx.Background(), "/path", ast)
	require.Len(t, imps, 1)

	imp := imps[0]

	require.Len(t, imp.Path.Parts, 1)
	assert.Equal(t, "foo", imp.Path.Parts[0])
}
