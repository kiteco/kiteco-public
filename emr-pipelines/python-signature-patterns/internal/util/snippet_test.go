package util

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/stretchr/testify/assert"
)

// --

var (
	emptyGraph = pythonimports.MockGraph()

	emptyParams = Params{
		Graph:       emptyGraph,
		ArgSpecs:    make(pythonimports.ArgSpecs),
		TypeInducer: typeinduction.MockClient(emptyGraph, map[string]string{}),
		AnyNames:    pythonimports.ComputeAnyPaths(emptyGraph),
	}
)

// --

func assertCallSpec(t *testing.T, graph *pythonimports.Graph, actual *CallSpec, expected *CallSpec) {
	node, err := graph.Find(expected.AnyName.String())
	if err != nil {
		t.Errorf("\nNo graph node found for call %s", expected.AnyName)
	} else if node.CanonicalName.Hash != actual.AnyName.Hash {
		t.Errorf("\nGraph node name mismatch. Expected: %s. Actual: %s.",
			node.CanonicalName.String(), actual.AnyName)
	}

	if expected.Code != actual.Code {
		t.Errorf("Expected:\n%s\nActual:\n%s", expected.Code, actual.Code)
	}

	if len(actual.Args) != len(expected.Args) {
		t.Errorf("\nExpected:\n%s\nActual:\n%s", expected.String(), actual.String())
	} else {
		for i := range actual.Args {
			if *actual.Args[i] != *expected.Args[i] {
				t.Errorf("\nExpected: {%s}\nActual: {%s}", expected.Args[i].String(), actual.Args[i].String())
			}
		}
	}

	if len(actual.Kwargs) != len(expected.Kwargs) {
		t.Errorf("\nExpected:\n%s\nActual:\n%s", expected.String(), actual.String())
	} else {
		for i := range actual.Kwargs {
			if *expected.Kwargs[i] != *actual.Kwargs[i] {
				t.Errorf("\nExpected: {%s}\nActual: {%s}", expected.Kwargs[i].String(), actual.Kwargs[i].String())
			}
		}
	}
}

// Note: this cannot handle the case in which to separate calls have the same Code
// e.g foo(1); foo(1)
func assertSnippets(t *testing.T, graph *pythonimports.Graph, typeInducer *typeinduction.Client,
	src string, expected map[string]*CallSpec) {

	snippet := Extract([]byte(src), Params{
		Graph:       graph,
		TypeInducer: typeInducer,
		AnyNames:    pythonimports.ComputeAnyPaths(graph),
		ArgSpecs:    make(pythonimports.ArgSpecs),
	})

	if snippet == nil {
		t.Errorf("\nExpected non nil snippet for src:\n%s", src)
		return
	}

	for _, spec := range snippet.Incantations {
		exp := expected[spec.Code]
		if exp == nil {
			continue
		}
		assertCallSpec(t, graph, spec, exp)
	}

	for _, spec := range snippet.Decorators {
		exp := expected[spec.Code]
		if exp == nil {
			continue
		}
		assertCallSpec(t, graph, spec, exp)
	}

	var missing []*CallSpec
Missing:
	for call, spec := range expected {
		for _, cs := range snippet.Incantations {
			if cs.Code == call {
				continue Missing
			}
		}
		for _, cs := range snippet.Decorators {
			if cs.Code == call {
				continue Missing
			}
		}
		missing = append(missing, spec)
	}

	for _, spec := range missing {
		t.Errorf("\nMissing expected spec\n%s", spec.String())
	}

	var extra []*CallSpec
	for _, cs := range snippet.Incantations {
		if _, found := expected[cs.Code]; found {
			continue
		}
		extra = append(extra, cs)
	}
	for _, cs := range snippet.Decorators {
		if _, found := expected[cs.Code]; found {
			continue
		}
		extra = append(extra, cs)
	}

	for _, spec := range extra {
		t.Errorf("\nExtra actual spec\n%s", spec.String())
	}
}

func assertSnippet(t *testing.T, graph *pythonimports.Graph, typeInducer *typeinduction.Client,
	src string, expected *CallSpec) {

	assertSnippets(t, graph, typeInducer, src, map[string]*CallSpec{
		expected.Code: expected,
	})
}

func argSpec(key, exprStr, typ, lit string) *pythoncode.ArgSpec {
	return &pythoncode.ArgSpec{
		Key:     key,
		ExprStr: exprStr,
		Type:    typ,
		Literal: lit,
	}
}

// --

