package pythongraph

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/stretchr/testify/require"
)

type inScopeCase struct {
	node       string
	vs         []string
	stopAtFunc bool
}

func newInScopeCase(node string, stopAtFunc bool, vs ...string) inScopeCase {
	return inScopeCase{
		node:       node,
		vs:         vs,
		stopAtFunc: stopAtFunc,
	}
}

func requireNode(t *testing.T, root pythonast.Node, src, node string) pythonast.Node {
	var found pythonast.Node
	pythonast.Inspect(root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !pythonast.IsNil(found) {
			return false
		}

		buf := src[n.Begin():n.End()]

		if buf == node {
			found = n
		}

		return true
	})

	require.False(t, pythonast.IsNil(found))
	return found
}

func assertInScopeAt(t *testing.T, src string, cases ...inScopeCase) {
	fmtstr := `
--------- Case %d ------------
At: %s
Expected: %s
Actual: %s	
	`

	type miss struct {
		expected string
		actual   string
		idx      int
	}

	b := requireBuilder(t, emptyRM(t), src)
	varsToString := func(vs []*variable) string {
		var parts []string
		for _, variable := range vs {
			parts = append(parts, fmt.Sprintf("%s (%d)", variable.Origin.Ident.Literal, variable.ID))
		}

		return strings.Join(parts, ", ")
	}

	for i, c := range cases {
		at := requireNode(t, b.a.RAST.Root, src, c.node)

		vs := b.vm.InScope(at, c.stopAtFunc)

		if len(vs) != len(c.vs) {
			t.Errorf(fmtstr, i, c.node, strings.Join(c.vs, ", "), varsToString(vs))
			t.Errorf("expected %d variables, got %d\n", len(c.vs), len(vs))
			continue
		}

		var misses []miss
		for j, expected := range c.vs {
			actual := vs[j]
			if expected != actual.Origin.Ident.Literal {
				misses = append(misses, miss{
					expected: expected,
					actual:   actual.Origin.Ident.Literal,
					idx:      j,
				})
			}
		}

		if len(misses) > 0 {
			t.Errorf(fmtstr, i, c.node, strings.Join(c.vs, ", "), varsToString(vs))
			for _, miss := range misses {
				t.Errorf("%s != %s for variable %d\n", miss.expected, miss.actual, miss.idx)
			}
		}

	}

}

func TestInScopeAtBasic(t *testing.T) {
	src := `
import numpy as np

foo()
x = 1
bar()
`

	assertInScopeAt(t, src,
		newInScopeCase("foo()", false, "np"),
		newInScopeCase("bar()", false, "np", "foo", "x"),
	)
}

func TestInScopeAtIfStmt(t *testing.T) {
	src := `
x = 1
if 1:
	y = 2
	one
else:
	z = 3
	two

three
`

	assertInScopeAt(t, src,
		newInScopeCase("one", false, "x", "y"),
		newInScopeCase("two", false, "x", "z"),
		newInScopeCase("three", false, "x", "y", "one", "z", "two"),
	)
}

func TestInScopeAtFuncDef(t *testing.T) {
	src := `
x = 1
def foo(y, (z,)):
	q = 1
	one

two
	`

	assertInScopeAt(t, src,
		newInScopeCase("one", true, "y", "z", "q"),
		newInScopeCase("two", true, "x", "foo"),
	)
}

func TestInScopeAtImportFrom(t *testing.T) {
	src := `
from foo import *
one
	`

	assertInScopeAt(t, src, newInScopeCase("one", false))
}

func TestInScopeAtForStmt(t *testing.T) {
	src := `
x = 1
for y in []:
	z = 2
	one
else:
	w = 3
	two
three
`

	assertInScopeAt(t, src,
		newInScopeCase("one", false, "x", "y", "z"),
		newInScopeCase("two", false, "x", "y", "z", "one", "w"),
		newInScopeCase("three", false, "x", "y", "z", "one", "w", "two"),
	)
}

func TestInScopeAtImportNameStmt(t *testing.T) {
	src := `
import os.path, json, numpy as np
x
	`

	assertInScopeAt(t, src,
		newInScopeCase("x", false, "os", "json", "np"),
	)
}

func TestInScopeAtImportFromStmt(t *testing.T) {
	src := `
from json import dump, dumps
def foo():
	x
	`

	assertInScopeAt(t, src,
		newInScopeCase("x", false, "dump", "dumps"),
	)
}
