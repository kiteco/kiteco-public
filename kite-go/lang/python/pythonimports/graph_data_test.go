package pythonimports

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file defines tests for the content of the import graph.
// It is meant to be a last-line-of-defence check on the
// integrity of the import graph. You can run these tests with
//    go test --graphdata

var (
	enableDataTests  bool
	graphData        *Graph
	graphStringsData GraphStrings
	argSpecsData     *ArgSpecs
)

func init() {
	flag.BoolVar(&enableDataTests, "graphdata", false, "enable import graph data tests")
}

type nameKind struct {
	Name string
	Kind Kind
}

func loadCurrentGraph(t *testing.T) *Graph {
	if graphData == nil {
		var err error
		graphData, err = NewGraph(DefaultImportGraph)
		require.NoError(t, err)
	}
	return graphData
}

func loadCurrentGraphStrings(t *testing.T) GraphStrings {
	if graphStringsData == nil {
		var err error
		graphStringsData, err = LoadGraphStrings(DefaultImportGraphStrings)
		require.NoError(t, err)
	}
	return graphStringsData
}

func loadCurrentArgSpecs(t *testing.T) *ArgSpecs {
	if argSpecsData == nil {
		var err error
		argSpecsData, err = LoadArgSpecs(graph, DefaultImportGraphArgSpecs, DefaultTypeshedArgSpecs)
		require.NoError(t, err)
	}
	return argSpecsData
}

func TestImportGraphData_InitMember(t *testing.T) {
	if !enableDataTests {
		t.Skip("use --graphdata to enable import graph data tests")
	}

	graph := loadCurrentGraph(t)

	root, err := graph.Find("json.decoder.JSONDecoder")
	require.Nil(t, err, "node for json.decoder.JSONDecoder not found")

	_, ok := root.Members["__init__"]
	assert.True(t, ok, "json.decoder.JSONDecoder should have the __init__ member")
}

func TestImportGraphData_JSONClassification(t *testing.T) {
	if !enableDataTests {
		t.Skip("use --graphdata to enable import graph data tests")
	}

	graph := loadCurrentGraph(t)

	root, err := graph.Find("json")
	require.Nil(t, err, "node for json not found")

	jsonEncoder, ok := root.Members["JSONEncoder"]
	require.True(t, ok, "node for JSONEncoder member not found")
	assert.Equal(t, Type, jsonEncoder.Classification, "type of JSONEncoder should be type")

	jsonDecoder, ok := root.Members["JSONDecoder"]
	require.True(t, ok, "node for JSONDecoder member not found")
	assert.Equal(t, Type, jsonDecoder.Classification, "type of JSONDecoder should be type")

	defaultEncoder, ok := root.Members["_default_encoder"]
	require.True(t, ok, "node for _default_encoder member not found")
	assert.Equal(t, Object, defaultEncoder.Classification, "type of _default_encoder should be object")

	defaultDecoder, ok := root.Members["_default_decoder"]
	require.True(t, ok, "node for _default_decoder member not found")
	assert.Equal(t, Object, defaultDecoder.Classification, "type of _default_decoder should be type")
}

func TestImportGraphData_NamedNodes(t *testing.T) {
	if !enableDataTests {
		t.Skip("use --graphdata to enable import graph data tests")
	}

	expected := map[string]nameKind{
		"builtins":                     {"builtins", Module},
		"builtins.str":                 {"builtins.str", Type},
		"builtins.str.join":            {"builtins.str.join", Function},
		"types":                        {"types", Module},
		"types.FileType":               {"io.IOBase", Type},
		"exceptions":                   {"exceptions", Module},
		"exceptions.Exception":         {"exceptions.Exception", Type},
		"datetime.datetime":            {"datetime.datetime", Type},
		"datetime.datetime.time":       {"datetime.datetime.time", Function},
		"numpy":                        {"numpy", Module},
		"numpy.ndarray":                {"numpy.ndarray", Type},
		"numpy.ndarray.T":              {"numpy.ndarray.T", Descriptor},
		"numpy.zeros":                  {"numpy.core.multiarray.zeros", Function},
		"flask.request.content_length": {"", Object},
	}

	graph := loadCurrentGraph(t)
	for cn, expect := range expected {
		n, err := graph.Find(cn)
		if !assert.NoError(t, err) {
			continue
		}
		if !n.CanonicalName.Equals(expect.Name) {
			t.Errorf("found node for %s but its name was '%s' (expected '%s')",
				cn, n.CanonicalName.String(), expect.Name)
			continue
		}
		if n.Classification != expect.Kind {
			t.Errorf("found node for %s but its kind was %s (expected %s)",
				cn, n.Classification, expect.Kind)
		}
	}
}

func TestImportGraphData_GraphStrings(t *testing.T) {
	if !enableDataTests {
		t.Skip("use --graphdata to enable import graph data tests")
	}

	graph := loadCurrentGraph(t)
	ids := make(map[int64]struct{})
	for _, node := range graph.Nodes {
		ids[node.ID] = struct{}{}
	}

	var missingIDs []int64
	argspecs := loadCurrentArgSpecs(t).ImportGraphArgSpecs
	for id := range argspecs {
		if _, found := ids[id]; !found {
			missingIDs = append(missingIDs, id)
		}
	}
	t.Logf("%d of %d arg specs IDs were not in the graph", len(missingIDs), len(argspecs))
	if 10*len(missingIDs) > 9*len(argspecs) {
		t.Errorf("%d of %d arg specs IDs were not in the graph", len(missingIDs), len(argspecs))
	}

	missingIDs = []int64{}
	graphstrings := loadCurrentGraphStrings(t)
	for id := range graphstrings {
		if _, found := ids[id]; !found {
			missingIDs = append(missingIDs, id)
		}
	}
	t.Logf("%d of %d graph strings IDs were not in the graph", len(missingIDs), len(graphstrings))
	if 10*len(missingIDs) > 9*len(graphstrings) {
		t.Errorf("%d of %d graph strings IDs were not in the graph", len(missingIDs), len(graphstrings))
	}
}
