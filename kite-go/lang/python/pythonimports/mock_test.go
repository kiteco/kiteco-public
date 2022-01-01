package pythonimports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockGraph(t *testing.T) {
	graph := MockGraph("json.dumps")

	node, err := graph.Find("json")
	require.NoError(t, err)
	require.NotNil(t, node)

	node, err = graph.Find("builtins.list")
	require.NoError(t, err)
	require.NotNil(t, node)
}
