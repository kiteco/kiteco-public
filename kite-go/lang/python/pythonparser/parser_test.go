package pythonparser

import (
	"bytes"
	"fmt"
	"go/token"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var unaryOperators = []pythonscanner.Token{
	pythonscanner.BitNot,
	pythonscanner.Not,
	pythonscanner.Add,
	pythonscanner.Sub,
}

var binaryOperators = []pythonscanner.Token{
	pythonscanner.Add,
	pythonscanner.Sub,
	pythonscanner.Mul,
	pythonscanner.Pow,
	pythonscanner.Div,
	pythonscanner.Truediv,
	pythonscanner.Pct,

	pythonscanner.BitAnd,
	pythonscanner.BitOr,
	pythonscanner.BitXor,
	pythonscanner.BitLshift,
	pythonscanner.BitRshift,

	pythonscanner.Le,
	pythonscanner.Ge,
	pythonscanner.Lt,
	pythonscanner.Gt,

	pythonscanner.And,
	pythonscanner.Or,
}

func assertParse(t *testing.T, expected string, src string) {
	t.Log(src)
	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)
}

func assertParseExpr(t *testing.T, expected string, src string) {
	assertParseExprWithPos(t, expected, src, 0, len(src))
}

func assertParseExprWithPos(t *testing.T, expected string, src string, start, end int) {
	t.Log(src)
	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.ExprStmt{}, mod.Body[0])

	expr := mod.Body[0].(*pythonast.ExprStmt).Value
	require.NotNil(t, expr)

	assert.EqualValues(t, start, expr.Begin())
	assert.EqualValues(t, end, expr.End())

	assertAST(t, expected, expr)
}

type nestingResult struct {
	violations []string
}

type nestingVerifier struct {
	result      *nestingResult
	parentBegin token.Pos
	parentEnd   token.Pos
}

func (p *nestingVerifier) Visit(n pythonast.Node) pythonast.Visitor {
	if n == nil {
		return nil
	}
	if n.Begin() < p.parentBegin || n.End() > p.parentEnd {
		msg := fmt.Sprintf("error at %T (%d...%d)", n, n.Begin(), n.End())
		p.result.violations = append(p.result.violations, msg)
	}
	return &nestingVerifier{p.result, n.Begin(), n.End()}
}

// assertNesting checks that the Begin and End of each node fully encloses the Begin and End of each of its children
func assertNesting(t *testing.T, node pythonast.Node) {
	var result nestingResult
	verifier := nestingVerifier{&result, node.Begin(), node.End()}
	pythonast.Walk(&verifier, node)
	if len(result.violations) > 0 {
		msg := strings.Join(result.violations, "\n")
		var buf bytes.Buffer
		pythonast.PrintPositions(node, &buf, "\t")
		t.Errorf("Nesting violations:\n%s\n%s", msg, buf.String())
	}
}

// assertAllUsagesDecided checks that there are no nodes with Usage=0 (which indicates an un-filled usage)
func assertAllUsagesDecided(t *testing.T, node pythonast.Node) {
	pythonast.Inspect(node, func(n pythonast.Node) bool {
		switch n := n.(type) {
		case *pythonast.NameExpr:
			assert.NotEqual(t, undecided, n.Usage, "%s has undecided usage", pythonast.String(n))
		case *pythonast.AttributeExpr:
			assert.NotEqual(t, undecided, n.Usage, "%s has undecided usage", pythonast.String(n))
		case *pythonast.TupleExpr:
			assert.NotEqual(t, undecided, n.Usage, "%s has undecided usage", pythonast.String(n))
		case *pythonast.ListExpr:
			assert.NotEqual(t, undecided, n.Usage, "%s has undecided usage", pythonast.String(n))
		}
		return true
	})
}

type assertWord struct {
	t *testing.T
}

func (a assertWord) VisitNode(r pythonast.NodeRef) {
	pythonast.Iterate(a, r.Lookup())
}

func (a assertWord) VisitSlice(s pythonast.NodeSliceRef) {
	pythonast.VisitNodeSlice(a, s)
}

func (a assertWord) VisitWord(wp **pythonscanner.Word) {
	w := *wp
	if w == nil {
		return
	}

	if !w.Valid() {
		a.t.Errorf("invalid word: begin: %d end: %d tok: %s lit: '%s'", w.Begin, w.End, w.Token.String(), w.Literal)
	}
}

func assertWords(t *testing.T, node pythonast.Node) {
	if pythonast.IsNil(node) {
		return
	}
	node.Iterate(assertWord{t: t})
}

func assertAST(t *testing.T, expected string, node pythonast.Node) {
	var buf bytes.Buffer
	pythonast.Print(node, &buf, "\t")
	actual := buf.String()

	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	if actual != expected {
		expectedLines := strings.Split(expected, "\n")
		actualLines := strings.Split(actual, "\n")

		n := len(expectedLines)
		if len(actualLines) > n {
			n = len(actualLines)
		}

		errorLine := -1
		sidebyside := fmt.Sprintf("      | %-40s | %-40s |\n", "EXPECTED", "ACTUAL")
		var errorExpected, errorActual string
		for i := 0; i < n; i++ {
			var expectedLine, actualLine string
			if i < len(expectedLines) {
				expectedLine = strings.Replace(expectedLines[i], "\t", "    ", -1)
			}
			if i < len(actualLines) {
				actualLine = strings.Replace(actualLines[i], "\t", "    ", -1)
			}
			symbol := "   "
			if actualLine != expectedLine {
				symbol = "***"
				if errorLine == -1 {
					errorLine = i
					errorExpected = strings.TrimSpace(expectedLine)
					errorActual = strings.TrimSpace(actualLine)
				}
			}
			sidebyside += fmt.Sprintf("%-6s| %-40s | %-40s |\n", symbol, expectedLine, actualLine)
		}

		t.Errorf("expected %s but got %s (line %d):\n%s", errorExpected, errorActual, errorLine, sidebyside)
	}

	t.Log("\n" + actual)

	assertNesting(t, node)

	if _, ismodule := node.(*pythonast.Module); ismodule {
		assertAllUsagesDecided(t, node)
	}

	assertWords(t, node)
}

