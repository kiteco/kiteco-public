package calls

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/calls/internal/pigeon"
	"github.com/stretchr/testify/require"
)

func assertParseEntrypoint(t *testing.T, entrypoint, expected, src string) {
	t.Log(src)
	node, err := pigeon.Parse("", []byte(src), pigeon.Entrypoint(entrypoint))
	require.NoError(t, err)
	require.NotNil(t, node)
	assertAST(t, expected, node.(pythonast.Node), false)
}

func TestTupleEmpty(t *testing.T) {
	src := `()`

	expected := `
TupleExpr
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleSingleValue(t *testing.T) {
	src := `(1,)`

	expected := `
TupleExpr
	NumberExpr[1]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleTwoValues(t *testing.T) {
	src := `(1, 2)`

	expected := `
TupleExpr
	NumberExpr[1]
	NumberExpr[2]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleTrailingComma(t *testing.T) {
	src := `(1, 2, )`

	expected := `
TupleExpr
	NumberExpr[1]
	NumberExpr[2]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleManyEmptyNested(t *testing.T) {
	src := `((()))`

	// because the initial empty tuple contains only a single
	// expression (another empty tuple) with no comma, the nested
	// empty tuples are flattened in a single empty tuple.
	expected := `
TupleExpr
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleOfEmptyTuple(t *testing.T) {
	// see SO link: /a/37895685/1094941
	// the comma is required, as for any single-value tuple
	src := `((),)`
	expected := `
TupleExpr
	TupleExpr
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleNestedWithElements(t *testing.T) {
	src := `((("a", ), "b"), "c")`

	expected := `
TupleExpr
	TupleExpr
		TupleExpr
			StringExpr["a"]
		StringExpr["b"]
	StringExpr["c"]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleWithMixedElements(t *testing.T) {
	src := `(a, [b, c], {d: "e"}, {f}, (g,))`

	expected := `
TupleExpr
	NameExpr[a]
	ListExpr
		NameExpr[b]
		NameExpr[c]
	DictExpr
		KeyValuePair
			NameExpr[d]
			StringExpr["e"]
	SetExpr
		NameExpr[f]
	TupleExpr
		NameExpr[g]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleSingleValueNoComma(t *testing.T) {
	src := `(1)`

	// returns the inner expression, not a tuple
	expected := `
NumberExpr[1]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleStarExpr1(t *testing.T) {
	src := `(*a)`
	expected := `
UnaryExpr[*]
	NameExpr[a]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleStarExpr2(t *testing.T) {
	src := `(*a,)`
	expected := `
TupleExpr
	UnaryExpr[*]
		NameExpr[a]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleStarExpr3(t *testing.T) {
	src := `(1, "x", *a, false)`
	expected := `
TupleExpr
	NumberExpr[1]
	StringExpr["x"]
	UnaryExpr[*]
		NameExpr[a]
	NameExpr[false]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleStarExpr4(t *testing.T) {
	src := `(1, *fn(x), 2)`
	expected := `
TupleExpr
	NumberExpr[1]
	UnaryExpr[*]
		CallExpr
			NameExpr[fn]
			Argument
				NameExpr[x]
	NumberExpr[2]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestTupleStarExpr5(t *testing.T) {
	// technically invalid but we want to be loose in what we accept.
	src := `(*x, **y, *z)`
	expected := `
TupleExpr
	UnaryExpr[*]
		NameExpr[x]
	UnaryExpr[**]
		NameExpr[y]
	UnaryExpr[*]
		NameExpr[z]
`
	assertParseEntrypoint(t, "TestTuple", expected, src)
}

func TestListEmpty(t *testing.T) {
	src := "[]"
	expected := `
ListExpr
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListSingleName(t *testing.T) {
	src := "[a]"
	expected := `
ListExpr
	NameExpr[a]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListOfTwo(t *testing.T) {
	src := "[a, 1]"
	expected := `
ListExpr
	NameExpr[a]
	NumberExpr[1]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListTrailingComma(t *testing.T) {
	src := "[a, 1 ,]"
	expected := `
ListExpr
	NameExpr[a]
	NumberExpr[1]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListOfMany(t *testing.T) {
	src := `[a, 1, "b", r"c"]`
	expected := `
ListExpr
	NameExpr[a]
	NumberExpr[1]
	StringExpr["b"]
	StringExpr[r"c"]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListManyEmptyNested(t *testing.T) {
	src := `[[[]]]`
	expected := `
ListExpr
	ListExpr
		ListExpr
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListNestedWithElements(t *testing.T) {
	src := `[[[a], b], c]`
	expected := `
ListExpr
	ListExpr
		ListExpr
			NameExpr[a]
		NameExpr[b]
	NameExpr[c]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListMixedElements(t *testing.T) {
	src := `[a, "b", 1.3, ['inner'], (true, 2,), {set}, {dict: true}]`
	expected := `
ListExpr
	NameExpr[a]
	StringExpr["b"]
	NumberExpr[1.3]
	ListExpr
		StringExpr['inner']
	TupleExpr
		NameExpr[true]
		NumberExpr[2]
	SetExpr
		NameExpr[set]
	DictExpr
		KeyValuePair
			NameExpr[dict]
			NameExpr[true]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListStarExpr1(t *testing.T) {
	src := "[*a]"
	expected := `
ListExpr
	UnaryExpr[*]
		NameExpr[a]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListStarExpr2(t *testing.T) {
	src := `[*a,]`
	expected := `
ListExpr
	UnaryExpr[*]
		NameExpr[a]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListStarExpr3(t *testing.T) {
	src := `[1, "x", *a, false]`
	expected := `
ListExpr
	NumberExpr[1]
	StringExpr["x"]
	UnaryExpr[*]
		NameExpr[a]
	NameExpr[false]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListStarExpr4(t *testing.T) {
	src := `[1, *fn(x), 2]`
	expected := `
ListExpr
	NumberExpr[1]
	UnaryExpr[*]
		CallExpr
			NameExpr[fn]
			Argument
				NameExpr[x]
	NumberExpr[2]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestListStarExpr5(t *testing.T) {
	// technically invalid but we want to be loose in what we accept.
	src := `[*x, **y, *z]`
	expected := `
ListExpr
	UnaryExpr[*]
		NameExpr[x]
	UnaryExpr[**]
		NameExpr[y]
	UnaryExpr[*]
		NameExpr[z]
`
	assertParseEntrypoint(t, "TestList", expected, src)
}

func TestSetSingleItem(t *testing.T) {
	src := `{a}`
	expected := `
SetExpr
	NameExpr[a]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetTwoItems(t *testing.T) {
	src := `{a, 1}`
	expected := `
SetExpr
	NameExpr[a]
	NumberExpr[1]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetNested(t *testing.T) {
	src := `{ {{a}, b}, c, }`
	expected := `
SetExpr
	SetExpr
		SetExpr
			NameExpr[a]
		NameExpr[b]
	NameExpr[c]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetMixedElements(t *testing.T) {
	src := `{a, 0xabc, {"inner", r"set", }, ['list', 2], (t, 3), {key: "value", other: 42,}}`
	expected := `
SetExpr
	NameExpr[a]
	NumberExpr[0xabc]
	SetExpr
		StringExpr["inner"]
		StringExpr[r"set"]
	ListExpr
		StringExpr['list']
		NumberExpr[2]
	TupleExpr
		NameExpr[t]
		NumberExpr[3]
	DictExpr
		KeyValuePair
			NameExpr[key]
			StringExpr["value"]
		KeyValuePair
			NameExpr[other]
			NumberExpr[42]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetStarExpr1(t *testing.T) {
	src := "{*a}"
	expected := `
SetExpr
	UnaryExpr[*]
		NameExpr[a]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetStarExpr2(t *testing.T) {
	src := `{*a,}`
	expected := `
SetExpr
	UnaryExpr[*]
		NameExpr[a]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetStarExpr3(t *testing.T) {
	src := `{1, "x", *a, false}`
	expected := `
SetExpr
	NumberExpr[1]
	StringExpr["x"]
	UnaryExpr[*]
		NameExpr[a]
	NameExpr[false]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetStarExpr4(t *testing.T) {
	src := `{1, *fn(x), 2}`
	expected := `
SetExpr
	NumberExpr[1]
	UnaryExpr[*]
		CallExpr
			NameExpr[fn]
			Argument
				NameExpr[x]
	NumberExpr[2]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestSetStarExpr5(t *testing.T) {
	// technically invalid but we want to be loose in what we accept.
	src := `{*x, **y, *z}`
	expected := `
SetExpr
	UnaryExpr[*]
		NameExpr[x]
	UnaryExpr[**]
		NameExpr[y]
	UnaryExpr[*]
		NameExpr[z]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictEmpty(t *testing.T) {
	src := `{}`
	expected := `
DictExpr
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictSinglePair(t *testing.T) {
	src := `{a : b}`
	expected := `
DictExpr
	KeyValuePair
		NameExpr[a]
		NameExpr[b]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictTwoPairs(t *testing.T) {
	src := `{a: b, "c": 3}`
	expected := `
DictExpr
	KeyValuePair
		NameExpr[a]
		NameExpr[b]
	KeyValuePair
		StringExpr["c"]
		NumberExpr[3]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictNested(t *testing.T) {
	src := `{a: {b: {c: v}, d: e}, f: g, }`
	expected := `
DictExpr
	KeyValuePair
		NameExpr[a]
		DictExpr
			KeyValuePair
				NameExpr[b]
				DictExpr
					KeyValuePair
						NameExpr[c]
						NameExpr[v]
			KeyValuePair
				NameExpr[d]
				NameExpr[e]
	KeyValuePair
		NameExpr[f]
		NameExpr[g]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictMixedElements(t *testing.T) {
	src := `{"a": 1, 3: "b", (): [c, 2, {a: b}], "d": {f}}`
	expected := `
DictExpr
	KeyValuePair
		StringExpr["a"]
		NumberExpr[1]
	KeyValuePair
		NumberExpr[3]
		StringExpr["b"]
	KeyValuePair
		TupleExpr
		ListExpr
			NameExpr[c]
			NumberExpr[2]
			DictExpr
				KeyValuePair
					NameExpr[a]
					NameExpr[b]
	KeyValuePair
		StringExpr["d"]
		SetExpr
			NameExpr[f]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictStarExpr1(t *testing.T) {
	src := "{x: *a}"
	expected := `
DictExpr
	KeyValuePair
		NameExpr[x]
		UnaryExpr[*]
			NameExpr[a]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictStarExpr2(t *testing.T) {
	src := `{x: *a,}`
	expected := `
DictExpr
	KeyValuePair
		NameExpr[x]
		UnaryExpr[*]
			NameExpr[a]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictStarExpr3(t *testing.T) {
	src := `{x: 1, y: "x", z: *a}`
	expected := `
DictExpr
	KeyValuePair
		NameExpr[x]
		NumberExpr[1]
	KeyValuePair
		NameExpr[y]
		StringExpr["x"]
	KeyValuePair
		NameExpr[z]
		UnaryExpr[*]
			NameExpr[a]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictStarExpr4(t *testing.T) {
	src := `{x: 1, y: *fn(x), z: 2}`
	expected := `
DictExpr
	KeyValuePair
		NameExpr[x]
		NumberExpr[1]
	KeyValuePair
		NameExpr[y]
		UnaryExpr[*]
			CallExpr
				NameExpr[fn]
				Argument
					NameExpr[x]
	KeyValuePair
		NameExpr[z]
		NumberExpr[2]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}

func TestDictStarExpr5(t *testing.T) {
	// TODO: this parses as Set because there's no key-value pair,
	// we could post-process sets to check if they only have kwargs
	// and if so change them to Dicts? Not sure if this matters a lot
	// for the caller of this parser?
	src := `{**y}`
	expected := `
SetExpr
	UnaryExpr[**]
		NameExpr[y]
`
	assertParseEntrypoint(t, "TestDictSet", expected, src)
}
