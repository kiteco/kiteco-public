package pythonast

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/stretchr/testify/assert"
)

func newWord(s string) *pythonscanner.Word {
	return &pythonscanner.Word{
		Literal: s,
	}
}

func newName(s string) *NameExpr {
	return &NameExpr{
		Ident: newWord(s),
	}
}

var (
	// create an expression like (a + (b + c))
	a     = newName("a")
	b     = newName("b")
	c     = newName("b")
	inner = &BinaryExpr{
		Left:  b,
		Right: c,
	}
	outer = &BinaryExpr{
		Left:  a,
		Right: inner,
	}
)

func TestInspect(t *testing.T) {
	type edge struct {
		parent, child Node
		field         string
	}
	expected := []Node{
		outer,
		a,
		nil, // closes "a"
		inner,
		b,
		nil, // closes "b"
		c,
		nil, // closes "c"
		nil, // closes "inner"
		nil, // closes "outer"
	}

	var actual []Node
	Inspect(outer, func(n Node) bool {
		actual = append(actual, n)
		return true
	})

	assert.Equal(t, expected, actual)
}

func TestInspectEdges(t *testing.T) {
	type edge struct {
		parent, child Node
		field         string
	}
	expected := []edge{
		edge{nil, outer, ""},
		edge{outer, a, "Left"},
		edge{a, nil, ""},
		edge{outer, inner, "Right"},
		edge{inner, b, "Left"},
		edge{b, nil, ""},
		edge{inner, c, "Right"},
		edge{c, nil, ""},
		edge{inner, nil, ""},
		edge{outer, nil, ""},
	}

	var actual []edge
	InspectEdges(outer, func(parent, child Node, field string) bool {
		actual = append(actual, edge{parent, child, field})
		return true
	})

	assert.Equal(t, expected, actual)
}