func TestImportFromStmt(t *testing.T) {
	src := `from foo.bar import baz, ham as spam`

	expected := `
Module
	ImportFromStmt
		DottedExpr
			NameExpr[foo]
			NameExpr[bar]
		ImportAsName
			NameExpr[baz]
		ImportAsName
			NameExpr[ham]
			NameExpr[spam]
`

	t.Log(src)
	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)

	assertAST(t, expected, mod)

	// make sure we include the import token
	require.Len(t, mod.Body, 1)

	require.IsType(t, &pythonast.ImportFromStmt{}, mod.Body[0])

	imp := mod.Body[0].(*pythonast.ImportFromStmt)
	require.NotNil(t, imp.Import)

	assert.EqualValues(t, 13, imp.Import.Begin)
	assert.EqualValues(t, 19, imp.Import.End)

	// make sure we included comma
	require.Len(t, imp.Commas, 1)
	assert.EqualValues(t, 23, imp.Commas[0].Begin)
	assert.EqualValues(t, 24, imp.Commas[0].End)
}

func TestImportFromStmtExtraComma(t *testing.T) {
	src := `from json.decoder import errmsg,`

	expected := `
Module
	BadStmt
	`

	t.Log(src)
	mod, err := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})
	assert.NotNil(t, err)

	assertAST(t, expected, mod)
}

func TestImportNameStmt(t *testing.T) {
	src := `import foo.bar, ham as spam`

	expected := `
Module
	ImportNameStmt
		DottedAsName
			DottedExpr
				NameExpr[foo]
				NameExpr[bar]
		DottedAsName
			DottedExpr
				NameExpr[ham]
			NameExpr[spam]
`

	t.Log(src)
	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)

	assertAST(t, expected, mod)

	imp := mod.Body[0].(*pythonast.ImportNameStmt)

	// make sure comma is correct
	require.Len(t, imp.Commas, 1)
	assert.EqualValues(t, 14, imp.Commas[0].Begin)
	assert.EqualValues(t, 15, imp.Commas[0].End)
}

func TestImportNameStmtExtraComma(t *testing.T) {
	src := `import os.path,`
	expected := `
Module
	BadStmt
	`

	t.Log(src)
	mod, err := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})
	assert.NotNil(t, err)

	assertAST(t, expected, mod)
}

func TestRelativeImportStmt(t *testing.T) {
	src := `from . import *`

	expected := `
Module
	ImportFromStmt
`

	assertParse(t, expected, src)
}

func TestImportUnitaryTuple(t *testing.T) {
	src := `from foo import (bar,)`
	expected := `
Module
	ImportFromStmt
		DottedExpr
			NameExpr[foo]
		ImportAsName
			NameExpr[bar]
`
	assertParse(t, expected, src)
}

func TestParenthesizedImport(t *testing.T) {
	src := `from foo import (ham as spam)`
	expected := `
Module
	ImportFromStmt
		DottedExpr
			NameExpr[foo]
		ImportAsName
			NameExpr[ham]
			NameExpr[spam]
	`

	assertParse(t, expected, src)
}

func TestIfStmt(t *testing.T) {
	src := `if foo or bar: x = 1`

	expected := `
Module
	IfStmt
		Branch
			BinaryExpr[or]
				NameExpr[foo]
				NameExpr[bar]
			AssignStmt
				NameExpr[x]
				NumberExpr[1]
`

	assertParse(t, expected, src)
}

func TestForStmt(t *testing.T) {
	src := `for i, x in enumerate(ham): print x`

	expected := `
Module
	ForStmt
		NameExpr[i]
		NameExpr[x]
		CallExpr
			NameExpr[enumerate]
			Argument
				NameExpr[ham]
		PrintStmt
			NameExpr[x]
`

	assertParse(t, expected, src)
}

func TestWhileStmt(t *testing.T) {
	src := `while not eof: count += 1`

	expected := `
Module
	WhileStmt
		UnaryExpr[not]
			NameExpr[eof]
		AugAssignStmt[+=]
			NameExpr[count]
			NumberExpr[1]
`

	assertParse(t, expected, src)
}

func TestTryStmt(t *testing.T) {
	src := `
try:
	print abc
except IOError as ex:
	pass
except TypeError, e:
	pass
except:
	pass
else:
	break
finally:
	continue
`

	expected := `
Module
	TryStmt
		PrintStmt
			NameExpr[abc]
		ExceptClause
			NameExpr[IOError]
			NameExpr[ex]
			PassStmt
		ExceptClause
			NameExpr[TypeError]
			NameExpr[e]
			PassStmt
		ExceptClause
			PassStmt
		BreakStmt
		ContinueStmt
`

	assertParse(t, expected, src)
}

func TestTryStmt2(t *testing.T) {
	src := `
try:
	print foo
	`

	expected := `
Module
	TryStmt
		PrintStmt
			NameExpr[foo]
	`
	assertParse(t, expected, src)
}

func TestTryStmt3(t *testing.T) {
	src := `
try:
	print foo
except IOError as ex:
	pass
	`

	expected := `
Module
	TryStmt
		PrintStmt
			NameExpr[foo]
		ExceptClause
			NameExpr[IOError]
			NameExpr[ex]
			PassStmt
	`

	assertParse(t, expected, src)
}

func TestTryStmt4(t *testing.T) {
	src := `
try:
	print foo
except IOError as ex:
	pass
else:
	break
	`

	expected := `
Module
	TryStmt
		PrintStmt
			NameExpr[foo]
		ExceptClause
			NameExpr[IOError]
			NameExpr[ex]
			PassStmt
		BreakStmt
	`

	assertParse(t, expected, src)
}

func TestRaiseStmt(t *testing.T) {
	src := `raise 123`
	expected := `
Module
	RaiseStmt
		NumberExpr[123]
`

	assertParse(t, expected, src)
}

