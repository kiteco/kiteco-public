package api

import (
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/stretchr/testify/assert"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"

	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	rm            pythonresource.Manager
	models        *pythonmodels.Models
	lexicalModels *lexicalmodels.Models
)

func init() {
	datadeps.Enable()

	var errc <-chan error
	rm, errc = pythonresource.NewManager(pythonresource.SmallOptions)
	if err := <-errc; err != nil {
		panic(err)
	}

	var err error
	models, err = pythonmodels.New(pythonmodels.DefaultOptions)
	if err != nil {
		panic(err)
	}

	lexicalModels, err = lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions)
	if err != nil {
		panic(err)
	}
}

type testCase struct {
	Orig   string
	Global pythonproviders.Global
	Inputs pythonproviders.Inputs
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

func requireTestCaseWithModelAndSymbols(t *testing.T, src string) testCase {
	require.NoError(t, datadeps.Enable())
	sb := requireSelectedBuffer(t, src)

	global := pythonproviders.Global{
		FilePath:        "/src.py",
		ResourceManager: pythonresource.DefaultTestManager(t),
		Models:          models,
		Product:         licensing.Pro,
	}
	global.Lexical.FilePath = global.FilePath
	global.Lexical.Models = lexicalModels

	inputs, err := pythonproviders.NewInputs(kitectx.Background(), global, sb, false, false)
	require.NoError(t, err)
	return testCase{
		Orig:   src,
		Global: global,
		Inputs: inputs,
	}
}

func completionMap(completions []data.NRCompletion, functor func(completion *data.RCompletion)) {
	for ii := range completions {
		c := &completions[ii]
		functor(&c.RCompletion)
	}
}

func requireCompletions(t *testing.T, tc testCase, ps ...pythonproviders.Provider) data.APIResponse {
	req := data.APIRequest{
		SelectedBuffer: tc.Inputs.SelectedBuffer,
		UMF: data.UMF{
			Filename: "/test.py",
		},
	}
	opts := IDCCCompleteOptions
	opts.BlockDebug = true

	var resp data.APIResponse
	require.NoError(t, kitectx.Background().WithTimeout(5*time.Second, func(ctx kitectx.Context) error {
		resp = New(ctx.Context(), Options{
			ResourceManager: rm,
			Models:          models,
			LexicalModels:   lexicalModels,
		}, licensing.Pro).Complete(ctx, opts, req, nil, nil)
		return resp.ToError()
	}))
	return resp
}

func assertCompletion(t *testing.T, comps []data.NRCompletion, snippet string, assertPresent bool) {
	var compCounter int
	completionMap(comps, func(completion *data.RCompletion) {
		if completion.Snippet.Text == snippet {
			compCounter++
		}
	})
	if assertPresent {
		assert.NotEqual(t, 0, compCounter, "The completion ##%s## was absent from the completion list", snippet)
	} else {
		assert.Equal(t, 0, compCounter, "The completion ##%s## was present in the completion list and it shouldn't have", snippet)
	}
}

func assertCompletionPresent(t *testing.T, comps []data.NRCompletion, snippet string) {
	assertCompletion(t, comps, snippet, true)
}

func assertCompletionNotPresent(t *testing.T, comps []data.NRCompletion, snippet string) {
	assertCompletion(t, comps, snippet, false)
}

func TestMixingDedup(t *testing.T) {
	t.Skip()
	tc := requireTestCaseWithModelAndSymbols(t, `
open("the_file.txt", $)
	`)

	resp := requireCompletions(t, tc)
	assertCompletionPresent(t, resp.Completions, "mode)")
	assertCompletionNotPresent(t, resp.Completions, "mode=...)")
}
