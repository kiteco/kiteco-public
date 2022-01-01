// package localcodetests contains tests for pythonanalyzer that would create circular
// dependencies if they were in the pythonanalyzer package itself

package localcodetests

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type opts struct {
	manager    pythonresource.Manager
	src        string            // the python source code to analyze
	srcpath    string            // treat src as being from this path, or empty for no local index
	localfiles map[string]string // other local paths to include in index
	expected   map[string]string // map from expressions to their expected values
}

func valueString(v pythontype.Value) string {
	if v == nil {
		return "<nil>"
	}
	vs := fmt.Sprintf("%v", v)
	if strings.HasPrefix(vs, "generic:") {
		return strings.TrimPrefix(vs, "generic:")
	}
	return vs
}

func managerAdd(t testing.TB, batchOpts pythonbatch.Options, m *pythonbatch.BatchManager, path, src string) {
	parseOpts := batchOpts.PathSelection.Parse
	parseOpts.ScanOptions.Label = path
	mod, _ := pythonparser.Parse(kitectx.Background(), []byte(src), parseOpts)
	require.NotNil(t, mod)
	m.Add(&pythonbatch.SourceUnit{
		ASTBundle: pythonstatic.ASTBundle{AST: mod, Path: path, Imports: pythonstatic.FindImports(kitectx.Background(), path, mod)},
		Contents:  []byte(src),
		Hash:      path,
	})
}

func assertValue(t *testing.T, expectedName string, v pythontype.Value, expr string) {
	if expectedName == "unknown" {
		assert.Nil(t, v, "expected node for %s to be nil but got %v", expr, v)
		return
	}
	if !assert.NotNil(t, v, "expected %s to resolve to %s but node was nil", expr, expectedName) {
		return
	}
	if strings.HasPrefix(expectedName, "instanceof ") {
		if !assert.NotNil(t, v.Type(),
			"expected %s to resolve to %s but got %s", expr, expectedName, v) {
			return
		}
		expectedType := strings.TrimPrefix(expectedName, "instanceof ")
		assert.Equal(t, expectedType, valueString(v.Type()),
			"expected %s to resolve to %s but got instanceof %s", expr, expectedName, valueString(v.Type()))
	} else {
		assert.Equal(t, expectedName, valueString(v),
			"expected %s to resolve to %s but got %s", expr, expectedName, valueString(v))
	}
}