func TestYieldStmt(t *testing.T) {
	src := `yield 123`
	expected := `
Module
	YieldStmt
		NumberExpr[123]
`

	assertParse(t, expected, src)
}

func TestReturnStmt(t *testing.T) {
	src := `return 123`
	expected := `
Module
	ReturnStmt
		NumberExpr[123]
`

	assertParse(t, expected, src)
}

func TestGlobalStmt(t *testing.T) {
	src := `global x`
	expected := `
Module
	GlobalStmt
		NameExpr[x]
`

	assertParse(t, expected, src)
}

func TestNonLocalStmt(t *testing.T) {
	src := `nonlocal x, y`
	expected := `
Module
	NonLocalStmt
		NameExpr[x]
		NameExpr[y]
`

	assertParse(t, expected, src)
}

func TestAssertStmt(t *testing.T) {
	src := `assert x`
	expected := `
Module
	AssertStmt
		NameExpr[x]
`

	assertParse(t, expected, src)
}

func TestDelStmt(t *testing.T) {
	src := `del x`
	expected := `
Module
	DelStmt
		NameExpr[x]
`

	assertParse(t, expected, src)
}

func TestExecStmt(t *testing.T) {
	src := `exec x`
	expected := `
Module
	ExecStmt
		NameExpr[x]
`

	assertParse(t, expected, src)
}

func TestExecAsExpr1(t *testing.T) {
	src := `a = exec(b)`
	expected := `
Module
	AssignStmt
		NameExpr[a]
		CallExpr
			NameExpr[exec]
			Argument
				NameExpr[b]
`
	assertParse(t, expected, src)
}

func TestExecAsExpr2(t *testing.T) {
	src := `def foo(exec): pass`
	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[exec]
		PassStmt
`
	assertParse(t, expected, src)
}

func TestExecAsExpr3(t *testing.T) {
	src := `class exec(object): pass`
	expected := `
Module
	ClassDefStmt
		NameExpr[exec]
		Argument
			NameExpr[object]
		PassStmt
`
	assertParse(t, expected, src)
}

func TestPassStmt(t *testing.T) {
	src := `pass`
	expected := `
Module
	PassStmt
`

	assertParse(t, expected, src)
}

func TestBreakStmt(t *testing.T) {
	src := `break`
	expected := `
Module
	BreakStmt
`

	assertParse(t, expected, src)
}

func TestContinueStmt(t *testing.T) {
	src := `continue`
	expected := `
Module
	ContinueStmt
`

	assertParse(t, expected, src)
}

func TestFunctionDef(t *testing.T) {
	src := "def foo(a, b=1, *args, **kwargs): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[a]
		Parameter
			NameExpr[b]
			NumberExpr[1]
		ArgsParameter
			NameExpr[args]
		ArgsParameter
			NameExpr[kwargs]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestFunctionDefNoParams(t *testing.T) {
	src := "def foo(): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestFunctionDefTrailingComma(t *testing.T) {
	src := "def foo(a,): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[a]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestReturnAnnotation(t *testing.T) {
	src := "def foo() -> int: pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		NameExpr[int]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestReturnAnnotationWithParams(t *testing.T) {
	src := "def foo(*a, b, c) -> 'abc': pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[b]
		Parameter
			NameExpr[c]
		ArgsParameter
			NameExpr[a]
		StringExpr['abc']
		PassStmt
`

	assertParse(t, expected, src)
}

