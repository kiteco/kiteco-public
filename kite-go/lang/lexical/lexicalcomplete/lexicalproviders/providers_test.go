package lexicalproviders

import (
	"strings"
	"sync"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/stretchr/testify/require"
)

type testOutputs map[data.SelectedBufferHash][]MetaCompletion

func (o testOutputs) add(ctx kitectx.Context, buf data.SelectedBuffer, compl MetaCompletion) {
	/*
		log.Println("testOutputs got:")
		log.Printf("buffer hash: %v, cursor: %v",
			buf.Hash().BufferHash.Hash64(), buf.Hash().Selection,
		)

		log.Printf("compl snippet:\n'%s'\nphs: %v", compl.Snippet.Text, compl.Snippet.Placeholders())
		log.Printf("compl replace:\n %v", compl.Replace)
	*/

	o[buf.Hash()] = append(o[buf.Hash()], compl)
}

type provisionResult struct {
	in  Inputs
	out testOutputs
}

var models *lexicalmodels.Models
var modelMutex sync.Mutex

func init() {
	err := datadeps.Enable()
	if err != nil {
		panic(err)
	}
	datadeps.SetLocalOnly()
}

func initModels(t *testing.T, opts lexicalmodels.ModelOptions) {
	modelMutex.Lock()
	defer modelMutex.Unlock()
	if models != nil {
		for _, lineLog := range lineLogs {
			lineLog.Purge()
		}
		return
	}

	var err error
	models, err = lexicalmodels.NewModels(opts)
	require.NoError(t, err)
	for _, lineLog := range lineLogs {
		lineLog.Purge()
	}
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

func processTemplate(t *testing.T, template string) data.SelectedBuffer {
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

	return data.NewBuffer(text).Select(sel)
}

func runProvider(t *testing.T, p Provider, template, filePath string) (provisionResult, error) {
	return runWithEditorEvents(t, p, template, filePath, nil)
}

func runWithEditorEvents(t *testing.T, p Provider, template, filePath string, editorEvents []*component.EditorEvent) (provisionResult, error) {
	var err error
	switch p.(type) {
	case Text, Python:
		err = nil
	default:
		err = errors.Errorf("unsupported provider type %T", p)
	}
	if err != nil {
		return provisionResult{}, err
	}
	global := Global{
		FilePath:     filePath,
		Models:       models,
		EditorEvents: editorEvents,
		Product:      licensing.Pro,
	}

	buf := processTemplate(t, template)
	in, err := NewInputs(kitectx.Background(), global, buf, false)
	require.NoError(t, err)

	out := make(testOutputs)
	err = p.Provide(kitectx.Background(), global, in, out.add)
	return provisionResult{in, out}, err
}

func requireRes(t *testing.T, p Provider, src, filePath string) provisionResult {
	res, err := runProvider(t, p, src, filePath)
	require.NoError(t, err)
	return res
}
