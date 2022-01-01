package pythonproviders

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"

	"github.com/stretchr/testify/assert"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	Orig   string
	Global Global
	Inputs Inputs
}

var (
	models        *pythonmodels.Models
	lexicalModels *lexicalmodels.Models
)

func initModels(t *testing.T) {
	var err error
	if models == nil {
		models, err = pythonmodels.New(pythonmodels.DefaultOptions)
		require.NoError(t, err)
	}
	if lexicalModels == nil {
		lexicalModels, err = lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions)
		require.NoError(t, err)
	}
}

func requireSelectedBuffer(t *testing.T, src string) data.SelectedBuffer {
	var sb data.SelectedBuffer
	switch parts := strings.Split(src, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(src).Select(data.Cursor(len(src)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
		t.Fatalf("bad test case source, expect 1 or 2 parts, got %d for: %s", len(parts), src)
	}
	return sb
}

func requireTestCase(t *testing.T, src string) testCase {
	sb := requireSelectedBuffer(t, src)

	var models *pythonmodels.Models
	var err error

	global := Global{
		FilePath:        "/src.py",
		ResourceManager: pythonresource.DefaultTestManager(t),
		Models:          models,
		Product:         licensing.Pro,
	}
	global.Lexical.FilePath = global.FilePath
	global.Lexical.Models = lexicalModels

	inputs, err := NewInputs(kitectx.Background(), global, sb, false, false)
	require.NoError(t, err)
	return testCase{
		Orig:   src,
		Global: global,
		Inputs: inputs,
	}
}

func requireTestCaseWithModelAndSymbols(t *testing.T, src string) testCase {
	require.NoError(t, datadeps.Enable())
	initModels(t)
	sb := requireSelectedBuffer(t, src)

	global := Global{
		FilePath:        "/src.py",
		ResourceManager: pythonresource.DefaultTestManager(t),
		Models:          models,
		Product:         licensing.Pro,
	}
	global.Lexical.FilePath = global.FilePath
	global.Lexical.Models = lexicalModels

	inputs, err := NewInputs(kitectx.Background(), global, sb, false, false)
	require.NoError(t, err)
	return testCase{
		Orig:   src,
		Global: global,
		Inputs: inputs,
	}
}

func requireCompletions(t *testing.T, tc testCase, providers ...Provider) []MetaCompletion {
	var completions []MetaCompletion
	for _, p := range providers {
		err := p.Provide(kitectx.Background(), tc.Global, tc.Inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc MetaCompletion) {
			completions = append(completions, mc)
		})
		require.NoError(t, err)
	}
	return completions
}

func requireCompletionsOrError(t *testing.T, tc testCase, providers ...Provider) ([]MetaCompletion, error) {
	var completions []MetaCompletion
	for _, p := range providers {
		err := p.Provide(kitectx.Background(), tc.Global, tc.Inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc MetaCompletion) {
			completions = append(completions, mc)
		})
		if err != nil {
			return nil, err
		}
	}
	return completions, nil
}

func TestNoSelfOrCls(t *testing.T) {
	tc := requireTestCase(t, `
"".split($)
	`)

	comps := requireCompletions(t, tc, KWArgs{}, CallPatterns{})
	for _, c := range comps {
		for _, ph := range c.Snippet.Placeholders() {
			arg := c.Snippet.Text[ph.Begin:ph.End]
			assert.NotContains(t, "self", arg)
			assert.NotContains(t, "cls", arg)
		}
	}
}

func TestUsedArg(t *testing.T) {
	tc := requireTestCase(t, `
"".split(s,$)
	`)

	comps := requireCompletions(t, tc, KWArgs{}, CallPatterns{})
	for _, c := range comps {
		for _, ph := range c.Snippet.Placeholders() {
			arg := c.Snippet.Text[ph.Begin:ph.End]
			assert.NotEqual(t, "sep", arg)
		}
	}
}

// This test makes sure that call pattern Provider works just after a comma even if there's no closing parenthesis
func TestCallModelSecondArg(t *testing.T) {
	t.Skip("call model times out right now")
	tc := requireTestCaseWithModelAndSymbols(t, `import requests 

url = "https://it-is-a-good-question/42-is-a-good-answer"
data = dict(field=5)

requests.get(url, $`)
	completions := requireCompletions(t, tc, CallModel{})
	require.NotEmpty(t, completions)
}

func TestEmptyCallCallExists(t *testing.T) {
	tc := requireTestCase(t, `import json
json.loads$
	`)
	requireCompletions(t, tc, EmptyCalls{})

	tc = requireTestCase(t, `import json
json.loads$(
	`)
	_, err := requireCompletionsOrError(t, tc, EmptyCalls{})
	require.Error(t, err, "empty call should not be emitted if call exists")
}

func TestEmptyCallFunctionDef(t *testing.T) {
	tc := requireTestCase(t, `def foo$`)
	_, err := requireCompletionsOrError(t, tc, EmptyCalls{})
	require.Error(t, err, "empty call should not be emitted for function definition")

	// should emit for function call within function definition
	tc = requireTestCase(t, `import json
def foo():
	json.loads$
	`)
	requireCompletions(t, tc, EmptyCalls{})
}

func TestEmptyCall(t *testing.T) {
	tc := requireTestCase(t, `
def foo():

foo$
`)
	requireCompletions(t, tc, EmptyCalls{})
}