func TestArgsParameterAnnotation(t *testing.T) {
	src := "def foo(x, *args: Iterable[basestring], **kwargs: Any): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[x]
		ArgsParameter
			NameExpr[args]
			IndexExpr
				NameExpr[Iterable]
				IndexSubscript
					NameExpr[basestring]
		ArgsParameter
			NameExpr[kwargs]
			NameExpr[Any]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestParameterAnnotation(t *testing.T) {
	src := "def foo(a, b:list, c:foo(x, y)=123): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[a]
		Parameter
			NameExpr[b]
			NameExpr[list]
		Parameter
			NameExpr[c]
			CallExpr
				NameExpr[foo]
				Argument
					NameExpr[x]
				Argument
					NameExpr[y]
			NumberExpr[123]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestKeywordOnlyParams(t *testing.T) {
	src := "def foo(a, *args, b, **kwargs): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[a]
		Parameter
			NameExpr[b]
		ArgsParameter
			NameExpr[args]
		ArgsParameter
			NameExpr[kwargs]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestAnonymousVararg(t *testing.T) {
	src := "def foo(a, *, b): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[a]
		Parameter
			NameExpr[b]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestAllKeywordOnlyParams(t *testing.T) {
	src := "def foo(*args, b, c): pass"

	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[b]
		Parameter
			NameExpr[c]
		ArgsParameter
			NameExpr[args]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestDecoratorWithParams(t *testing.T) {
	src := `
@foo.bar(1)
def foo(): pass`

	expected := `
Module
	FunctionDefStmt
		CallExpr
			AttributeExpr[bar]
				NameExpr[foo]
			Argument
				NumberExpr[1]
		NameExpr[foo]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestDecorator(t *testing.T) {
	src := `@mydecorator
def foo():
	pass`

	expected := `
Module
	FunctionDefStmt
		NameExpr[mydecorator]
		NameExpr[foo]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestClassDef(t *testing.T) {
	src := `
class Foo(object):
	def __init__(self):
		pass
`

	expected := `
Module
	ClassDefStmt
		NameExpr[Foo]
		Argument
			NameExpr[object]
		FunctionDefStmt
			NameExpr[__init__]
			Parameter
				NameExpr[self]
			PassStmt
`

	assertParse(t, expected, src)
}

func TestClassDef_EmptyBases(t *testing.T) {
	src := `class Foo(): pass`

	expected := `
Module
	ClassDefStmt
		NameExpr[Foo]
		PassStmt
`

	assertParse(t, expected, src)
}

func TestClassDef_Decorated(t *testing.T) {
	src := `
@foo
class car():
	pass
	`
	expected := `
Module
	ClassDefStmt
		NameExpr[foo]
		NameExpr[car]
		PassStmt
	`

	assertParse(t, expected, src)
}

func TestClassDef_Varargs(t *testing.T) {
	src := `
class C(*x, **y):
	pass
	`
	expected := `
Module
	ClassDefStmt
		NameExpr[C]
		NameExpr[x]
		NameExpr[y]
		PassStmt
	`

	assertParse(t, expected, src)
}

func TestClassDef_KeywordArgs(t *testing.T) {
	src := `
class C(A, metaclass=B):
	pass
	`
	expected := `
Module
	ClassDefStmt
		NameExpr[C]
		Argument
			NameExpr[A]
		Argument
			NameExpr[metaclass]
			NameExpr[B]
		PassStmt
	`

	assertParse(t, expected, src)
}

func TestPrintStmt1(t *testing.T) {
	src := `print 123`
	expected := `
Module
	PrintStmt
		NumberExpr[123]
`
	assertParse(t, expected, src)
}

func TestPrintStmt2(t *testing.T) {
	src := `print >> foo, x,`
	expected := `
Module
	PrintStmt
		NameExpr[foo]
		NameExpr[x]
`
	assertParse(t, expected, src)
}

func TestPrintStmt3(t *testing.T) {
	src := `print >> foo, (x, y, z)`
	expected := `
Module
	PrintStmt
		NameExpr[foo]
		TupleExpr
			NameExpr[x]
			NameExpr[y]
			NameExpr[z]
`
	assertParse(t, expected, src)
}

func TestPrintStmt4(t *testing.T) {
	src := `print(123, a, c=d, **kwargs)`
	expected := `
Module
	ExprStmt
		CallExpr
			NameExpr[print]
			Argument
				NumberExpr[123]
			Argument
				NameExpr[a]
			Argument
				NameExpr[c]
				NameExpr[d]
			NameExpr[kwargs]
`
	assertParse(t, expected, src)
}

func TestPrintStmt5(t *testing.T) {
	// yup this is some serious spartan madness...
	src := `print(),print()`
	expected := `
Module
	ExprStmt
		CallExpr
			NameExpr[print]
`
	assertParse(t, expected, src)
}

func TestPrintAsExpr1(t *testing.T) {
	src := `a = print(1)`
	expected := `
Module
	AssignStmt
		NameExpr[a]
		CallExpr
			NameExpr[print]
			Argument
				NumberExpr[1]
`
	assertParse(t, expected, src)
}

func TestPrintAsExpr2(t *testing.T) {
	src := `def foo(print): pass`
	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		Parameter
			NameExpr[print]
		PassStmt
`
	assertParse(t, expected, src)
}

func TestPrintAsExpr3(t *testing.T) {
	src := `class print(object): pass`
	expected := `
Module
	ClassDefStmt
		NameExpr[print]
		Argument
			NameExpr[object]
		PassStmt
`
	assertParse(t, expected, src)
}

func TestIntExpr(t *testing.T) {
	src := "123"
	assertParseExpr(t, "NumberExpr[123]", src)
}

func TestLongExpr(t *testing.T) {
	src := "123L"
	assertParseExpr(t, "NumberExpr[123L]", src)
}

func TestFloatExpr(t *testing.T) {
	src := "123.45e6"
	assertParseExpr(t, "NumberExpr[123.45e6]", src)
}

func TestStringExpr(t *testing.T) {
	src := `"abc"      '''def'''`
	expected := `StringExpr["abc" '''def''']`
	assertParseExpr(t, expected, src)
}

func TestListExpr(t *testing.T) {
	src := `[1, foo, "bar"]`
	expected := `
ListExpr
	NumberExpr[1]
	NameExpr[foo]
	StringExpr["bar"]
`
	assertParseExpr(t, expected, src)
}

func TestDictExpr(t *testing.T) {
	src := `{1:2, foo:"bar"}`
	expected := `
DictExpr
	KeyValuePair
		NumberExpr[1]
		NumberExpr[2]
	KeyValuePair
		NameExpr[foo]
		StringExpr["bar"]
`
	assertParseExpr(t, expected, src)
}

func TestSetExpr(t *testing.T) {
	src := `{1, foo, "bar"}`
	expected := `
SetExpr
	NumberExpr[1]
	NameExpr[foo]
	StringExpr["bar"]
`
	assertParseExpr(t, expected, src)
}

func TestEmptyList(t *testing.T) {
	src := `[]`
	expected := `ListExpr`
	assertParseExpr(t, expected, src)
}

func TestEmptyDict(t *testing.T) {
	src := `{}`
	expected := `DictExpr`
	assertParseExpr(t, expected, src)
}

func TestEmptyTuple(t *testing.T) {
	src := `()`
	expected := `TupleExpr`
	assertParseExpr(t, expected, src)
}

func TestParenNoTuple(t *testing.T) {
	src := `(a)`
	expected := `NameExpr[a]`
	assertParseExprWithPos(t, expected, src, 1, 2)
}

func TestSingularTuple(t *testing.T) {
	src := `(a,)`
	expected := `
TupleExpr
	NameExpr[a]`
	assertParseExpr(t, expected, src)
}

func TestSingularTupleNoParens(t *testing.T) {
	src := `a,`
	expected := `
TupleExpr
	NameExpr[a]`
	assertParseExpr(t, expected, src)
}

func TestUnaryOperators(t *testing.T) {
	srcTpl := "%s foobar"
	expectedTpl := `
UnaryExpr[%s]
	NameExpr[foobar]
`
	for _, op := range unaryOperators {
		src := fmt.Sprintf(srcTpl, op.String())
		expected := fmt.Sprintf(expectedTpl, op.String())
		assertParseExpr(t, expected, src)
	}
}

func TestBinaryOperators(t *testing.T) {
	srcTpl := "ham %s spam"
	expectedTpl := `
BinaryExpr[%s]
	NameExpr[ham]
	NameExpr[spam]
`
	for _, op := range binaryOperators {
		src := fmt.Sprintf(srcTpl, op.String())
		expected := fmt.Sprintf(expectedTpl, op.String())
		assertParseExpr(t, expected, src)
	}
}

func TestListComprehension(t *testing.T) {
	src := `[x for x in list if condition for z in ham if spam]`
	expected := `
ListComprehensionExpr
	NameExpr[x]
	Generator
		NameExpr[x]
		NameExpr[list]
		NameExpr[condition]
	Generator
		NameExpr[z]
		NameExpr[ham]
		NameExpr[spam]
`

	assertParseExpr(t, expected, src)
}

func TestDictComprehension(t *testing.T) {
	src := `{x:y for x in list if condition for z in ham if spam}`
	expected := `
DictComprehensionExpr
	NameExpr[x]
	NameExpr[y]
	Generator
		NameExpr[x]
		NameExpr[list]
		NameExpr[condition]
	Generator
		NameExpr[z]
		NameExpr[ham]
		NameExpr[spam]
`

	assertParseExpr(t, expected, src)
}

func TestArgComprehension(t *testing.T) {
	src := `foo(x for x in list if condition for z in ham if spam)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		ComprehensionExpr
			NameExpr[x]
			Generator
				NameExpr[x]
				NameExpr[list]
				NameExpr[condition]
			Generator
				NameExpr[z]
				NameExpr[ham]
				NameExpr[spam]
`

	assertParseExpr(t, expected, src)
}

func TestCommaArgs(t *testing.T) {
	src := `foo(x,)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[x]
`

	assertParseExpr(t, expected, src)
}

func TestCommasArgsNoCommas(t *testing.T) {
	src := `foo(x)`

	expected := `
Module
	ExprStmt
		CallExpr
			NameExpr[foo]
			Argument
				NameExpr[x]
`

	mod, err := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})

	assert.Nil(t, err)

	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)

	exprStmt, ok := mod.Body[0].(*pythonast.ExprStmt)
	require.True(t, ok)
	require.NotNil(t, exprStmt.Value)

	call, ok := exprStmt.Value.(*pythonast.CallExpr)

	require.True(t, ok)
	require.Len(t, call.Commas, 0)
}

func TestCommasArgsExtraComma(t *testing.T) {
	src := `foo(x,)`

	expected := `
Module
	ExprStmt
		CallExpr
			NameExpr[foo]
			Argument
				NameExpr[x]
	`

	mod, err := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})

	assert.Nil(t, err)

	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)

	exprStmt, ok := mod.Body[0].(*pythonast.ExprStmt)
	require.True(t, ok)
	require.NotNil(t, exprStmt.Value)

	call, ok := exprStmt.Value.(*pythonast.CallExpr)

	require.True(t, ok)
	require.Len(t, call.Commas, 1)

	assert.EqualValues(t, 5, call.Commas[0].Begin)
	assert.EqualValues(t, 6, call.Commas[0].End)
	assert.Equal(t, pythonscanner.Comma, call.Commas[0].Token)
}

func TestCommasArgsMultiArgs(t *testing.T) {
	src := `foo(x,y)`

	expected := `
Module
	ExprStmt
		CallExpr
			NameExpr[foo]
			Argument
				NameExpr[x]
			Argument
				NameExpr[y]
	`
	mod, err := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})

	assert.Nil(t, err)

	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)

	exprStmt, ok := mod.Body[0].(*pythonast.ExprStmt)
	require.True(t, ok)
	require.NotNil(t, exprStmt.Value)

	call, ok := exprStmt.Value.(*pythonast.CallExpr)

	require.True(t, ok)
	require.Len(t, call.Commas, 1)

	assert.EqualValues(t, 5, call.Commas[0].Begin)
	assert.EqualValues(t, 6, call.Commas[0].End)
	assert.Equal(t, pythonscanner.Comma, call.Commas[0].Token)
}

func TestIfExpr(t *testing.T) {
	src := `a if b else c`
	expected := `
IfExpr
	NameExpr[a]
	NameExpr[b]
	NameExpr[c]
`

	assertParseExpr(t, expected, src)
}

func TestIndex(t *testing.T) {
	src := `foo[1, :, 2:, :3, ::4, 5:6:7, ::, ...]`
	expected := `
IndexExpr
	NameExpr[foo]
	IndexSubscript
		NumberExpr[1]
	SliceSubscript
	SliceSubscript
		NumberExpr[2]
	SliceSubscript
		NumberExpr[3]
	SliceSubscript
		NumberExpr[4]
	SliceSubscript
		NumberExpr[5]
		NumberExpr[6]
		NumberExpr[7]
	SliceSubscript
	EllipsisExpr
`

	assertParseExpr(t, expected, src)
}

func TestLambda(t *testing.T) {
	src := `lambda x, y=1, *z: x + y`
	expected := `
LambdaExpr
	Parameter
		NameExpr[x]
	Parameter
		NameExpr[y]
		NumberExpr[1]
	ArgsParameter
		NameExpr[z]
	BinaryExpr[+]
		NameExpr[x]
		NameExpr[y]
`

	assertParseExpr(t, expected, src)
}

func TestLambdaOneParam(t *testing.T) {
	src := `lambda x: x + 1`
	expected := `
LambdaExpr
	Parameter
		NameExpr[x]
	BinaryExpr[+]
		NameExpr[x]
		NumberExpr[1]
`

	assertParseExpr(t, expected, src)
}

func TestLambdaAnonymousVararg(t *testing.T) {
	src := `lambda x, *, y: x + y`
	expected := `
LambdaExpr
	Parameter
		NameExpr[x]
	Parameter
		NameExpr[y]
	BinaryExpr[+]
		NameExpr[x]
		NameExpr[y]
`

	assertParseExpr(t, expected, src)
}

func TestLambdaNoParams(t *testing.T) {
	src := `lambda: 0`
	expected := `
LambdaExpr
	NumberExpr[0]
`

	assertParseExpr(t, expected, src)
}

func TestChainedAssignments(t *testing.T) {
	src := `a,b = c,d = 1,2`
	expected := `
Module
	AssignStmt
		TupleExpr
			NameExpr[a]
			NameExpr[b]
		TupleExpr
			NameExpr[c]
			NameExpr[d]
		TupleExpr
			NumberExpr[1]
			NumberExpr[2]
`

	assertParse(t, expected, src)
}

func TestAssignNakedYield(t *testing.T) {
	src := `x = yield`
	expected := `
Module
	AssignStmt
		NameExpr[x]
		YieldExpr
	`

	assertParse(t, expected, src)
}

func TestAssignPowYield(t *testing.T) {
	src := `x = (yield) ** 2`
	expected := `
Module
	AssignStmt
		NameExpr[x]
		BinaryExpr[**]
			YieldExpr
			NumberExpr[2]
	`

	assertParse(t, expected, src)
}

func TestChainedAnnotatedAssignment(t *testing.T) {
	src := `x: foo = y = 1`
	expected := `
Module
	BadStmt
`
	mod, _ := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})

	assertAST(t, expected, mod)
}

func TestAnnotatedAssignment(t *testing.T) {
	src := `x.y: foo = 1`
	expected := `
Module
	AssignStmt
		AttributeExpr[y]
			NameExpr[x]
		NameExpr[foo]
		NumberExpr[1]
`
	assertParse(t, expected, src)
}

func TestNakedAnnotation(t *testing.T) {
	src := `bar[baz]: foo`
	expected := `
Module
	AnnotationStmt
		IndexExpr
			NameExpr[bar]
			IndexSubscript
				NameExpr[baz]
		NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestEmptyTuples(t *testing.T) {
	src := `()<()>()`
	expected := `
BinaryExpr[<]
	TupleExpr
	BinaryExpr[>]
		TupleExpr
		TupleExpr
`
	assertParseExpr(t, expected, src)
}

func TestFunctionCallEquals(t *testing.T) {
	src := `foo(bar=car)`
	expected := `
Module
	ExprStmt
		CallExpr
			NameExpr[foo]
			Argument
				NameExpr[bar]
				NameExpr[car]
	`
	mod, err := Parse(kitectx.Background(), []byte(src), Options{})
	require.Nil(t, err)
	require.NotNil(t, mod)

	assertAST(t, expected, mod)

	exprStmt, ok := mod.Body[0].(*pythonast.ExprStmt)
	require.True(t, ok)

	call, ok := exprStmt.Value.(*pythonast.CallExpr)
	require.True(t, ok)

	require.NotNil(t, call.Args[0].Equals)
	assert.EqualValues(t, 7, call.Args[0].Equals.Begin)
	assert.EqualValues(t, 8, call.Args[0].Equals.End)
}

func TestAttributeExprDots(t *testing.T) {
	src := `foo.bar`
	expected := `
Module
	ExprStmt
		AttributeExpr[bar]
			NameExpr[foo]
	`
	mod, err := Parse(kitectx.Background(), []byte(src), Options{})
	require.Nil(t, err)
	require.NotNil(t, mod)

	assertAST(t, expected, mod)

	exprStmt, ok := mod.Body[0].(*pythonast.ExprStmt)
	require.True(t, ok)

	attrib, ok := exprStmt.Value.(*pythonast.AttributeExpr)
	require.True(t, ok)

	require.NotNil(t, attrib.Dot)
	assert.EqualValues(t, 3, attrib.Dot.Begin)
	assert.EqualValues(t, 4, attrib.Dot.End)
}

func TestAttributeExprDotsInDottedExpr(t *testing.T) {
	src := `
@foo.bar.car
class zar():
	pass
	`
	expected := `
Module
	ClassDefStmt
		AttributeExpr[car]
			AttributeExpr[bar]
				NameExpr[foo]
		NameExpr[zar]
		PassStmt
	`
	mod, err := Parse(kitectx.Background(), []byte(src), Options{})
	require.Nil(t, err)
	require.NotNil(t, mod)

	assertAST(t, expected, mod)

}

func TestBadStmtRecover(t *testing.T) {
	src := `
<<
return
`
	expected := `
Module
	BadStmt
	ReturnStmt
`

	mod, _ := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})

	assert.NotNil(t, mod)

	assertAST(t, expected, mod)

	assert.Equal(t, 2, len(mod.Body))

	stmt, bad := mod.Body[0].(*pythonast.BadStmt)
	assert.True(t, bad, "missing bad stmt")

	assert.EqualValues(t, 1, stmt.Begin())
	assert.EqualValues(t, 4, stmt.End())
}

func TestGoodStmtBadStmtNoOverlap(t *testing.T) {
	src := `
a = 1
b = wrong!
	`
	expected := `
Module
	AssignStmt
		NameExpr[a]
		NumberExpr[1]
	BadStmt
	`

	mod, _ := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})

	assertAST(t, expected, mod)

}

func TestNoDedentStartSmallStmt(t *testing.T) {
	// since we can sync up to a Dedent token we
	// need to check if this token is at the begining of parsing a small stmt
	// if so then we remove the Dedent and post an error and resync, if we do not do this then
	// parser ends up in state where it keeps trying to sync, but since the current token is a Dedent
	//sync does not make any progress and eventually we max out the number of recoveries made and
	// a nil module is returned.

	src := `
class foo():
	a = 1
		c=2
 	b = wrong^
	`

	expected := `
Module
	ClassDefStmt
		NameExpr[foo]
		AssignStmt
			NameExpr[a]
			NumberExpr[1]
		BadStmt
	BadStmt
	BadStmt
	`

	mod, _ := Parse(kitectx.Background(), []byte(src), Options{
		ErrorMode: Recover,
	})

	assertAST(t, expected, mod)
}

func TestAttributeExprAtCursor(t *testing.T) {
	src := `
foo.
print bar
`

	expected := `
Module
	ExprStmt
		AttributeExpr[Cursor]
			NameExpr[foo]
	PrintStmt
		NameExpr[bar]
	`

	cursor := token.Pos(5)
	mod, _ := Parse(kitectx.Background(), []byte(src), Options{
		Approximate: true,
		Cursor:      &cursor,
	})

	assertAST(t, expected, mod)
}

func TestCallExprAtCursor(t *testing.T) {
	src := `foo([1,2,3],`

	expected := `
Module
	BadStmt
		ExprStmt
			CallExpr
				NameExpr[foo]
				Argument
					ListExpr
						NumberExpr[1]
						NumberExpr[2]
						NumberExpr[3]
				Argument
					BadExpr
`

	cursor := token.Pos(12)
	mod, _ := Parse(kitectx.Background(), []byte(src), Options{
		Approximate: true,
		Cursor:      &cursor,
	})

	assertAST(t, expected, mod)
}

func TestCallAndAttrAtCursor(t *testing.T) {
	src := `foo(bar.`
	expected := `
Module
	BadStmt
`
	cursor := token.Pos(len(src))

	mod, _ := Parse(kitectx.Background(), []byte(src), Options{
		Cursor:    &cursor,
		ErrorMode: Recover,
	})

	assertAST(t, expected, mod)
}

func TestChainedCall(t *testing.T) {
	src := `foo().bar`
	expected := `
Module
	ExprStmt
		AttributeExpr[bar]
			CallExpr
				NameExpr[foo]
	`

	mod, err := Parse(kitectx.Background(), []byte(src), Options{
		Approximate: true,
	})

	assert.Nil(t, err)

	assertAST(t, expected, mod)
}

func TestClassDef_GetBases(t *testing.T) {
	src := `class Foo(a, b, metaclass=c): pass`
	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)

	class := mod.Body[0].(*pythonast.ClassDefStmt)
	bases := class.Bases()
	assert.Len(t, bases, 2)
}

func TestHellish(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/hellish.py")
	require.NoError(t, err)

	expected, err := ioutil.ReadFile("testdata/hellish.ast")
	require.NoError(t, err)

	assertParse(t, string(expected), string(src))
}

func TestUnicode(t *testing.T) {
	src := `
M = '−' #unicode character
c = '<no description>'`

	expected := `
Module
	AssignStmt
		NameExpr[M]
		StringExpr['−']
	AssignStmt
		NameExpr[c]
		StringExpr['<no description>']
	`

	assertParse(t, expected, src)
}

func TestEllipsis(t *testing.T) {
	src := `
x = ...
...
	`
	expected := `
Module
	AssignStmt
		NameExpr[x]
		EllipsisExpr
	ExprStmt
		EllipsisExpr
`

	assertParse(t, expected, src)
}

func TestAsyncFunctionDef(t *testing.T) {
	src := `
async def foo():
    pass
`
	expected := `
Module
	FunctionDefStmt
		NameExpr[foo]
		PassStmt
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.FunctionDefStmt{}, mod.Body[0])

	fd := mod.Body[0].(*pythonast.FunctionDefStmt)
	require.NotNil(t, fd.Async)

	assert.EqualValues(t, 1, fd.Begin())
	assert.EqualValues(t, len(src)-1, fd.End())
}

func TestAsyncDecoratedFunctionDef(t *testing.T) {
	src := `
@bar
async def foo():
    pass
`
	expected := `
Module
	FunctionDefStmt
		NameExpr[bar]
		NameExpr[foo]
		PassStmt
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.FunctionDefStmt{}, mod.Body[0])

	fd := mod.Body[0].(*pythonast.FunctionDefStmt)
	require.NotNil(t, fd.Async)

	// decorators are AttributeExpr and the @ sign is dropped, so starts at 2
	assert.EqualValues(t, 2, fd.Begin())
	assert.EqualValues(t, len(src)-1, fd.End())
}

func TestAsyncWithStmt(t *testing.T) {
	src := `
async with x as y:
    pass
`
	expected := `
Module
	WithStmt
		WithItem
			NameExpr[x]
			NameExpr[y]
		PassStmt
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.WithStmt{}, mod.Body[0])

	with := mod.Body[0].(*pythonast.WithStmt)
	require.NotNil(t, with.Async)

	assert.EqualValues(t, 1, with.Begin())
	assert.EqualValues(t, len(src)-1, with.End())
}

func TestAsyncForStmt(t *testing.T) {
	src := `
async for x in y:
    pass
`
	expected := `
Module
	ForStmt
		NameExpr[x]
		NameExpr[y]
		PassStmt
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.ForStmt{}, mod.Body[0])

	forStmt := mod.Body[0].(*pythonast.ForStmt)
	require.NotNil(t, forStmt.Async)

	assert.EqualValues(t, 1, forStmt.Begin())
	assert.EqualValues(t, len(src)-1, forStmt.End())
}

