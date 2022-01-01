package python

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/lang/python/testcorpus"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hackNonUnionsEqual checks if u "equals" v.
// We can't use pythontype.Equal because u, v may not be from the same analysis/propagation run,
// as we may comparing values from the local code index to those from the resolved AST.
// So for source values defined in the current resolved file (defined by path) we just check for address equality.
// This is obviously a hack and isn't great, since we aren't really checking the right thing in e.g. duplicate functions defs with the same name.
// TODO(naman) eventually, in Kite Local world, we should enforce consistency guarantees about the up-to-date-ness of the buffer,
// and then keep the buffer index around as part of the local code index in such a way that this works without hacks.
func hackNonUnionsEqual(path string, u, v pythontype.Value) bool {
	switch u := u.(type) {
	case pythontype.SourceInstance:
		if v, ok := v.(pythontype.SourceInstance); ok {
			return hackNonUnionsEqual(path, u.Class, v.Class)
		}
	case pythontype.SourceValue:
		if addr := u.Address(); addr.File == path {
			return addr.Equals(v.Address())
		}
	}
	return pythontype.Equal(kitectx.Background(), u, v)
}

// hackValuesEqual is similar to pythontype.Union.equal, but calls out to hackNonUnionsEqual
func hackValuesEqual(path string, u, v pythontype.Value) bool {
	uVals := pythontype.Disjuncts(kitectx.Background(), u)
	vVals := pythontype.Disjuncts(kitectx.Background(), v)
	if len(uVals) != len(vVals) {
		return false
	}

outer:
	for _, u := range uVals {
		for _, v := range vVals {
			if hackNonUnionsEqual(path, u, v) {
				continue outer
			}
		}
		return false
	}

	return true
}

func TestCorpusResolves(t *testing.T) {
	testcorpus.DoTest(t, "hover", func(builder *pythonbatch.BuilderLoader, index *pythonlocal.SymbolIndex, path string) {
		ctx := kitectx.Background()

		f, err := os.Open(path)
		require.NoError(t, err)
		contents, err := ioutil.ReadAll(f)
		require.NoError(t, err)
		ast, _ := pythonparser.Parse(ctx, contents, pythonparser.Options{
			Approximate: true,
			// in reality, we pass a cursor position, but for the purposes of hover, it's not relevant
		})
		require.NotNil(t, ast)

		resolved, err := pythonanalyzer.Resolve(ctx, pythonanalyzer.Models{
			Importer: pythonstatic.Importer{
				Path:        path,
				PythonPaths: index.PythonPaths,
				Global:      builder.Graph,
				Local:       index.SourceTree,
			},
		}, ast, pythonanalyzer.Options{
			User:    testcorpus.UserID,
			Machine: testcorpus.MachineID,
			Path:    path,
			Trace:   os.Stdout,
		})
		require.NoError(t, err)

		ins := resolveInputs{
			LocalIndex:  index,
			BufferIndex: newBufferIndex(ctx, resolved, contents, path),
			Resolved:    resolved,
			Graph:       builder.Graph,
			PrintDebug:  log.Printf,
		}
		pythonast.Inspect(ast, func(node pythonast.Node) bool {
			switch node.(type) {
			case *pythonast.NameExpr, *pythonast.AttributeExpr:
			default:
				return true
			}
			expr := node.(pythonast.Expr)

			_, sbs, _ := resolveNode(ctx, expr, ins)

			// the "inferredVal" is just the result of union-ing over all the values yielded by symbol bundle resolution
			var inferredVals []pythontype.Value
			for _, sb := range sbs {
				assert.NotEmpty(t, renderSymbolID(ctx, builder.Graph, sb).String())
				inferredVals = append(inferredVals, sb.valueBundle.val)
			}
			inferredVal := pythontype.Unite(ctx, inferredVals...)

			actualVal := pythontype.Translate(ctx, resolved.References[node.(pythonast.Expr)], builder.Graph)
			if pythonast.GetUsage(expr) == pythonast.Assign {
				// for assigned (LHS) expressions, the resolved AST delegate is called only with the assigned value,
				// which might be just a single disjunct of the actual value we compute for the symbol
				// so here, we just check that the resolved AST value is contained in the "inferredVal"
				// TODO(naman) we should somehow make this consistent so that this becomes unnecessary
				inferreds := pythontype.Disjuncts(ctx, inferredVal)
			outer:
				for _, actual := range pythontype.Disjuncts(ctx, actualVal) {
					for _, inferred := range inferreds {
						if hackNonUnionsEqual(path, inferred, actual) {
							continue outer
						}
					}
					require.Fail(t, "fail", "[%s:%d:%d] could not find resolved value %v for %s in inferred %v", path, node.Begin(), node.End(), actual, pythonast.String(node), inferredVal)
				}
			} else if !hackValuesEqual(path, inferredVal, actualVal) {
				require.Fail(t, "fail", "[%s:%d:%d] resolved value %v for %s does not match inferred %v", path, node.Begin(), node.End(), actualVal, pythonast.String(node), inferredVal)
			}
			return true
		})
	})
}
