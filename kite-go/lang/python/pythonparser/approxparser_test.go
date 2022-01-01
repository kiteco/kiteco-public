package pythonparser

import (
	"go/token"
	"log"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --

func parseTestSnippet(snippet string) (*token.Pos, []byte) {
	parts := strings.Split(snippet, "$")
	switch len(parts) {
	case 1:
		// assume cursor at end of buffer
		cursor := token.Pos(len(snippet))
		return &cursor, []byte(snippet)
	case 2:
		cursor := token.Pos(len(parts[1]))
		return &cursor, []byte(strings.Join(parts, ""))
	default:
		log.Fatalf("invalid test snippet: %s", snippet)
		return nil, nil
	}
}

// --

func ApproxParse(src []byte, opts Options) (*pythonast.Module, error) {
	opts.Approximate = true
	return Parse(kitectx.Background(), src, opts)
}

func TestApproxParse(t *testing.T) {
	src := `
import bar.,
from foo import
class Car:
	,hp = int(0)
	def crash(self):
		self.hp = hp())
		return "you crashed!!!"
	int(str()
	`

	expected := `
Module
	BadStmt
		ImportFromStmt
			DottedExpr
				NameExpr[foo]
		ImportNameStmt
			DottedAsName
				DottedExpr
					NameExpr[bar]
					NameExpr[]
	ClassDefStmt
		NameExpr[Car]
		BadStmt
			AssignStmt
				NameExpr[hp]
				CallExpr
					NameExpr[int]
					Argument
						NumberExpr[0]
		FunctionDefStmt
			NameExpr[crash]
			Parameter
				NameExpr[self]
			BadStmt
				AssignStmt
					AttributeExpr[hp]
						NameExpr[self]
					CallExpr
						NameExpr[hp]
			ReturnStmt
				StringExpr["you crashed!!!"]
		BadStmt
			ExprStmt
				CallExpr
					NameExpr[int]
					Argument
						CallExpr
							NameExpr[str]
	`

	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

func TestApproxParse2(t *testing.T) {
	src := `
class Car:
	def repair(self, amount):
		self.hp += amount
		print("your hp is now", self.hp,,)

	def speed(self):
		return self.speed

`
	expected := `
Module
	ClassDefStmt
		NameExpr[Car]
		FunctionDefStmt
			NameExpr[repair]
			Parameter
				NameExpr[self]
			Parameter
				NameExpr[amount]
			AugAssignStmt[+=]
				AttributeExpr[hp]
					NameExpr[self]
				NameExpr[amount]
			BadStmt
				ExprStmt
					CallExpr
						NameExpr[print]
						Argument
							StringExpr["your hp is now"]
						Argument
							AttributeExpr[hp]
								NameExpr[self]
						Argument
							BadExpr
		FunctionDefStmt
			NameExpr[speed]
			Parameter
				NameExpr[self]
			ReturnStmt
				AttributeExpr[speed]
					NameExpr[self]
`

	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

func TestApproxParse3(t *testing.T) {
	src := `
class Car:
	speed = 0
	hp =
	tires.num = 4

	def accel(self, accel, time):
		foo(
		self.speed += accel * time
	`

	expected := `
Module
	ClassDefStmt
		NameExpr[Car]
		AssignStmt
			NameExpr[speed]
			NumberExpr[0]
		BadStmt
			ExprStmt
				NameExpr[hp]
			ExprStmt
				AttributeExpr[num]
					NameExpr[tires]
		FunctionDefStmt
			NameExpr[accel]
			Parameter
				NameExpr[self]
			Parameter
				NameExpr[accel]
			Parameter
				NameExpr[time]
			BadStmt
				ExprStmt
					CallExpr
						NameExpr[foo]
				ExprStmt
					AttributeExpr[speed]
						NameExpr[self]
				ExprStmt
					NameExpr[accel]
				ExprStmt
					NameExpr[time]
	`

	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

func TestApproxParse4(t *testing.T) {
	src := `
numtires = car.tires.num
for i in range(numtires):
	x = zoo(,bar)
	print(foo, bar, zar)

hello.world(!!!)
	`

	expected := `
Module
	AssignStmt
		NameExpr[numtires]
		AttributeExpr[num]
			AttributeExpr[tires]
				NameExpr[car]
	ForStmt
		NameExpr[i]
		CallExpr
			NameExpr[range]
			Argument
				NameExpr[numtires]
		BadStmt
			AssignStmt
				NameExpr[x]
				CallExpr
					NameExpr[zoo]
					Argument
						BadExpr
					Argument
						NameExpr[bar]
			ExprStmt
				CallExpr
					NameExpr[print]
					Argument
						NameExpr[foo]
					Argument
						NameExpr[bar]
					Argument
						NameExpr[zar]
	BadStmt
		ExprStmt
			CallExpr
				AttributeExpr[world]
					NameExpr[hello]
				Argument
					BadExpr
	`

	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

func TestApproxParse5(t *testing.T) {
	src := `
class foo()
bar(x,
`
	expected := `
Module
	BadStmt
		ClassDefStmt
			NameExpr[foo]
			BadStmt
		ExprStmt
			CallExpr
				NameExpr[bar]
				Argument
					NameExpr[x]
				Argument
					BadExpr
`
	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

func TestApproxParse6(t *testing.T) {
	src := `
class a()
if b()
with c()
def d()
fn(e
while f
for g in h
i = j.k()
`
	// NOTE: order depends on order of calls to approximate parsers
	expected := `
Module
	BadStmt
		AssignStmt
			NameExpr[i]
			CallExpr
				AttributeExpr[k]
					NameExpr[j]
		ClassDefStmt
			NameExpr[a]
			BadStmt
		FunctionDefStmt
			NameExpr[d]
			BadStmt
		IfStmt
			Branch
				CallExpr
					NameExpr[b]
				BadStmt
		WithStmt
			WithItem
				CallExpr
					NameExpr[c]
			BadStmt
		WhileStmt
			NameExpr[f]
			BadStmt
		ForStmt
			NameExpr[g]
			NameExpr[h]
			BadStmt
		ExprStmt
			CallExpr
				NameExpr[fn]
				Argument
					NameExpr[e]
`
	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

func TestSyncToDedent(t *testing.T) {
	src := `
class Foo(object):
	def foo(self):
		a = wrong^
	def bar(self):
		pass
	`

	expected := `
Module
	ClassDefStmt
		NameExpr[Foo]
		Argument
			NameExpr[object]
		FunctionDefStmt
			NameExpr[foo]
			Parameter
				NameExpr[self]
			BadStmt
				ExprStmt
					NameExpr[a]
				ExprStmt
					NameExpr[wrong]
		FunctionDefStmt
			NameExpr[bar]
			Parameter
				NameExpr[self]
			PassStmt
	`

	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

// --

func TestDotExprOffset(t *testing.T) {
	src :=
		`


a`
	expected := `
NameExpr[a]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src[2:]), 2)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])

	assert.EqualValues(t, 3, dotExprs[0].Begin())
	assert.EqualValues(t, 4, dotExprs[0].End())
}

func TestDotExprMultipleLineExtraDot(t *testing.T) {
	src := `
a..
b.c
	`
	expected := []string{
		`
AttributeExpr[]
	AttributeExpr[]
		NameExpr[a]
`,
		`
AttributeExpr[c]
	NameExpr[b]
`,
	}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 2)

	for i := range expected {
		assertAST(t, expected[i], dotExprs[i])
	}
}

func TestDotExprMultipleSameLine(t *testing.T) {
	src := `a.b c.d e.f`
	expected := []string{
		`
AttributeExpr[b]
	NameExpr[a]
`,
		`
AttributeExpr[d]
	NameExpr[c]
`,
		`
AttributeExpr[f]
	NameExpr[e]
`,
	}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 3)

	for i := range expected {
		assertAST(t, expected[i], dotExprs[i])
	}
}

func TestDotExprParens(t *testing.T) {
	src := `a.b()`
	expected := `
AttributeExpr[b]
	NameExpr[a]
	`
	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])
}

func TestDotExprThreeParts(t *testing.T) {
	src := `a.b.c`
	expected := `
AttributeExpr[c]
	AttributeExpr[b]
		NameExpr[a]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])
}

func TestDotExprTwoParts(t *testing.T) {
	src := `a.b`
	expected := `
AttributeExpr[b]
	NameExpr[a]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])
}

func TestDotExprNoDot(t *testing.T) {
	src := `a`
	expected := `
NameExpr[a]
	`
	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])
}

func TestDotExprsDoubleDot(t *testing.T) {
	src := `a..`
	expected := `
AttributeExpr[]
	AttributeExpr[]
		NameExpr[a]
	`
	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])
}

func TestNoDotExprsInStrings(t *testing.T) {
	src := `
"this.is.not.a.dot.expr"
this.is.a.dot.expr
	`

	expected := `
AttributeExpr[expr]
	AttributeExpr[dot]
		AttributeExpr[a]
			AttributeExpr[is]
				NameExpr[this]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])
}

func TestNoDotExprsInStrings2(t *testing.T) {
	src := `
yield ("your hp is now", self.hp,,....)
	`

	expected := `
AttributeExpr[hp]
	NameExpr[self]
`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)

	assertAST(t, expected, dotExprs[0])

}

func TestDotExprHasDots(t *testing.T) {
	src := `foo.bar.`
	expected := `
AttributeExpr[]
	AttributeExpr[bar]
		NameExpr[foo]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, dotExprs, 1)
	assertAST(t, expected, dotExprs[0])

	require.IsType(t, &pythonast.AttributeExpr{}, dotExprs[0])
	de := dotExprs[0].(*pythonast.AttributeExpr)
	require.NotNil(t, de.Dot)
	require.EqualValues(t, 7, de.Dot.Begin)
	require.EqualValues(t, 8, de.Dot.End)

	require.IsType(t, &pythonast.AttributeExpr{}, de.Value)
	de = de.Value.(*pythonast.AttributeExpr)
	require.NotNil(t, de.Dot)
	require.EqualValues(t, 3, de.Dot.Begin)
	require.EqualValues(t, 4, de.Dot.End)
}

func TestDotExprBegin(t *testing.T) {
	src := `   foo.bar`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	dotExprs := extractDotExprs(kitectx.Background(), words, []byte(src[3:]), 3)

	require.Len(t, dotExprs, 1)

	attr := dotExprs[0].(*pythonast.AttributeExpr)
	assert.Equal(t, token.Pos(3), attr.Value.Begin())
	assert.Equal(t, token.Pos(6), attr.Value.End())

	assert.Equal(t, token.Pos(7), attr.Attribute.Begin)
	assert.Equal(t, token.Pos(10), attr.Attribute.End)
}

// --

func TestFunctionOffset(t *testing.T) {
	src :=
		`


foo()`

	expected := `
CallExpr
	NameExpr[foo]
`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src[2:]), 2)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])

	assert.EqualValues(t, 3, calls[0].Begin())
	assert.EqualValues(t, 8, calls[0].End())
}

func TestNotFunction(t *testing.T) {
	src := `a`
	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)
	assert.Nil(t, calls)
}

func TestFunctionNoArgs(t *testing.T) {
	src := `a.b()`

	expected := `
CallExpr
	AttributeExpr[b]
		NameExpr[a]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	call := calls[0].(*pythonast.CallExpr)
	assertAST(t, expected, call)

	// make sure we match the lexer
	assert.Empty(t, call.LeftParen.Literal, "literal for left paren should be empty")
	assert.Empty(t, call.RightParen.Literal, "literal for right parent should be empty")
}

func TestFunctionNoRightParen(t *testing.T) {
	src := `a(`

	expected := `
CallExpr
	NameExpr[a]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

// TODO(juan/naman) should we treat this as two bad arguments or one? Note that TestFunctionNoRightParen is treated as 0
func TestFunctionNoRightParenExtraComma(t *testing.T) {
	src := `a(,`

	expected := `
CallExpr
	NameExpr[a]
	Argument
		BadExpr
	Argument
		BadExpr
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

func TestFunctionArgs(t *testing.T) {
	src := `a.b(foo,bar,baz)`

	expected := `
CallExpr
	AttributeExpr[b]
		NameExpr[a]
	Argument
		NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		NameExpr[baz]
	`
	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

func TestFunctionEllipsisArg(t *testing.T) {
	src := `foo(...,...,....)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		EllipsisExpr
	Argument
		EllipsisExpr
	Argument
		AttributeExpr[]
			EllipsisExpr
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

func TestFunctionVarArgs(t *testing.T) {
	src := `a.b(*args)`

	// TODO(naman) specialized call parser fails to parse vararg
	expected := `
CallExpr
	AttributeExpr[b]
		NameExpr[a]
	NameExpr[args]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

func TestFuntionKWArgs(t *testing.T) {
	src := `a.b(**kwargs)`

	// TODO(naman) specialized call parser fails to parse kwarg
	expected := `
CallExpr
	AttributeExpr[b]
		NameExpr[a]
	NameExpr[kwargs]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

func TestFunctionParseNewLine(t *testing.T) {
	src := `
foo(b

`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[b]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

func TestFunctionEndCommas(t *testing.T) {
	src := `foo(bar,,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		BadExpr
	Argument
		BadExpr
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
	assert.EqualValues(t, 9, calls[0].End())
}

func TestFunctionEmptyArgs(t *testing.T) {
	src := `foo(,a,,b,)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	Argument
		NameExpr[a]
	Argument
		BadExpr
	Argument
		NameExpr[b]
	`
	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])

}

func TestFunctionCommas(t *testing.T) {
	src := `foo(bar,,car,,zar)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		BadExpr
	Argument
		NameExpr[car]
	Argument
		BadExpr
	Argument
		NameExpr[zar]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)
	require.IsType(t, &pythonast.CallExpr{}, calls[0])
	call := calls[0].(*pythonast.CallExpr)

	assertAST(t, expected, call)

	require.Len(t, call.Commas, 4)

	assert.EqualValues(t, 7, call.Commas[0].Begin)
	assert.EqualValues(t, 8, call.Commas[0].End)

	assert.EqualValues(t, 8, call.Commas[1].Begin)
	assert.EqualValues(t, 9, call.Commas[1].End)

	assert.EqualValues(t, 12, call.Commas[2].Begin)
	assert.EqualValues(t, 13, call.Commas[2].End)

	assert.EqualValues(t, 13, call.Commas[3].Begin)
	assert.EqualValues(t, 14, call.Commas[3].End)
}

func TestFunctionNoClosingParen(t *testing.T) {
	src := `
x = foo(
z = bar()
	`

	expected := []string{
		`
CallExpr
	NameExpr[foo]
	`,
		`
CallExpr
	NameExpr[bar]
	`,
	}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, len(expected))

	for i := range expected {
		assertAST(t, expected[i], calls[i])
	}
}

func TestFunctionNoClosingParen2(t *testing.T) {
	src := `
x = foo(
z = bar())
	`

	expected := []string{
		`
CallExpr
	NameExpr[foo]
	`,
		`
CallExpr
	NameExpr[bar]
`,
	}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, len(expected))

	for i := range expected {
		assertAST(t, expected[i], calls[i])
	}
}

func TestFunctionNestedNoClosing(t *testing.T) {
	src := `
foo(bar()
foo(bar(
`
	expected := []string{
		`
CallExpr
	NameExpr[foo]
	Argument
		CallExpr
			NameExpr[bar]
`,
		`
CallExpr
	NameExpr[bar]
`,
		`
CallExpr
	NameExpr[foo]
	Argument
		CallExpr
			NameExpr[bar]
`,
		`
CallExpr
	NameExpr[bar]
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, len(expected))

	for i := range expected {
		assertAST(t, expected[i], calls[i])
	}
}

// --

func TestFunctionNested(t *testing.T) {
	src := `a.b(c.d(e.f()))`

	expected := []string{
		`
CallExpr
	AttributeExpr[b]
		NameExpr[a]
	Argument
		CallExpr
			AttributeExpr[d]
				NameExpr[c]
			Argument
				CallExpr
					AttributeExpr[f]
						NameExpr[e]
`,
		`
CallExpr
	AttributeExpr[d]
		NameExpr[c]
	Argument
		CallExpr
			AttributeExpr[f]
				NameExpr[e]
`,
		`
CallExpr
	AttributeExpr[f]
		NameExpr[e]
`,
	}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 3)

	for i := range expected {
		assertAST(t, expected[i], calls[i])
	}
}

func TestSkipStringLiteral(t *testing.T) {
	src := `
car
for foo in bar:
'''
p()
dont = parse(me)
'''	
`
	expected := `
Module
	ExprStmt
		NameExpr[car]
	BadStmt
		ForStmt
			NameExpr[foo]
			NameExpr[bar]
			BadStmt
	`

	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

// --

func TestReturnAssignmentOffset(t *testing.T) {
	src := `



x = foo()`

	expected := `
AssignStmt
	NameExpr[x]
	CallExpr
		NameExpr[foo]
	`
	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	returns := extractReturnAssignments(kitectx.Background(), words, []byte(src[4:]), 4)

	require.Len(t, returns, 1)

	assertAST(t, expected, returns[0])

	assert.EqualValues(t, 4, returns[0].Begin())
	assert.EqualValues(t, 13, returns[0].End())

}

func TestReturnAssignment(t *testing.T) {
	src := `x = car()`

	expected := `
AssignStmt
	NameExpr[x]
	CallExpr
		NameExpr[car]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	returns := extractReturnAssignments(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, returns, 1)

	assertAST(t, expected, returns[0])
}

func TestReturnAssignmentMissingRightParen(t *testing.T) {
	src := `x = foo(`

	expected := `
AssignStmt
	NameExpr[x]
	CallExpr
		NameExpr[foo]
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	returns := extractReturnAssignments(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, returns, 1)

	assertAST(t, expected, returns[0])
}

func TestReturnAssignmentMultiple(t *testing.T) {
	src := `
x = foo( 
y = bar(`
	expected := []string{
		`
AssignStmt
	NameExpr[x]
	CallExpr
		NameExpr[foo]
	`,
		`
AssignStmt
	NameExpr[y]
	CallExpr
		NameExpr[bar]
`,
	}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	returns := extractReturnAssignments(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, returns, 2)

	for i := range expected {
		assertAST(t, expected[i], returns[i])
	}
}

func TestReturnAssignmentArguments(t *testing.T) {
	src := `
x = foo(bar)
y = star(car)
`

	expected := []string{`
AssignStmt
	NameExpr[x]
	CallExpr
		NameExpr[foo]
		Argument
			NameExpr[bar]
	`,
		`
AssignStmt
	NameExpr[y]
	CallExpr
		NameExpr[star]
		Argument
			NameExpr[car]
`,
	}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	returns := extractReturnAssignments(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, returns, len(expected))

	for i := range expected {
		assertAST(t, expected[i], returns[i])
	}
}

func TestReturnAssignmentArguments2(t *testing.T) {
	src := `
x = requests.get(''
	`

	expected := `
AssignStmt
	NameExpr[x]
	CallExpr
		AttributeExpr[get]
			NameExpr[requests]
		Argument
			StringExpr['']
	`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	returns := extractReturnAssignments(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, returns, 1)

	assertAST(t, expected, returns[0])
}

func TestReturnAssignmentNoLHS(t *testing.T) {
	src := `if = requests.get("url")`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})

	returns := extractReturnAssignments(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, returns, 0)
}

// --

func TestDottedExpr(t *testing.T) {
	src := `foo.bar`
	expected := `
DottedExpr
	NameExpr[foo]
	NameExpr[bar]
	`

	dotted := parseDottedExpr([]byte(src), 0)

	assertAST(t, expected, dotted)

	require.Len(t, dotted.Names, 2)
	assert.EqualValues(t, 0, dotted.Names[0].Begin())
	assert.EqualValues(t, 3, dotted.Names[0].End())
	assert.EqualValues(t, 4, dotted.Names[1].Begin())
	assert.EqualValues(t, 7, dotted.Names[1].End())

	require.Len(t, dotted.Dots, 1)
	dot := dotted.Dots[0]
	assert.EqualValues(t, 3, dot.Begin)
	assert.EqualValues(t, 4, dot.End)

	// make sure we match the lexer
	assert.Empty(t, dot.Literal, "dot literal should be empty")
}

func TestDottedExpr2(t *testing.T) {
	src := `foo.bar.car`
	expected := `
DottedExpr
	NameExpr[foo]
	NameExpr[bar]
	NameExpr[car]
`
	dotted := parseDottedExpr([]byte(src), 0)

	assertAST(t, expected, dotted)

	require.Len(t, dotted.Names, 3)
	assert.EqualValues(t, 0, dotted.Names[0].Begin())
	assert.EqualValues(t, 3, dotted.Names[0].End())
	assert.EqualValues(t, 4, dotted.Names[1].Begin())
	assert.EqualValues(t, 7, dotted.Names[1].End())
	assert.EqualValues(t, 8, dotted.Names[2].Begin())
	assert.EqualValues(t, 11, dotted.Names[2].End())

	require.Len(t, dotted.Dots, 2)
	assert.EqualValues(t, 3, dotted.Dots[0].Begin)
	assert.EqualValues(t, 4, dotted.Dots[0].End)
	assert.EqualValues(t, 7, dotted.Dots[1].Begin)
	assert.EqualValues(t, 8, dotted.Dots[1].End)
}

func TestDottedExprOffset(t *testing.T) {
	src := `  foo.bar`
	expected := `
DottedExpr
	NameExpr[foo]
	NameExpr[bar]
	`

	dotted := parseDottedExpr([]byte(src[2:]), 2)

	assertAST(t, expected, dotted)
	require.Len(t, dotted.Names, 2)
	assert.EqualValues(t, 2, dotted.Names[0].Begin())
	assert.EqualValues(t, 5, dotted.Names[0].End())
	assert.EqualValues(t, 6, dotted.Names[1].Begin())
	assert.EqualValues(t, 9, dotted.Names[1].End())

	require.Len(t, dotted.Dots, 1)
	dot := dotted.Dots[0]
	assert.EqualValues(t, 5, dot.Begin)
	assert.EqualValues(t, 6, dot.End)
}

func TestDottedExprNoDot(t *testing.T) {
	src := `foo`
	expected := `
DottedExpr
	NameExpr[foo]
	`

	dotted := parseDottedExpr([]byte(src), 0)

	assertAST(t, expected, dotted)

	require.Len(t, dotted.Names, 1)
	assert.EqualValues(t, 0, dotted.Names[0].Begin())
	assert.EqualValues(t, 3, dotted.Names[0].End())

	assert.Len(t, dotted.Dots, 0)
}

func TestDottedExprSpace(t *testing.T) {
	src := `  foo`
	expected := `
DottedExpr
	NameExpr[  foo]
	`
	// note spaces
	dotted := parseDottedExpr([]byte(src), 0)

	assertAST(t, expected, dotted)

	assert.Len(t, dotted.Dots, 0)
	require.Len(t, dotted.Names, 1)

	assert.EqualValues(t, 0, dotted.Names[0].Begin())
	assert.EqualValues(t, 5, dotted.Names[0].End())
}

func TestDottedExprExtraDot(t *testing.T) {
	src := `foo.`
	expected := `
DottedExpr
	NameExpr[foo]
	NameExpr[]
	`

	dotted := parseDottedExpr([]byte(src), 0)

	assertAST(t, expected, dotted)

	require.Len(t, dotted.Names, 2)

	assert.EqualValues(t, 0, dotted.Names[0].Begin())
	assert.EqualValues(t, 3, dotted.Names[0].End())

	assert.EqualValues(t, 4, dotted.Names[1].Begin())
	assert.EqualValues(t, 4, dotted.Names[1].End())
}

// --

func TestImportAsName(t *testing.T) {
	src := `foo as   bar,   baz,`
	expected := []string{
		`
ImportAsName
	NameExpr[foo]
	NameExpr[bar]
`,
		`
ImportAsName
	NameExpr[baz]
`,
	}

	names, commas := extractImportAsName([]byte(src), 0)

	require.Len(t, names, len(expected))

	for i := range expected {
		assertAST(t, expected[i], names[i])
	}

	name := names[0]
	assert.EqualValues(t, 0, name.Begin())
	assert.EqualValues(t, 12, name.End())
	assert.EqualValues(t, 0, name.External.Begin())
	assert.EqualValues(t, 3, name.External.End())
	assert.EqualValues(t, 9, name.Internal.Begin())
	assert.EqualValues(t, 12, name.Internal.End())

	name = names[1]
	assert.EqualValues(t, 16, name.Begin())
	assert.EqualValues(t, 19, name.End())
	assert.EqualValues(t, 16, name.External.Begin())
	assert.EqualValues(t, 19, name.External.End())

	require.Len(t, commas, 2)
	assert.EqualValues(t, 12, commas[0].Begin)
	assert.EqualValues(t, 13, commas[0].End)

	assert.EqualValues(t, 19, commas[1].Begin)
	assert.EqualValues(t, 20, commas[1].End)
}

// --

func TestImportFrom(t *testing.T) {
	src := `     from foo import bar as car, ham,`
	expected := `
ImportFromStmt
	DottedExpr
		NameExpr[foo]
	ImportAsName
		NameExpr[bar]
		NameExpr[car]
	ImportAsName
		NameExpr[ham]
	`
	imports := extractImportFrom([]byte(src[4:]), 4)

	require.Len(t, imports, 1)

	imp := imports[0].(*pythonast.ImportFromStmt)

	assertAST(t, expected, imp)

	// check begin/end of sub expressions
	assert.EqualValues(t, 5, imp.From.Begin)
	assert.EqualValues(t, 9, imp.From.End)

	assert.EqualValues(t, 10, imp.Package.Begin())
	assert.EqualValues(t, 13, imp.Package.End())

	assert.Len(t, imp.Names, 2)

	assert.EqualValues(t, 21, imp.Names[0].Begin())
	assert.EqualValues(t, 31, imp.Names[0].End())
	assert.EqualValues(t, 21, imp.Names[0].External.Begin())
	assert.EqualValues(t, 24, imp.Names[0].External.End())
	assert.EqualValues(t, 28, imp.Names[0].Internal.Begin())
	assert.EqualValues(t, 31, imp.Names[0].Internal.End())

	assert.EqualValues(t, 33, imp.Names[1].Begin())
	assert.EqualValues(t, 36, imp.Names[1].End())
	assert.EqualValues(t, 33, imp.Names[1].External.Begin())
	assert.EqualValues(t, 36, imp.Names[1].External.End())

	// make sure import word is not nil and begin/end correct
	require.NotNil(t, imp.Import)
	assert.EqualValues(t, 14, imp.Import.Begin)
	assert.EqualValues(t, 20, imp.Import.End)

	// make sure from word is correct
	require.NotNil(t, imp.From)
	assert.EqualValues(t, 5, imp.From.Begin)
	assert.EqualValues(t, 9, imp.From.End)

	// make sure comma is correct
	require.Len(t, imp.Commas, 2)
	assert.EqualValues(t, 31, imp.Commas[0].Begin)
	assert.EqualValues(t, 32, imp.Commas[0].End)

	assert.EqualValues(t, 36, imp.Commas[1].Begin)
	assert.EqualValues(t, 37, imp.Commas[1].End)
}

func TestImportFromNoImport(t *testing.T) {
	src := `from foo.bar`
	expected := `
ImportFromStmt
	DottedExpr
		NameExpr[foo]
		NameExpr[bar]
	`

	imports := extractImportFrom([]byte(src), 0)

	require.Len(t, imports, 1)

	assertAST(t, expected, imports[0])

	require.IsType(t, &pythonast.ImportFromStmt{}, imports[0])

	imp := imports[0].(*pythonast.ImportFromStmt)

	// check begin/end of sub expressions
	assert.EqualValues(t, 0, imp.Begin())
	assert.EqualValues(t, len(src), imp.End())

	require.Len(t, imp.Package.Names, 2)

	assert.EqualValues(t, 5, imp.Package.Names[0].Begin())
	assert.EqualValues(t, 8, imp.Package.Names[0].End())

	assert.EqualValues(t, 9, imp.Package.Names[1].Begin())
	assert.EqualValues(t, 12, imp.Package.Names[1].End())

	// make sure import word is nil
	assert.Nil(t, imp.Import)
	// make sure from word is correct
	require.NotNil(t, imp.From)
	assert.EqualValues(t, 0, imp.From.Begin)
	assert.EqualValues(t, 4, imp.From.End)
}

func TestImportFromOnlyFrom(t *testing.T) {
	src := `from `
	expected := `
ImportFromStmt
	`
	imports := extractImportFrom([]byte(src), 0)

	require.Len(t, imports, 1)

	assertAST(t, expected, imports[0])
}

func TestImportFromWithDots(t *testing.T) {
	src := `
from ..
from . import
from ...foo.bar import baz
`
	imports := extractImportFrom([]byte(src), 0)
	require.Len(t, imports, 3)

	imp := imports[0].(*pythonast.ImportFromStmt)
	require.Equal(t, 2, len(imp.Dots))

	imp = imports[1].(*pythonast.ImportFromStmt)
	require.Equal(t, 1, len(imp.Dots))

	imp = imports[2].(*pythonast.ImportFromStmt)
	require.Equal(t, 3, len(imp.Dots))
	require.Equal(t, "foo", imp.Package.Names[0].Ident.Literal)
	require.Equal(t, "bar", imp.Package.Names[1].Ident.Literal)
	require.Equal(t, "baz", imp.Names[0].External.Ident.Literal)
}

func TestAttributeBaseNeverEmpty(t *testing.T) {
	src := `foo().bar(`

	expected := `
CallExpr
	AttributeExpr[bar]
		CallExpr
			NameExpr[foo]
	`

	words, err := pythonscanner.Lex([]byte(src), pythonscanner.DefaultOptions)
	require.NoError(t, err)
	calls := extractFunctionCalls(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, calls, 1)

	assertAST(t, expected, calls[0])
}

func TestAttribute(t *testing.T) {
	src := `foo.`

	expected := `
AttributeExpr[]
	NameExpr[foo]
`

	a := parseDotExpr(kitectx.Background(), []byte(src), 0)

	require.NotNil(t, a)

	attr := a.(*pythonast.AttributeExpr)

	assertAST(t, expected, attr)

	// make sure we match the lexer
	assert.Empty(t, attr.Dot.Literal, "dot for attribute expression should have an emtpy literal")
}

// --

func TestNameExpr(t *testing.T) {
	src := `foo  `
	expected := `
NameExpr[foo  ]
	`
	// NOTE: extra spaces in name above

	name := parseNameExpr([]byte(src), 0)
	assertAST(t, expected, name)
}

// --

func TestDottedAsName(t *testing.T) {
	src := ` foo  as   bar `
	expected :=
		`
DottedAsName
	DottedExpr
		NameExpr[foo]
	NameExpr[bar]
	`

	dn := parseDottedAsName([]byte(src), 0)

	assertAST(t, expected, dn)

	assert.EqualValues(t, 1, dn.Begin())
	assert.EqualValues(t, 14, dn.End())

	assert.EqualValues(t, 1, dn.External.Begin())
	assert.EqualValues(t, 4, dn.External.End())

	assert.EqualValues(t, 11, dn.Internal.Begin())
	assert.EqualValues(t, 14, dn.Internal.End())
}

func TestDottedAsNameNoInternal(t *testing.T) {
	src := `    foo.bar  `
	expected := `
DottedAsName
	DottedExpr
		NameExpr[foo]
		NameExpr[bar]
	`

	dn := parseDottedAsName([]byte(src[3:]), 3)

	assertAST(t, expected, dn)

	assert.EqualValues(t, 4, dn.Begin())
	assert.EqualValues(t, 11, dn.End())

	require.Len(t, dn.External.Names, 2)

	assert.EqualValues(t, 4, dn.External.Names[0].Begin())
	assert.EqualValues(t, 7, dn.External.Names[0].End())

	assert.EqualValues(t, 8, dn.External.Names[1].Begin())
	assert.EqualValues(t, 11, dn.External.Names[1].End())
}

// --
func TestImportName(t *testing.T) {
	src := `    import   foo  as  bar, car.bar, star as mar`
	expected := `
ImportNameStmt
	DottedAsName
		DottedExpr
			NameExpr[foo]
		NameExpr[bar]
	DottedAsName
		DottedExpr
			NameExpr[car]
			NameExpr[bar]
	DottedAsName
		DottedExpr
			NameExpr[star]
		NameExpr[mar]
`
	imps := extractImportName([]byte(src[2:]), 2)

	require.Len(t, imps, 1)

	assertAST(t, expected, imps[0])

	require.IsType(t, &pythonast.ImportNameStmt{}, imps[0])

	imp := imps[0].(*pythonast.ImportNameStmt)

	assert.EqualValues(t, 4, imp.Begin())
	assert.EqualValues(t, len(src), imp.End())

	require.Len(t, imp.Names, 3)

	assert.EqualValues(t, 13, imp.Names[0].Begin())
	assert.EqualValues(t, 25, imp.Names[0].End())

	assert.EqualValues(t, 27, imp.Names[1].Begin())
	assert.EqualValues(t, 34, imp.Names[1].End())

	assert.EqualValues(t, 36, imp.Names[2].Begin())
	assert.EqualValues(t, 47, imp.Names[2].End())

	// check commas
	require.Len(t, imp.Commas, 2)

	comma := imp.Commas[0]
	assert.EqualValues(t, 25, comma.Begin)
	assert.EqualValues(t, 26, comma.End)

	comma = imp.Commas[1]
	assert.EqualValues(t, 34, comma.Begin)
	assert.EqualValues(t, 35, comma.End)
}

func TestImportNameNoName(t *testing.T) {
	src := `import`

	expected := `
ImportNameStmt
	`

	imp := extractImportName([]byte(src), 0)

	require.Len(t, imp, 1)

	assertAST(t, expected, imp[0])
}

func TestImportNameNotInImportFrom(t *testing.T) {
	src := `from foo import bar`
	imps := extractImportName([]byte(src), 0)
	assert.Len(t, imps, 0)
}

// --

func TestCombineConsectiveBadStmts(t *testing.T) {
	src := `
def
class
def foo(): pass
def
class bar():
	def foo(self): pass
	def
	class
	def foo(self): pass
	class
class
	`
	expected := `
Module
	BadStmt
	FunctionDefStmt
		NameExpr[foo]
		PassStmt
	BadStmt
	ClassDefStmt
		NameExpr[bar]
		FunctionDefStmt
			NameExpr[foo]
			Parameter
				NameExpr[self]
			PassStmt
		BadStmt
		FunctionDefStmt
			NameExpr[foo]
			Parameter
				NameExpr[self]
			PassStmt
		BadStmt
	BadStmt
`
	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

// TODO(juan): should we not allow the def.def case?
func TestParseKeywordsInAttribute(t *testing.T) {
	src := `
collections.def.while
def
def.def
`
	expected := `
Module
	BadStmt
		ExprStmt
			AttributeExpr[while]
				AttributeExpr[def]
					NameExpr[collections]
		ExprStmt
			AttributeExpr[def]
				NameExpr[def]
	`

	mod, _ := ApproxParse([]byte(src), opts)
	assertAST(t, expected, mod)
}

func TestNoOverlappingNodes(t *testing.T) {
	src := `import foo.bar.`
	mod, _ := ApproxParse([]byte(src), Options{})

	require.Len(t, mod.Body, 1)

	require.IsType(t, &pythonast.BadStmt{}, mod.Body[0])

	stmt := mod.Body[0].(*pythonast.BadStmt)

	require.Len(t, stmt.Approximation, 1)

	require.IsType(t, &pythonast.ImportNameStmt{}, stmt.Approximation[0])
}

// --

func TestRemoveComments(t *testing.T) {
	src := `
import os.path.join

from 

# class foo():
`

	expected := `
import os.path.join

from 

              
`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{
		ScanComments: true,
	})

	actual := removeComments([]byte(src), words)

	assert.Equal(t, expected, string(actual))
}

func TestNoParseInComments(t *testing.T) {
	src := `
import os.path.join

from

# class foo():
# 	bar = 1

# 	def baz(self):
# 		"""baz doc string"""
# 		pass

# 	def maz(self):
# 		''' maz doc string  '''
# 		pass

# 	def caz(self):
# 		"""caz doc strings"""
# 		self.maz()
# 		pass

# def car():
# 	"""a doc string!"""
# 	pass
	`

	expected := `
Module
	ImportNameStmt
		DottedAsName
			DottedExpr
				NameExpr[os]
				NameExpr[path]
				NameExpr[join]
	BadStmt
		ImportFromStmt
	`

	mod, _ := ApproxParse([]byte(src), Options{})

	assertAST(t, expected, mod)
}

func TestNoCrashRemoveComments(t *testing.T) {
	src := `
# this is a comment
# this is another comment

import

# this is also a comment
# and so is this!
	`

	expected := `
Module
	BadStmt
		ImportNameStmt
`

	mod, _ := ApproxParse([]byte(src), Options{})
	assertAST(t, expected, mod)
}

// --

func TestWordsInRegion(t *testing.T) {
	src :=
		`# this is a comment
def foo():
	print bar`

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{
		ScanComments: true,
	})

	commentWords := wordsInRegion(words, 0, 19)
	require.Len(t, commentWords, 1)
	assert.EqualValues(t, 0, commentWords[0].Begin)
	assert.EqualValues(t, 19, commentWords[0].End)
	assert.Equal(t, pythonscanner.Comment, commentWords[0].Token)

	fnWords := wordsInRegion(words, 20, 41)
	assert.Len(t, fnWords, 9)
}

func TestFunctionNonNameKeywordArg(t *testing.T) {
	// Issue https://github.com/kiteco/kiteco/issues/7528
	src := `
x.fn(self.v=1)
`
	expected := `
Module
	BadStmt
		ExprStmt
			CallExpr
				AttributeExpr[fn]
					NameExpr[x]
				Argument
					BadExpr
`

	mod, _ := ApproxParse([]byte(src), Options{})
	assertAST(t, expected, mod)
}

func TestParseClassDefs(t *testing.T) {
	src := `
class A::
`
	expected := `
Module
	ClassDefStmt
		NameExpr[A]
		BadStmt
`
	mod, _ := ApproxParse([]byte(src), Options{})
	assertAST(t, expected, mod)
}

func TestClassDefinitions(t *testing.T) {
	src := `
"""
class Nope
"""
  class A
`

	expected := []string{`
ClassDefStmt
	NameExpr[A]
	BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	classes := extractNamedDefinitions(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, classes, len(expected))

	for i := range classes {
		assertAST(t, expected[i], classes[i])
	}
}

func TestMultiClassDefinitions(t *testing.T) {
	src := `
class A
class B
`

	expected := []string{`
ClassDefStmt
	NameExpr[A]
	BadStmt
`,
		`
ClassDefStmt
	NameExpr[B]
	BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	classes := extractNamedDefinitions(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, classes, len(expected))

	for i := range classes {
		assertAST(t, expected[i], classes[i])
	}
}

func TestFunctionDefinitions(t *testing.T) {
	src := `
"""
def nope:
"""
  def a
`

	expected := []string{`
FunctionDefStmt
	NameExpr[a]
	BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	funcs := extractNamedDefinitions(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, funcs, len(expected))

	for i := range funcs {
		assertAST(t, expected[i], funcs[i])
	}
}

func TestMultiFunctionDefinitions(t *testing.T) {
	src := `
def a
def b (x, y=1, ,
`

	expected := []string{`
FunctionDefStmt
	NameExpr[a]
	BadStmt
`,
		`
FunctionDefStmt
	NameExpr[b]
	Parameter
		NameExpr[x]
	Parameter
		NameExpr[y]
		NumberExpr[1]
	Parameter
		BadExpr
	Parameter
		BadExpr
	BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	funcs := extractNamedDefinitions(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, funcs, len(expected))

	for i := range funcs {
		assertAST(t, expected[i], funcs[i])
	}
}

func TestIfStatements(t *testing.T) {
	src := `
"""
if x:
"""
if y:
if obj.fn(z)
`

	expected := []string{`
IfStmt
	Branch
		NameExpr[y]
		BadStmt
`,
		`IfStmt
	Branch
		CallExpr
			AttributeExpr[fn]
				NameExpr[obj]
			Argument
				NameExpr[z]
		BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	ifs := extractKeywordStatements(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, ifs, len(expected))

	for i := range ifs {
		assertAST(t, expected[i], ifs[i])
	}
}

func TestWithStatements(t *testing.T) {
	src := `
"""
with x:
"""
with y:
with obj.fn(x) as y
`

	expected := []string{`
WithStmt
	WithItem
		NameExpr[y]
	BadStmt
`,
		`WithStmt
	WithItem
		CallExpr
			AttributeExpr[fn]
				NameExpr[obj]
			Argument
				NameExpr[x]
		NameExpr[y]
	BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	withs := extractKeywordStatements(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, withs, len(expected))

	for i := range withs {
		assertAST(t, expected[i], withs[i])
	}
}

func TestWhileStatements(t *testing.T) {
	src := `
"""
while x:
"""
while y:
while obj.fn(x)
`

	expected := []string{`
WhileStmt
	NameExpr[y]
	BadStmt
`,
		`WhileStmt
	CallExpr
		AttributeExpr[fn]
			NameExpr[obj]
		Argument
			NameExpr[x]
	BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	whiles := extractKeywordStatements(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, whiles, len(expected))

	for i := range whiles {
		assertAST(t, expected[i], whiles[i])
	}
}

func TestForStatements(t *testing.T) {
	src := `
"""
for x in y:
"""
for z:
for a in obj.fn(x)
`

	expected := []string{`
ForStmt
	NameExpr[z]
	BadExpr
	BadStmt
`,
		`ForStmt
	NameExpr[a]
	CallExpr
		AttributeExpr[fn]
			NameExpr[obj]
		Argument
			NameExpr[x]
	BadStmt
`}

	words, _ := pythonscanner.Lex([]byte(src), pythonscanner.Options{})
	whiles := extractKeywordStatements(kitectx.Background(), words, []byte(src), 0)

	require.Len(t, whiles, len(expected))

	for i := range whiles {
		assertAST(t, expected[i], whiles[i])
	}
}