func TestAsyncInvalidStmt(t *testing.T) {
	src := `
async if x:
    pass
`
	_, err := Parse(kitectx.Background(), []byte(src), opts)
	require.Error(t, err)
}

func TestAsyncInvalidDecorated(t *testing.T) {
	src := `
@foo
async with x:
    pass
`
	_, err := Parse(kitectx.Background(), []byte(src), opts)
	require.Error(t, err)
}

func TestAsyncForComp(t *testing.T) {
	src := `
g = (x async for i in f())
`
	expected := `
Module
	AssignStmt
		NameExpr[g]
		ComprehensionExpr
			NameExpr[x]
			Generator
				NameExpr[i]
				CallExpr
					NameExpr[f]
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.AssignStmt{}, mod.Body[0])

	assign := mod.Body[0].(*pythonast.AssignStmt)
	require.IsType(t, &pythonast.ComprehensionExpr{}, assign.Value)
	comp := assign.Value.(*pythonast.ComprehensionExpr)

	require.Len(t, comp.BaseComprehension.Generators, 1)
	gen := comp.BaseComprehension.Generators[0]
	require.NotNil(t, gen.Async)

	assert.EqualValues(t, 8, gen.Begin())
	assert.EqualValues(t, 26, gen.End())
}

func TestAsyncForListComp(t *testing.T) {
	src := `
g = [x async for i in f()]
`
	expected := `
Module
	AssignStmt
		NameExpr[g]
		ListComprehensionExpr
			NameExpr[x]
			Generator
				NameExpr[i]
				CallExpr
					NameExpr[f]
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.AssignStmt{}, mod.Body[0])

	assign := mod.Body[0].(*pythonast.AssignStmt)
	require.IsType(t, &pythonast.ListComprehensionExpr{}, assign.Value)
	comp := assign.Value.(*pythonast.ListComprehensionExpr)

	require.Len(t, comp.BaseComprehension.Generators, 1)
	gen := comp.BaseComprehension.Generators[0]
	require.NotNil(t, gen.Async)

	assert.EqualValues(t, 8, gen.Begin())
	assert.EqualValues(t, 26, gen.End())
}