func assertResolveOpts(t *testing.T, opts opts) *pythonanalyzer.ResolvedAST {
	for i, line := range strings.Split(opts.src, "\n") {
		t.Logf("%3d  %s", i+1, line)
	}

	manager := opts.manager
	if manager == nil {
		manager = pythonresource.MockManager(t, nil)
	}

	// build local index if requested
	var sourceTree *pythonenv.SourceTree
	if len(opts.localfiles) > 0 {
		var batchOpts pythonbatch.Options
		batchOpts.Options.Passes = 3
		bi := pythonbatch.BatchInputs{
			Graph: manager,
		}

		mgr := pythonbatch.NewBatchManager(kitectx.Background(), bi, batchOpts, nil)
		managerAdd(t, batchOpts, mgr, opts.srcpath, opts.src)
		for path, buf := range opts.localfiles {
			if path == opts.srcpath {
				continue
			}
			managerAdd(t, batchOpts, mgr, path, buf)
		}

		batch, err := mgr.Build(kitectx.Background())
		require.NoError(t, err)

		flatsources, err := batch.Assembly.Sources.Flatten(kitectx.Background())
		require.NoError(t, err)

		sourceTree, err = flatsources.Inflate(manager)
		require.NoError(t, err)
	}

	// Parse source
	var parseOpts pythonparser.Options
	mod, err := pythonparser.Parse(kitectx.Background(), []byte(opts.src), parseOpts)
	require.NoError(t, err)

	// Create resolver
	imp := pythonstatic.Importer{
		Path:   opts.srcpath,
		Global: manager,
		Local:  sourceTree,
	}
	r := pythonanalyzer.NewResolverUsingImporter(imp, pythonanalyzer.Options{
		Path: opts.srcpath,
	})

	// Resolve source
	result, err := r.Resolve(mod)
	require.NoError(t, err)

	file := pythonscanner.File([]byte(opts.src))
	require.NotNil(t, file)

	// Print references
	var refs []*reference
	for expr, ref := range result.References {
		refs = append(refs, &reference{
			Value:      ref,
			Expression: expr,
		})
	}
	sort.Sort(byPosition(refs))

	for _, ref := range refs {
		s := pythonast.String(ref.Expression)
		require.NotNil(t, ref.Expression)
		line := file.Line(ref.Expression.Begin())
		require.NotNil(t, ref, "nil value found in references (for %s)", s)
		require.NotNil(t, ref.Expression, "nil expression found in references (for %s)", s)
		t.Logf("%25s (line %2d) -> %-40s", s, line, valueString(ref.Value))
	}

	// Check that the types match their expected values
	for exprStr, expectedName := range opts.expected {
		expr := findExpr(mod, exprStr, opts.src)
		require.NotNil(t, expr, "could not find AST node for '%s'", exprStr)
		line := file.Line(expr.Begin())
		ref, hasref := result.References[expr]

		t.Logf("resolving %s (line %d)", pythonast.String(expr), line)
		if !assert.True(t, hasref, "expected %s to resolve to %s but no reference found", exprStr, expectedName) {
			continue
		}
		assertValue(t, expectedName, ref, exprStr)
	}

	// check that all values are resolvable in the local index
	if sourceTree != nil {
		for _, ref := range result.References {
			if ref == nil || ref.Address().Nil() {
				continue
			}

			loc := pythonenv.Locator(ref)
			if loc == "" {
				t.Errorf("got empty locator for %v", ref)
				continue
			}

			var val pythontype.Value
			if !pythonenv.IsLocator(loc) {
				addr, attr, err := pythonenv.ParseLocator(loc)
				if err != nil {
					t.Errorf("got error %v while trying to parse locator %s (%v)", err, loc, ref)
					continue
				}
				if attr != "" {
					addr.Path = addr.Path.WithTail(attr)
				}
				// global graph
				sym, err := manager.PathSymbol(addr.Path)
				if err != nil {
					t.Errorf("got error %v while trying to locate %s (%v)", err, loc, ref)
					continue
				}
				val = pythontype.NewExternal(sym, manager)
			} else {
				var err error
				val, err = sourceTree.Locate(kitectx.Background(), loc)
				if err != nil {
					t.Errorf("got error %v while trying to locate %s (%v)", err, loc, ref)
					continue
				}

				if val == nil {
					t.Errorf("unable to locate %s (%v)", loc, ref)
					continue
				}
			}

			assert.Equal(t, pythonlocal.LookupID(val), pythonlocal.LookupID(ref), "%v != %v", val, ref)
		}
	}

	return result
}

// Find a node in an AST given the source for the node
func findExpr(root pythonast.Node, s string, orig string) pythonast.Expr {
	var ret pythonast.Expr
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		expr, isexpr := node.(pythonast.Expr)
		if isexpr && orig[expr.Begin():expr.End()] == s {
			ret = expr
		}
		return ret == nil
	})
	return ret
}

// Find a function def in an AST
func findFunctionDef(root pythonast.Node, name string) *pythonast.FunctionDefStmt {
	var ret *pythonast.FunctionDefStmt
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		funcdef, isfunc := node.(*pythonast.FunctionDefStmt)
		if isfunc && funcdef.Name.Ident.Literal == name {
			ret = funcdef
		}
		return ret == nil
	})
	return ret
}

// Find a class def in an AST
func findClassDef(root pythonast.Node, name string) *pythonast.ClassDefStmt {
	var ret *pythonast.ClassDefStmt
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		classdef, isclass := node.(*pythonast.ClassDefStmt)
		if isclass && classdef.Name.Ident.Literal == name {
			ret = classdef
		}
		return ret == nil
	})
	return ret
}

type reference struct {
	Value      pythontype.Value
	Expression pythonast.Expr
}

type byPosition []*reference

func (xs byPosition) Len() int           { return len(xs) }
func (xs byPosition) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byPosition) Less(i, j int) bool { return xs[i].Expression.Begin() < xs[j].Expression.Begin() }
