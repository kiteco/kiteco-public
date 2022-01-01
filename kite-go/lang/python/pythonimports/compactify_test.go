package pythonimports

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompactifyNodes(t *testing.T) {
	foo := NewNode("foo", Type)
	bar := NewNode("bar", Type)
	ham := NewNode("ham", Object)
	foo.Members["x"] = bar
	bar.Members["blah"] = ham
	bar.Bases = []*Node{foo}
	ham.Type = foo

	nodes, _ := CompactifyNodes([]*Node{foo, bar, ham})

	require.Len(t, nodes, 3)
	assert.Equal(t, "foo", nodes[0].CanonicalName.String())
	assert.Equal(t, "bar", nodes[1].CanonicalName.String())
	assert.Equal(t, "ham", nodes[2].CanonicalName.String())

	// the following are tests on ptr identity, so use == not assert.Equal
	assert.True(t, nodes[0].Members["x"] == &nodes[1])
	assert.True(t, nodes[1].Members["blah"] == &nodes[2])
	assert.True(t, nodes[2].Type == &nodes[0])
	assert.True(t, nodes[1].Bases[0] == &nodes[0])

	// check that the original nodes were not modified
	assert.True(t, foo.Members["x"] == bar)
	assert.True(t, bar.Members["blah"] == ham)
	assert.True(t, bar.Bases[0] == foo)
}