func TestAsyncForSetComp(t *testing.T) {
	src := `
g = {x async for i in f()}
`
	expected := `
Module
	AssignStmt
		NameExpr[g]
		SetComprehensionExpr
			NameExpr[x]
			Generator
				NameExpr[i]
				CallExpr
					NameExpr[f]
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.AssignStmt{}, mod.Body[0])

	assign := mod.Body[0].(*pythonast.AssignStmt)
	require.IsType(t, &pythonast.SetComprehensionExpr{}, assign.Value)
	comp := assign.Value.(*pythonast.SetComprehensionExpr)

	require.Len(t, comp.BaseComprehension.Generators, 1)
	gen := comp.BaseComprehension.Generators[0]
	require.NotNil(t, gen.Async)

	assert.EqualValues(t, 8, gen.Begin())
	assert.EqualValues(t, 26, gen.End())
}

func TestAsyncForDictComp(t *testing.T) {
	src := `
g = {x:y async for i in f()}
`
	expected := `
Module
	AssignStmt
		NameExpr[g]
		DictComprehensionExpr
			NameExpr[x]
			NameExpr[y]
			Generator
				NameExpr[i]
				CallExpr
					NameExpr[f]
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.AssignStmt{}, mod.Body[0])

	assign := mod.Body[0].(*pythonast.AssignStmt)
	require.IsType(t, &pythonast.DictComprehensionExpr{}, assign.Value)
	comp := assign.Value.(*pythonast.DictComprehensionExpr)

	require.Len(t, comp.BaseComprehension.Generators, 1)
	gen := comp.BaseComprehension.Generators[0]
	require.NotNil(t, gen.Async)

	assert.EqualValues(t, 10, gen.Begin())
	assert.EqualValues(t, 28, gen.End())
}