func TestSnippet(t *testing.T) {
	src := `
import foo
from json import dumps
from os import path

w = "world"
x = path.join("hello", w)
y = dumps(q = {"a": 1, "b": 2})

@foo()
def car():
	pass
`

	graph := pythonimports.MockGraph("json.dumps",
		"os.path.join", "__builtin__.str", "foo",
		"__builtin__.int", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"os.path.join": "__builtin__.str",
		"json.dumps":   "__builtin__.str",
		"foo":          "__builtin__.None",
	})

	expected := map[string]*CallSpec{
		`path.join("hello", w)`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("os.path.join"),
			Code:    `path.join("hello", w)`,
			Args: []*pythoncode.ArgSpec{
				argSpec("", "", "__builtin__.str", `"hello"`),
				argSpec("", "w", "__builtin__.str", ""),
			},
		},
		`dumps(q = {"a": 1, "b": 2})`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("json.dumps"),
			Code:    `dumps(q = {"a": 1, "b": 2})`,
			Kwargs: []*pythoncode.ArgSpec{
				argSpec("q", "", "__builtin__.dict", `{"a": 1, "b": 2}`),
			},
		},
		"foo()": &CallSpec{
			AnyName: pythonimports.NewDottedPath("foo"),
			Code:    "foo()",
		},
	}

	assertSnippets(t, graph, typeInducer, src, expected)
}

func TestSnippetDecorators(t *testing.T) {
	src := `
import os
import testmodule

@testmodule.foo
@testmodule.bar("hello")
@testmodule.baz(foo=1)
def test():
    pass
 `

	graph := pythonimports.MockGraph("__builtin__.str",
		"testmodule", "testmodule.foo", "testmodule.bar",
		"testmodule.baz", "__builtin__.int", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"os.path.join":   "__builtin__.str",
		"testmodule.bar": "__builtin__.None",
	})

	expected := map[string]*CallSpec{
		`testmodule.bar("hello")`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("testmodule.bar"),
			Code:    `testmodule.bar("hello")`,
			Args: []*pythoncode.ArgSpec{
				argSpec("", "", "__builtin__.str", `"hello"`),
			},
		},
		`testmodule.baz(foo=1)`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("testmodule.baz"),
			Code:    `testmodule.baz(foo=1)`,
			Kwargs: []*pythoncode.ArgSpec{
				argSpec("foo", "", "__builtin__.int", `1`),
			},
		},
	}

	assertSnippets(t, graph, typeInducer, src, expected)
}

func TestSnippetLambda(t *testing.T) {
	src := `
import foo
import bar

@bar(q=lambda z: z)
@bar(lambda y: y)
def car():
	pass
foo(lambda x: x)
 `

	graph := pythonimports.MockGraph("foo", "bar", "__builtin__.None", "__builtin__.str")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
		"bar": "__builtin__.None",
	})

	expected := map[string]*CallSpec{
		`foo(lambda x: x)`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("foo"),
			Code:    `foo(lambda x: x)`,
			Args: []*pythoncode.ArgSpec{
				argSpec("", "", "__builtin__.function", `lambda x: x`),
			},
		},
		`bar(lambda y: y)`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("bar"),
			Code:    `bar(lambda y: y)`,
			Args: []*pythoncode.ArgSpec{
				argSpec("", "", "__builtin__.function", `lambda y: y`),
			},
		},
		`bar(q=lambda z: z)`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("bar"),
			Code:    `bar(q=lambda z: z)`,
			Kwargs: []*pythoncode.ArgSpec{
				argSpec("q", "", "__builtin__.function", `lambda z: z`),
			},
		},
	}

	assertSnippets(t, graph, typeInducer, src, expected)
}

func TestSnippetNested(t *testing.T) {
	src := `
import foo
import bar
import car

foo(bar(car()))
foo(w=car())
 `

	graph := pythonimports.MockGraph("foo", "bar", "car",
		"__builtin__.None", "__builtin__.str")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
		"bar": "__builtin__.str",
		"car": "__builtin__.str",
	})

	expected := map[string]*CallSpec{
		`foo(bar(car()))`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("foo"),
			Code:    `foo(bar(car()))`,
			Args: []*pythoncode.ArgSpec{
				argSpec("", "", "__builtin__.str", ""),
			},
		},
		`bar(car())`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("bar"),
			Code:    `bar(car())`,
			Args: []*pythoncode.ArgSpec{
				argSpec("", "", "__builtin__.str", ""),
			},
		},
		`car()`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("car"),
			Code:    `car()`,
		},
		`foo(w=car())`: &CallSpec{
			AnyName: pythonimports.NewDottedPath("foo"),
			Code:    `foo(w=car())`,
			Kwargs: []*pythoncode.ArgSpec{
				argSpec("w", "", "__builtin__.str", ""),
			},
		},
	}

	assertSnippets(t, graph, typeInducer, src, expected)
}

// --

