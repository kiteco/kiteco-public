package pythonproviders

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

type testOutput struct {
	data.SelectedBuffer
	MetaCompletion
}

type testOutputs map[data.SelectedBufferHash][]MetaCompletion

func (o testOutputs) add(ctx kitectx.Context, buf data.SelectedBuffer, compl MetaCompletion) {
	o[buf.Hash()] = append(o[buf.Hash()], compl)
}

func nopOutputFunc(kitectx.Context, data.SelectedBuffer, MetaCompletion) {
}

type provisionResult struct {
	in  Inputs
	out testOutputs
}

// contains checks that the output contains the given completion at the "root" state
func (r provisionResult) containsRoot(expected data.Completion) bool {
	_, ok := r.getFromRoot(expected)
	return ok
}
func (r provisionResult) getFromRoot(expected data.Completion) (MetaCompletion, bool) {
	expectedSnippet := expected.Snippet.ForFormat()

	expectedReplace := expected.Replace
	expectedReplace.Begin += r.in.Selection.Begin
	expectedReplace.End += r.in.Selection.End

	for _, compl := range r.out[r.in.SelectedBuffer.Hash()] {
		if compl.Snippet.ForFormat() == expectedSnippet && compl.Replace == expectedReplace {
			return compl, true
		}
	}
	return MetaCompletion{}, false
}

func runProvider(t *testing.T, p Provider, template string) (provisionResult, error) {
	return runProviderWithOpts(t, p, template, false)
}

func runProviderWithOpts(t *testing.T, p Provider, template string, usePartialDecoder bool) (provisionResult, error) {
	rm := pythonresource.DefaultTestManager(t)

	if models == nil || lexicalModels == nil {
		initModels(t)
	}
	global := Global{
		ResourceManager: rm,
		FilePath:        "/src.py",
		Models:          models,
		Product:         licensing.Pro,
	}
	global.Lexical.FilePath = global.FilePath
	global.Lexical.Models = lexicalModels

	parts := strings.Split(template, "$")
	text := strings.Join(parts, "")
	var sel data.Selection
	switch len(parts) {
	case 1:
		fallthrough
	case 2:
		sel = data.Cursor(len(parts[0]))
	case 3:
		sel = data.Selection{Begin: len(parts[0]), End: len(parts[0]) + len(parts[1])}
	default:
		require.Fail(t, "invalid template string for test Inputs")
	}

	buf := data.NewBuffer(text).Select(sel)
	in, err := NewInputs(kitectx.Background(), global, buf, false, usePartialDecoder)
	require.NoError(t, err)

	out := make(testOutputs)
	err = p.Provide(kitectx.Background(), global, in, out.add)
	return provisionResult{in, out}, err
}
