package pythonlocal

import (
	"log"
	"path"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var definitions map[string]Definition

// documentationFromAST extracts documentation from the current file.
func documentationFromAST(ast *pythonanalyzer.ResolvedAST, filepath string) map[string]Documentation {
	if ast == nil || ast.Module == nil {
		return nil
	}

	documentation := make(map[string]Documentation)
	module := ast.Root

	// Extract module documentation
	moduleName := strings.TrimSuffix(path.Base(filepath), ".py")

	if doc := BuildDocumentation(filepath, moduleName, moduleName, module.Body); doc != nil {
		documentation[moduleName] = *doc
	}

	// Extract class and function documentation
	pythonast.Inspect(ast.Root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}

		switch node := node.(type) {
		case *pythonast.ClassDefStmt:
			ref := ast.References[node.Name]
			if ref == nil {
				break
			}

			sc, ok := ref.(*pythontype.SourceClass)
			if !ok {
				break
			}

			name := sc.Address().String()
			doc := BuildDocumentation(filepath, node.Name.Ident.Literal, name, node.Body)
			if doc == nil {
				break
			}
			documentation[name] = *doc
		case *pythonast.FunctionDefStmt:
			ref := ast.References[node.Name]
			if ref == nil {
				break
			}

			sf, ok := ref.(*pythontype.SourceFunction)
			if !ok {
				break
			}

			name := sf.Address().String()
			doc := BuildDocumentation(filepath, node.Name.Ident.Literal, name, node.Body)
			if doc == nil {
				break
			}
			documentation[name] = *doc
		}
		return true
	})

	return documentation
}

// definitionsFromAST extracts definitions from a resolved AST.
func definitionsFromAST(ast *pythonanalyzer.ResolvedAST, lines *linenumber.Map, filepath string) map[string]Definition {
	if ast == nil || ast.Module == nil {
		return nil
	}

	definitions := make(map[string]Definition)
	pythonast.Inspect(ast.Root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}

		switch node := node.(type) {
		case *pythonast.ClassDefStmt:
			ref := ast.References[node.Name]
			if ref == nil {
				break
			}

			sc, ok := ref.(*pythontype.SourceClass)
			if !ok {
				break
			}

			def := BuildDefinition(filepath, node, lines)
			if def == nil {
				break
			}
			definitions[sc.Address().Path.String()] = *def
		case *pythonast.FunctionDefStmt:
			ref := ast.References[node.Name]
			if ref == nil {
				break
			}

			sf, ok := ref.(*pythontype.SourceFunction)
			if !ok {
				break
			}

			def := BuildDefinition(filepath, node, lines)
			if def == nil {
				break
			}
			definitions[sf.Address().Path.String()] = *def
		}
		return true
	})

	return definitions
}

func BenchmarkDefinitionsFromAST(b *testing.B) {
	src := `
def lettersA():
    """letters prints out all the letters!"""
    pass

def lettersB():
    """letters prints out all the letters!"""
    pass

def lettersC():
    """letters prints out all the letters!"""
    pass

def lettersD():
    """letters prints out all the letters!"""
    pass
`
	ast, err := resolveAST(b, src, pythonanalyzer.Options{

		Path: "/alphabet.py",
	})
	if err != nil {
		log.Fatalf("error resolving ast: %v", err)
	}

	b.ResetTimer()

	lines := linenumber.NewMap([]byte(src))

	var defns map[string]Definition
	for i := 0; i < b.N; i++ {
		defns = definitionsFromAST(ast, lines, "/go/alphabet.py")
	}
	definitions = defns
}

func resolveAST(t testing.TB, src string, opts pythonanalyzer.Options) (*pythonanalyzer.ResolvedAST, error) {
	manager := pythonresource.MockManager(t, nil)
	importer := pythonstatic.Importer{
		Path:   opts.Path,
		Global: manager,
	}
	resolver := pythonanalyzer.NewResolverUsingImporter(importer, opts)

	var parseopts pythonparser.Options
	parseopts.ErrorMode = pythonparser.FailFast
	mod, err := pythonparser.Parse(kitectx.Background(), []byte(src), parseopts)
	if err != nil {
		return nil, err
	}
	return resolver.Resolve(mod)

}

func requireResolvedAST(t *testing.T, src string, opts pythonanalyzer.Options) *pythonanalyzer.ResolvedAST {
	ast, err := resolveAST(t, src, opts)
	require.NoError(t, err)
	return ast
}

