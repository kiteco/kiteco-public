package python

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toValueExt(positional, kwargs []string, vararg, kwarg string, constructor bool) *editorapi.ValueExt {
	details := editorapi.FunctionDetails{}
	for _, p := range positional {
		details.Parameters = append(details.Parameters, &editorapi.Parameter{
			Name: p,
		})
	}
	details.LanguageDetails.Python = &editorapi.PythonFunctionDetails{}

	var kwparams []*editorapi.Parameter
	for _, kw := range kwargs {
		kwparams = append(kwparams, &editorapi.Parameter{
			Name: kw,
		})
	}
	details.LanguageDetails.Python.KwargParameters = kwparams

	if vararg != "" {
		details.LanguageDetails.Python.Vararg = &editorapi.Parameter{
			Name: vararg,
		}
	}
	if kwarg != "" {
		details.LanguageDetails.Python.Kwarg = &editorapi.Parameter{
			Name: kwarg,
		}
	}

	if constructor {
		return &editorapi.ValueExt{
			Details: editorapi.Details{
				Type: &editorapi.TypeDetails{
					LanguageDetails: editorapi.LanguageTypeDetails{
						Python: &editorapi.PythonTypeDetails{
							Constructor: &details,
						},
					},
				},
			},
		}
	}

	return &editorapi.ValueExt{
		Details: editorapi.Details{
			Function: &details,
		},
	}
}

func requireCallee(t *testing.T, def string, kwargs []string, constructor bool) *editorapi.ValueExt {
	stmt, err := pythonparser.ParseStatement(kitectx.Background(), []byte(def), pythonparser.Options{})
	require.NoError(t, err)

	require.IsType(t, &pythonast.FunctionDefStmt{}, stmt)
	fn := stmt.(*pythonast.FunctionDefStmt)

	var positional []string
	for _, param := range fn.Parameters {
		positional = append(positional, param.Name.(*pythonast.NameExpr).Ident.Literal)
	}

	var vararg, kwarg string
	if fn.Vararg != nil {
		vararg = fn.Vararg.Name.Ident.Literal
	}
	if fn.Kwarg != nil {
		kwarg = fn.Kwarg.Name.Ident.Literal
	}

	return toValueExt(positional, kwargs, vararg, kwarg, constructor)
}

func assertCall(t *testing.T, testCase string, callee *editorapi.ValueExt, expectedArgIndex int, expectedInKwargs bool) {

	src, cursor := requireCallExample(t, testCase)

	mgr, err := NewManager(callee, nil, "funcName", "filename", src, cursor, true)
	require.NoError(t, err)

	resp := mgr.Handle(src, cursor)
	require.NotNil(t, resp, "Handle() response was nil.")

	assert.Equal(t, "python", resp.Language)
	require.Len(t, resp.Calls, 1)

	call := resp.Calls[0]
	assert.Equal(t, expectedArgIndex, call.ArgIndex)
	assert.Equal(t, expectedInKwargs, call.LanguageDetails.Python.InKwargs)
}

func TestVarargPositional1(t *testing.T) {
	def := `
def foo(a,b,c,d, *vararg):
	pass
	`

	c := `foo(1,2<caret>)`

	assertCall(t, c, requireCallee(t, def, nil, false), 1, false)
}

func TestVarargPositional2(t *testing.T) {
	def := `
def foo(a,b,c,d, *vararg):
	pass
		`

	c := `foo(1,2,3,4,a<caret>)`

	// vararg is indicated by len(parameters), false
	assertCall(t, c, requireCallee(t, def, nil, false), 4, false)
}

func TestVarargPositional3(t *testing.T) {
	def := `
def foo(a,b,c,d, *vararg):
	pass
	`

	c := `foo(1,2,<caret>)`

	// `a` and `b` slots already filled, must suggest `c`
	assertCall(t, c, requireCallee(t, def, nil, false), 2, false)
}

func TestVarargPositional4(t *testing.T) {
	def := `
def foo(a,b, *vararg, **kwarg):
	pass
	`

	c := `foo(1,2,c=<caret>)`

	assertCall(t, c, requireCallee(t, def, []string{"cc"}, false), 0, true)
}

func TestVargargPositional5(t *testing.T) {
	def := `
def foo(a,b,c, *vararg):
	pass
	`

	c := `foo(1,2,c<caret>)`

	assertCall(t, c, requireCallee(t, def, nil, false), 2, false)
}