func TestAsyncForCompMultiGenerators(t *testing.T) {
	src := `
g = (x for i in f() async for x in i if g())
`
	expected := `
Module
	AssignStmt
		NameExpr[g]
		ComprehensionExpr
			NameExpr[x]
			Generator
				NameExpr[i]
				CallExpr
					NameExpr[f]
			Generator
				NameExpr[x]
				NameExpr[i]
				CallExpr
					NameExpr[g]
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.AssignStmt{}, mod.Body[0])

	assign := mod.Body[0].(*pythonast.AssignStmt)
	require.IsType(t, &pythonast.ComprehensionExpr{}, assign.Value)
	comp := assign.Value.(*pythonast.ComprehensionExpr)

	require.Len(t, comp.BaseComprehension.Generators, 2)
	gen := comp.BaseComprehension.Generators[0]
	require.Nil(t, gen.Async)
	gen = comp.BaseComprehension.Generators[1]
	require.NotNil(t, gen.Async)
}

func TestAwaitCall(t *testing.T) {
	src := `
await f()
`
	expected := `
Module
	ExprStmt
		AwaitExpr
			CallExpr
				NameExpr[f]
`

	mod, err := Parse(kitectx.Background(), []byte(src), opts)
	require.NoError(t, err)
	assertAST(t, expected, mod)

	require.Len(t, mod.Body, 1)
	require.IsType(t, &pythonast.ExprStmt{}, mod.Body[0])
	stmt := mod.Body[0].(*pythonast.ExprStmt)

	require.IsType(t, &pythonast.AwaitExpr{}, stmt.Value)
	await := stmt.Value.(*pythonast.AwaitExpr)

	assert.EqualValues(t, 1, await.Begin())
	assert.EqualValues(t, 10, await.End())
}

func TestAwaitInList(t *testing.T) {
	src := `
[a, await f()]
`
	expected := `
Module
	ExprStmt
		ListExpr
			NameExpr[a]
			AwaitExpr
				CallExpr
					NameExpr[f]
`

	assertParse(t, expected, src)
}

func TestAwaitAsyncComp(t *testing.T) {
	// example from PEP-0530
	src := `
[await fun() async for fun in funcs if await smth]
`
	expected := `
Module
	ExprStmt
		ListComprehensionExpr
			AwaitExpr
				CallExpr
					NameExpr[fun]
			Generator
				NameExpr[fun]
				NameExpr[funcs]
				AwaitExpr
					NameExpr[smth]
`

	assertParse(t, expected, src)
}

func TestKeywordsAsInvalidIdent(t *testing.T) {
	for k := range pythonscanner.Keywords {
		t.Run(k, func(t *testing.T) {
			src := fmt.Sprintf("%s = 1", k)
			_, err := Parse(kitectx.Background(), []byte(src), opts)
			require.Error(t, err)
		})
	}
}

func TestNonNameKeywordArg(t *testing.T) {
	// Bug from issue https://github.com/kiteco/kiteco/issues/7528
	src := `
x.fn(self.v=1)
`
	_, err := Parse(kitectx.Background(), []byte(src), opts)
	require.Error(t, err)
}

func TestValidKeywordArg(t *testing.T) {
	src := `
x.fn(v=1)
`
	expected := `
Module
	ExprStmt
		CallExpr
			AttributeExpr[fn]
				NameExpr[x]
			Argument
				NameExpr[v]
				NumberExpr[1]
`

	assertParse(t, expected, src)
}
