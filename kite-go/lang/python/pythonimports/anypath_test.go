package pythonimports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeAnyPaths(t *testing.T) {
	g := MockGraph("foo", "foo.bar", "ham", "ham.spam", "ham.spam.bam")

	anypaths := ComputeAnyPaths(g)
	for node, apath := range anypaths {
		// use assert.True here not assert.Equal because we want to compare pointer values
		p, err := g.Navigate(apath)
		if assert.NoError(t, err) {
			assert.True(t, node == p, "anypath for %s was %s, which resolved to %s",
				node.String(), apath.String(), p.String())
		}
	}
}

func TestVerifyCanonicalName(t *testing.T) {
	g := MockGraph("foo", "foo.bar")
	node := NewNode("foo.bar", Function)
	assert.False(t, verifyCanonicalName(node, g), "should be a dup")

	node, err := g.Find("foo.bar")
	assert.Nil(t, err, "should have found foo.bar")
	assert.True(t, verifyCanonicalName(node, g), "should not be a dup")
}