func TestLiteralNumber(t *testing.T) {
	src := `
import foo

foo(1)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(1)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.int", "1"),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralString(t *testing.T) {
	src := `
import foo

foo("hello")
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo("hello")`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.str", `"hello"`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralTuple(t *testing.T) {
	src := `
import foo

foo((a,b))
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo((a,b))`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.tuple", `(a,b)`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralList(t *testing.T) {
	src := `
import foo

foo([a,b])
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo([a,b])`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.list", `[a,b]`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralSet(t *testing.T) {
	src := `
import foo

foo({a,b})
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo({a,b})`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.set", `{a,b}`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralDict(t *testing.T) {
	src := `
import foo

foo({a:1})
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo({a:1})`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.dict", `{a:1}`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralGenerator(t *testing.T) {
	src := `
import foo

foo(a for a in b)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(a for a in b)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "types.GeneratorType", `a for a in b`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralListComprehension(t *testing.T) {
	src := `
import foo

foo([a for a in b])
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo([a for a in b])`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.list", `[a for a in b]`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralDictComprehension(t *testing.T) {
	src := `
import foo

foo({a:1 for a in b})
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo({a:1 for a in b})`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.dict", `{a:1 for a in b}`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralSetComprehension(t *testing.T) {
	src := `
import foo

foo({a for a in b})
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo({a for a in b})`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.set", `{a for a in b}`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralLambda(t *testing.T) {
	src := `
import foo

foo(lambda a: a)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(lambda a: a)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.function", `lambda a: a`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestLiteralYield(t *testing.T) {
	src := `
import foo

foo(yield a)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	snippet := Extract([]byte(src), Params{
		Graph:       graph,
		TypeInducer: typeInducer,
		AnyNames:    pythonimports.ComputeAnyPaths(graph),
		ArgSpecs:    make(pythonimports.ArgSpecs),
	})

	assert.Nil(t, snippet)
}

func TestLiteralIfExpr(t *testing.T) {
	src := `
import foo
a = "hello"
b = True
c = "world"
foo(a if b else c)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(a if b else c)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.str", `a if b else c`),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestAddNumberLiteral(t *testing.T) {
	src := `
import foo
foo(1 + 2 + 3)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None", "__builtin__.int")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(1 + 2 + 3)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.int", ``),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestPowNumberLiteral(t *testing.T) {
	src := `
import foo
foo(1**3)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None", "__builtin__.int")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(1**3)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.int", ``),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestAddNumber(t *testing.T) {
	src := `
import foo
a = 1
b = 2
foo(a + b)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None", "__builtin__.int")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(a + b)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.int", ``),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestConcatStringLiteral(t *testing.T) {
	src := `
import foo
foo("hello" + "world")
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None", "__builtin__.str")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo("hello" + "world")`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.str", ``),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestConcatString(t *testing.T) {
	src := `
import foo
a = "hello"
b = "world"
foo(a + b)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None", "__builtin__.str")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(a + b)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.str", ``),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestIn(t *testing.T) {
	src := `
import foo
foo(1 in [1,2])
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None", "__builtin__.int")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(1 in [1,2])`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.bool", ``),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestIs(t *testing.T) {
	src := `
import foo
foo(1 is 2)
	`

	graph := pythonimports.MockGraph("foo", "__builtin__.None", "__builtin__.int")

	typeInducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    `foo(1 is 2)`,
		Args: []*pythoncode.ArgSpec{
			argSpec("", "", "__builtin__.bool", ``),
		},
	}

	assertSnippet(t, graph, typeInducer, src, expected)
}

func TestIncludeUnresolvedArgs(t *testing.T) {
	src := `
import foo
foo(x, bar=y)
`
	graph := pythonimports.MockGraph("foo")

	typeinducer := typeinduction.MockClient(graph, map[string]string{
		"foo": "__builtin__.None",
	})

	expected := &CallSpec{
		AnyName: pythonimports.NewDottedPath("foo"),
		Code:    "foo(x, bar=y)",
		Args: []*pythoncode.ArgSpec{
			argSpec("", "x", "", ""),
		},
		Kwargs: []*pythoncode.ArgSpec{
			argSpec("bar", "y", "", ""),
		},
	}

	assertSnippet(t, graph, typeinducer, src, expected)
}

func TestNoLocalCalls(t *testing.T) {
	src := `
def foo(i):
    pass

foo(1)
 
@foo(2)
def bar():
    pass
    `

	snippet := Extract([]byte(src), emptyParams)

	assert.Nil(t, snippet)
}

func TestNoMod(t *testing.T) {
	snippet := Extract([]byte("\n"), emptyParams)
	assert.Nil(t, snippet)
}
