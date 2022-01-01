package dynamicanalysis

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceReferences(t *testing.T) {
	if !dockerTests {
		t.Skip("use --docker to run tests that require docker")
	}

	src := `
import collections
import sys as x
v = x.version
d = collections.deque()
d.appendleft(1)
`

	expected := map[string]string{
		"x":           "sys",
		"version":     "sys.version",
		"v":           "builtins.str",
		"collections": "collections",
		"deque":       "collections.deque",
		"d":           "collections.deque",
		"appendleft":  "collections.deque.appendleft",
	}

	refs, _, err := TraceReferences(src, DefaultTraceOptions)
	require.NoError(t, err)

	for _, ref := range refs {
		expr := src[ref.Begin:ref.End]
		assert.Equal(t, expr, ref.Original)
		assert.Equal(t, expected[expr], ref.FullyQualifiedName)
	}
}

func TestTraceGetSetDelAttr(t *testing.T) {
	if !dockerTests {
		t.Skip("use --docker to run tests that require docker")
	}

	src := `
import argparse
ns = argparse.Namespace()
ns.foo = 1
x = ns.foo
del ns.foo
`

	expected := map[string]string{
		"argparse":  "argparse",
		"Namespace": "argparse.Namespace",
		"ns":        "argparse.Namespace",
		"foo":       "argparse.Namespace.foo",
		"x":         "builtins.int",
	}

	refs, _, err := TraceReferences(src, DefaultTraceOptions)
	require.NoError(t, err)

	for _, ref := range refs {
		log.Printf("%#v", ref)
		expr := src[ref.Begin:ref.End]
		assert.Equal(t, expr, ref.Original)
		assert.Equal(t, expected[expr], ref.FullyQualifiedName, "for expression '%s'", expr)
	}
}
