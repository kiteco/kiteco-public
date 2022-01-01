package pythonstatic

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/stretchr/testify/require"
)

var fullTestManager pythonresource.Manager

func setup(t *testing.T) {
	if fullTestManager != nil {
		return
	}

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		t.Fatal("error loading import graph: " + err.Error())
	}

	if err = pythonskeletons.UpdateGraph(graph); err != nil {
		t.Fatal("error updating graph from skeletons: " + err.Error())
	}

	var errc <-chan error
	fullTestManager, errc = pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
}

func TestAssembler_Skeletons(t *testing.T) {
	if !skeletonTests {
		t.Skip("skipping TestAssembler_Skeletons, run go test -skeletons to run this test")
		return
	}
	setup(t)
	src := `
from django.contrib.contenttypes.models import ContentType
ct = ContentType.objects.get_for_model(None)
`
	ct, err := fullTestManager.PathSymbol(pythonimports.NewPath("django", "contrib", "contenttypes", "models", "ContentType"))
	require.NotNil(t, ct)
	require.NoError(t, err)

	assertTypes(t, src, fullTestManager, map[string]pythontype.Value{
		"ct": pythontype.ExternalInstance{TypeExternal: pythontype.NewExternal(ct, fullTestManager)},
	})
}
