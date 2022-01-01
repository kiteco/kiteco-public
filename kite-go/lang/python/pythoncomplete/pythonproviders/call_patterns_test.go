package pythonproviders

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"

	"github.com/stretchr/testify/assert"
)

func TestArgCountNoClosing(t *testing.T) {
	tc := requireTestCase(t, `
the_function(arg1, $
	`)
	callExpr, _ := deepestNotContained(tc.Inputs.UnderSelection(), tc.Inputs.Selection).(*pythonast.CallExpr)
	assert.NotNil(t, callExpr)
	assert.Equal(t, 1, getArgCount(callExpr.Args))

	tc = requireTestCase(t, `
the_function($
	`)
	callExpr, _ = deepestNotContained(tc.Inputs.UnderSelection(), tc.Inputs.Selection).(*pythonast.CallExpr)
	assert.NotNil(t, callExpr)
	assert.Equal(t, 0, getArgCount(callExpr.Args))

	tc = requireTestCase(t, `
the_function(arg1, arg2,$
	`)
	callExpr, _ = deepestNotContained(tc.Inputs.UnderSelection(), tc.Inputs.Selection).(*pythonast.CallExpr)
	assert.NotNil(t, callExpr)
	assert.Equal(t, 2, getArgCount(callExpr.Args))
}

func TestArgCountWithClosing(t *testing.T) {
	tc := requireTestCase(t, `
the_function(arg1, $)
	`)
	callExpr, _ := deepestNotContained(tc.Inputs.UnderSelection(), tc.Inputs.Selection).(*pythonast.CallExpr)
	assert.NotNil(t, callExpr)
	assert.Equal(t, 1, getArgCount(callExpr.Args))

	tc = requireTestCase(t, `
the_function($)
	`)
	callExpr, _ = deepestNotContained(tc.Inputs.UnderSelection(), tc.Inputs.Selection).(*pythonast.CallExpr)
	assert.NotNil(t, callExpr)
	assert.Equal(t, 0, getArgCount(callExpr.Args))
}

// This test makes sure that call pattern Provider works just after a comma even if there's no closing parenthesis
func TestCallPatternsSecondArg(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `import requests

url = "https://it-is-a-good-question/42-is-a-good-answer"
data = dict(field=5)

requests.post(url, $`)
	completions := requireCompletions(t, tc, CallPatterns{})
	require.NotEmpty(t, completions)
}

// TODO(naman, juan, moe) figure out why this is breaking
func TestCallPatternVarArgName(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
import requests
import os

print("test".format("blop", $ )
`)
	completions := requireCompletions(t, tc, CallPatterns{})
	require.NotEmpty(t, completions)
}

// TODO(naman, juan, moe) figure out why this is breaking
func TestCallPatternFormat(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
import os

"test".format($)
`)
	completions := requireCompletions(t, tc, CallPatterns{})
	require.NotEmpty(t, completions)
	require.Equal(t, 3, len(completions))
}

func TestCallPatternAppend(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
import os

[].append($)
`)
	completions := requireCompletions(t, tc, CallPatterns{})
	require.NotEmpty(t, completions)
}

func TestCallPatternGet(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
import os

{}.get($)
`)
	completions := requireCompletions(t, tc, CallPatterns{})
	require.Equal(t, 2, len(completions), "There should be 2 completions for get, just the key, and key + default")
	require.Equal(t, "(key)", completions[0].Snippet.Text)
	require.Equal(t, "(key, default)", completions[1].Snippet.Text)
}

func TestCallPatternPop(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
import os

{}.pop($)
`)
	completions := requireCompletions(t, tc, CallPatterns{})
	require.Equal(t, 2, len(completions), "There should be 2 completions for pop, just the key, and key + default")
	require.Equal(t, "(k, d)", completions[0].Snippet.Text)
	require.Equal(t, "(k)", completions[1].Snippet.Text)
}

func TestCallPatternJSONDumps(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
import json

json.dumps($)
`)
	completions := requireCompletions(t, tc, CallPatterns{})
	expectedCompletions := 4
	require.Equal(t, expectedCompletions, len(completions), fmt.Sprintf("There should be %d completions for dumps", expectedCompletions))
	require.Equal(t, "(obj)", completions[0].Snippet.Text)
	require.Equal(t, "(obj, indent=int)", completions[1].Snippet.Text)
	require.Equal(t, "(obj, indent=int, sort_keys=bool)", completions[2].Snippet.Text)
	require.Equal(t, "(obj, cls=DjangoJSONEncoder)", completions[3].Snippet.Text)
}