func TestDocumentationFromAST(t *testing.T) {
	src := `
class A(object):
    """
    Just in case that you haven't known. A is the first letter and the first vowel in the ISO basic Latin alphabet.
    It is similar to the Ancient Greek letter alpha, from which it derives. The upper-case version consists of the
    two slanting sides of a triangle, crossed in the middle by a horizontal bar.
    The lower-case version can be written in two forms: the double-storey a and single-storey ɑ.
    The latter is commonly used in handwriting and fonts based on it,
    especially fonts intended to be read by children. It is also found in italic type.
    """

    def __init__():
	"""init creates an A object"""
	self.upper_case = "A"
	self.lower_case = "a"

def letters():
    """letters prints out all the letters!"""
    pass
`
	ast := requireResolvedAST(t, src, pythonanalyzer.Options{
		Path: "/alphabet.py",
	})

	docs := documentationFromAST(ast, "/go/alphabet.py")
	assert.Equal(t, 3, len(docs))

	doc, found := docs["alphabet.py:A"]
	assert.True(t, found)

	assert.Equal(t, "alphabet.py:A", doc.CanonicalName)
	assert.Equal(t, "/go/alphabet.py", doc.Path)
	assert.Equal(t, "A", doc.Identifier)
}

func findClass(t *testing.T, ast pythonast.Node, className string) *pythonast.ClassDefStmt {
	var ret *pythonast.ClassDefStmt
	pythonast.Inspect(ast, func(node pythonast.Node) bool {
		if class, isclass := node.(*pythonast.ClassDefStmt); isclass && class.Name.Ident.Literal == className {
			ret = class
		}
		return ret == nil
	})
	if ret == nil {
		t.Fatalf("unable to find class %s", className)
	}
	return ret
}

func findFunc(t *testing.T, ast pythonast.Node, funcName string) *pythonast.FunctionDefStmt {
	var ret *pythonast.FunctionDefStmt
	pythonast.Inspect(ast, func(node pythonast.Node) bool {
		if fun, isfunc := node.(*pythonast.FunctionDefStmt); isfunc && fun.Name.Ident.Literal == funcName {
			ret = fun
		}
		return ret == nil
	})
	if ret == nil {
		t.Fatalf("unable to find function %s", funcName)
	}
	return ret
}

func findExpr(t *testing.T, ast pythonast.Node, src string, expr string) pythonast.Expr {
	var ret pythonast.Expr
	pythonast.Inspect(ast, func(node pythonast.Node) bool {
		if n, isexpr := node.(pythonast.Expr); isexpr && src[node.Begin():node.End()] == expr {
			ret = n
		}
		return ret == nil
	})
	if ret == nil {
		t.Fatalf("unable to find expression '%s'", expr)
	}
	return ret
}

func TestBuildDefinitions(t *testing.T) {
	src := `
class A(object):
	def __init__(self):
		print("test")

def test():
	print("test")

g = 3
`

	lines := linenumber.NewMap([]byte(src))
	ast, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err)

	class := findClass(t, ast, "A")
	def := BuildDefinition("/User/Kite/test.py", class, lines)
	assert.Equal(t, "/User/Kite/test.py", def.Path)
	assert.Equal(t, 1, def.Line)

	fun := findFunc(t, ast, "test")
	def = BuildDefinition("/User/Kite/test.py", fun, lines)
	assert.Equal(t, "/User/Kite/test.py", def.Path)
	assert.Equal(t, 5, def.Line)
}

func TestMakeName(t *testing.T) {
	assert.Equal(t, "builtins.print", pythontype.Address{Path: pythonimports.NewDottedPath("builtins.print")}.String())
}

func TestWindowsFilepath(t *testing.T) {
	src := `
class Foo():
	''' Foo is a class that bars '''
	def __init__(self): pass

f = Foo()
	`

	path := "/windows/c/users/juan/scratch.py"
	expectedPath := `c:\users\juan\scratch.py`

	lines := linenumber.NewMap([]byte(src))
	ast := requireResolvedAST(t, src, pythonanalyzer.Options{
		Path: "/scratch.py",
	})

	class := findClass(t, ast.Root, "Foo")
	def := BuildDefinition(path, class, lines)
	require.NotNil(t, def)
	assert.Equal(t, expectedPath, def.Path)

	docs := documentationFromAST(ast, path)

	doc := docs["scratch.py:Foo"]
	require.NotNil(t, doc)
	assert.Equal(t, expectedPath, doc.Path)
}
