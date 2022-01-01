package parser

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang/javascript/ast"
	"github.com/stretchr/testify/require"
)

func assertAST(t *testing.T, expected string, node *ast.Node, positions bool) {
	var buf bytes.Buffer
	if positions {
		ast.PrintPositions(node, &buf, "\t")
	} else {
		ast.Print(node, &buf, "\t")
	}

	actual := buf.String()

	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

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

		// trim trailing whitespace
		actualLine = strings.TrimRightFunc(actualLine, unicode.IsSpace)
		expectedLine = strings.TrimRightFunc(expectedLine, unicode.IsSpace)

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

	if errorLine > -1 {
		t.Errorf("expected %s but got %s (line %d):\n%s", errorExpected, errorActual, errorLine, sidebyside)
	}

	t.Log("\n" + actual)
}

func assertParse(t *testing.T, src, expected string, positions bool) *ast.Node {
	node, err := Parse([]byte(src), DefaultOptions)
	require.NoError(t, err)
	require.NotNil(t, node)
	assertAST(t, expected, node, positions)
	return node
}

func TestParser_Literal(t *testing.T) {
	src := `
true
false
null
this
`
	expected := `
ModuleDeclaration
	ExpressionStatement
		BooleanLiteral[true]
	ExpressionStatement
		BooleanLiteral[false]
	ExpressionStatement
		NullLiteral
	ExpressionStatement
		ThisExpression	
	`

	assertParse(t, src, expected, false)
}

func TestParser_Identifier(t *testing.T) {
	src := `
bar
foo
`

	expected := `
ModuleDeclaration[0...9]
	ExpressionStatement[1...5]
		Identifier[bar][1...4]
	ExpressionStatement[5...9]
		Identifier[foo][5...8]
`
	assertParse(t, src, expected, true)
}

func TestParser_MemberExpression(t *testing.T) {
	src := `
foo.bar.car	
	`
	expected := `
ModuleDeclaration
	ExpressionStatement
		MemberExpression
			MemberExpression
				Identifier[foo]
				Identifier[bar]
			Identifier[car]
`

	assertParse(t, src, expected, false)
}

func TestParser_BinaryExpression(t *testing.T) {
	src := `foo | bar & car`

	expected := `
ModuleDeclaration
	ExpressionStatement
		BinaryExpression
			Identifier[foo]
			BinaryExpression
				Identifier[bar]
				Identifier[car]
	`

	assertParse(t, src, expected, false)
}

func TestParser_LogicalExpression(t *testing.T) {
	src := `foo || bar && car`

	expected := `
ModuleDeclaration
	ExpressionStatement
		BinaryExpression
			Identifier[foo]
			BinaryExpression
				Identifier[bar]
				Identifier[car]
	`

	assertParse(t, src, expected, false)
}

func TestParser_Precedence(t *testing.T) {
	// NOTE: we ignore precedence between
	// binary operations and logical operations
	// and collapse them all into BinaryExpression nodes

	src := `foo | bar || car & mar && far`

	expected := `
ModuleDeclaration
	ExpressionStatement
		BinaryExpression
			Identifier[foo]
			BinaryExpression
				Identifier[bar]
				BinaryExpression
					Identifier[car]
					BinaryExpression
						Identifier[mar]
						Identifier[far]	
	`

	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression(t *testing.T) {
	src := `foo = 1`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			Identifier[foo]
			DecimalLiteral[1]
	`
	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression_ObjectPattern(t *testing.T) {
	src := `
({a,b, e:f, [g]:h, [key], ...rest} = foo)
	`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			ObjectPattern
				Identifier[a]
				Identifier[b]
				AssignmentPattern
					Identifier[e]
					Identifier[f]
				AssignmentPattern
					ComputedProperty
						Identifier[g]
					Identifier[h]
				ComputedProperty
					Identifier[key]
				RestElement
					Identifier[rest]
			Identifier[foo]
	`

	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression_ObjectPattern_Nested(t *testing.T) {
	src := `
({full:{first:first, last:last}, dob:[m,d,y]} = person)
`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			ObjectPattern
				AssignmentPattern
					Identifier[full]
					ObjectPattern
						AssignmentPattern
							Identifier[first]
							Identifier[first]
						AssignmentPattern
							Identifier[last]
							Identifier[last]
				AssignmentPattern
					Identifier[dob]
					ArrayPattern
						Identifier[m]
						Identifier[d]
						Identifier[y]
			Identifier[person]
	`

	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression_ObjectPattern_Defaults(t *testing.T) {
	src := `
({id=0, name={first:"unknown", last:"unknown"}, dob=[0,0,0]} = person)
	`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			ObjectPattern
				AssignmentPatternDefault
					Identifier[id]
					DecimalLiteral[0]
				AssignmentPatternDefault
					Identifier[name]
					ObjectLiteral
						Property
							Identifier[first]
							StringLiteral[unknown]
						Property
							Identifier[last]
							StringLiteral[unknown]
				AssignmentPatternDefault
					Identifier[dob]
					ArrayLiteral
						DecimalLiteral[0]
						DecimalLiteral[0]
						DecimalLiteral[0]
			Identifier[person]
	`

	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression_ArrayPattern(t *testing.T) {
	src := `
	([a, b,,, c ,,, ...d] = foo)
		`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			ArrayPattern
				Identifier[a]
				Identifier[b]
				Elision
				Identifier[c]
				Elision
				RestElement
					Identifier[d]
			Identifier[foo]
		`

	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression_ArrayPattern_Nested(t *testing.T) {
	src := `
	([[a]] = foo)
		`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			ArrayPattern
				ArrayPattern
					Identifier[a]
			Identifier[foo]
		`

	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression_ArrayPattern_AssignmentPattern(t *testing.T) {
	src := `
([a=[1,2],[b], c = {foo:bar}] = foo)
	`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			ArrayPattern
				AssignmentPatternDefault
					Identifier[a]
					ArrayLiteral
						DecimalLiteral[1]
						DecimalLiteral[2]
				ArrayPattern
					Identifier[b]
				AssignmentPatternDefault
					Identifier[c]
					ObjectLiteral
						Property
							Identifier[foo]
							Identifier[bar]
			Identifier[foo]
	`

	assertParse(t, src, expected, false)
}

func TestParser_AssignmentExpression_ArrayPattern_ObjectPattern(t *testing.T) {
	src := `
([{name:{first, last}}, ...rest] = person)
	`

	expected := `
ModuleDeclaration
	ExpressionStatement
		AssignmentExpression
			ArrayPattern
				ObjectPattern
					AssignmentPattern
						Identifier[name]
						ObjectPattern
							Identifier[first]
							Identifier[last]
				RestElement
					Identifier[rest]
			Identifier[person]
	`

	assertParse(t, src, expected, false)
}

func TestParser_ConditionalExpression(t *testing.T) {
	src := `bar ? car : far`

	expected := `
ModuleDeclaration
	ExpressionStatement
		ConditionalExpression
			Identifier[bar]
			Identifier[car]
			Identifier[far]	
	`

	assertParse(t, src, expected, false)
}

func TestParser_UnaryExpression(t *testing.T) {
	src := `
!foo
bar++	
	`

	expected := `
ModuleDeclaration
	ExpressionStatement
		UnaryExpression
			Identifier[foo]
	ExpressionStatement
		UnaryExpression
			Identifier[bar]	
	`

	assertParse(t, src, expected, false)
}
func TestParser_ImportDeclaration(t *testing.T) {
	src := `
import {Foo as foo, Bar as bar, car} from 'module'
import 'sideeffects'
import Foo from 'module'
import * as foo from 'module'
	`

	expected := `
ModuleDeclaration
	ImportDeclaration
		NameAsList
			NameAs
				Identifier[Foo]
				Identifier[foo]
			Elision
			NameAs
				Identifier[Bar]
				Identifier[bar]
			Elision
			NameAs
				Identifier[car]
		StringLiteral[module]
	ImportDeclaration
		StringLiteral[sideeffects]
	ImportDeclaration
		NameAsList
			NameAs
				Identifier[Foo]
		StringLiteral[module]
	ImportDeclaration
		NameAsList
			NameAs
				Star
				Identifier[foo]
		StringLiteral[module]
	`

	assertParse(t, src, expected, false)
}

func TestParser_VariableStatement(t *testing.T) {
	src := `
let l1 = 1, l2 = "hello"
const c1 = 2, c2 = 'world'
var v1 = 3, v2 = '!'
`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			Identifier[l1]
			DecimalLiteral[1]
		VariableDeclaration
			Identifier[l2]
			StringLiteral[hello]
	VariableStatement
		VariableDeclaration
			Identifier[c1]
			DecimalLiteral[2]
		VariableDeclaration
			Identifier[c2]
			StringLiteral[world]
	VariableStatement
		VariableDeclaration
			Identifier[v1]
			DecimalLiteral[3]
		VariableDeclaration
			Identifier[v2]
			StringLiteral[!]
`

	assertParse(t, src, expected, false)
}

func TestParser_VariableStatement_ObjectPattern(t *testing.T) {
	src := `
let {a,b, e:f, [g]:h, [key], ...rest} = foo
	`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			ObjectPattern
				Identifier[a]
				Identifier[b]
				AssignmentPattern
					Identifier[e]
					Identifier[f]
				AssignmentPattern
					ComputedProperty
						Identifier[g]
					Identifier[h]
				ComputedProperty
					Identifier[key]
				RestElement
					Identifier[rest]
			Identifier[foo]
	`

	assertParse(t, src, expected, false)
}

func TestParser_VariableStatement_ObjectPattern_Nested(t *testing.T) {
	src := `
let {full:{first:first, last:last}, dob:[m,d,y]} = person
`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			ObjectPattern
				AssignmentPattern
					Identifier[full]
					ObjectPattern
						AssignmentPattern
							Identifier[first]
							Identifier[first]
						AssignmentPattern
							Identifier[last]
							Identifier[last]
				AssignmentPattern
					Identifier[dob]
					ArrayPattern
						Identifier[m]
						Identifier[d]
						Identifier[y]
			Identifier[person]
	`

	assertParse(t, src, expected, false)
}

func TestParser_VariableStatement_ObjectPattern_Defaults(t *testing.T) {
	src := `
let {id=0, name={first:"unknown", last:"unknown"}, dob=[0,0,0]} = person
	`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			ObjectPattern
				AssignmentPatternDefault
					Identifier[id]
					DecimalLiteral[0]
				AssignmentPatternDefault
					Identifier[name]
					ObjectLiteral
						Property
							Identifier[first]
							StringLiteral[unknown]
						Property
							Identifier[last]
							StringLiteral[unknown]
				AssignmentPatternDefault
					Identifier[dob]
					ArrayLiteral
						DecimalLiteral[0]
						DecimalLiteral[0]
						DecimalLiteral[0]
			Identifier[person]
	`

	assertParse(t, src, expected, false)
}

func TestParser_VariableStatement_ArrayPattern(t *testing.T) {
	src := `
	let [a, b,,, c ,,, ...d] = foo
		`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			ArrayPattern
				Identifier[a]
				Identifier[b]
				Elision
				Identifier[c]
				Elision
				RestElement
					Identifier[d]
			Identifier[foo]
		`

	assertParse(t, src, expected, false)
}

func TestParser_VariableDeclaration_ArrayPattern_Nested(t *testing.T) {
	src := `
let [[a]] = foo
`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			ArrayPattern
				ArrayPattern
					Identifier[a]
			Identifier[foo]
		`

	assertParse(t, src, expected, false)
}

func TestParser_VariableStatement_ArrayPattern_AssignmentPattern(t *testing.T) {
	src := `
let [a=[1,2],[b], c = {foo:bar}] = foo
	`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			ArrayPattern
				AssignmentPatternDefault
					Identifier[a]
					ArrayLiteral
						DecimalLiteral[1]
						DecimalLiteral[2]
				ArrayPattern
					Identifier[b]
				AssignmentPatternDefault
					Identifier[c]
					ObjectLiteral
						Property
							Identifier[foo]
							Identifier[bar]
			Identifier[foo]
	`

	assertParse(t, src, expected, false)
}

func TestParser_VariableStatement_ArrayPattern_ObjectPattern(t *testing.T) {
	src := `
let [{name:{first, last}}, ...rest] = person
	`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			ArrayPattern
				ObjectPattern
					AssignmentPattern
						Identifier[name]
						ObjectPattern
							Identifier[first]
							Identifier[last]
				RestElement
					Identifier[rest]
			Identifier[person]
	`

	assertParse(t, src, expected, false)
}

func TestParser_ArrayLiteral(t *testing.T) {
	src := `[a,b,,, c ,{a:['hello']}, [2,true], [,,,], ...d]`

	expected := `
ModuleDeclaration
	ExpressionStatement
		ArrayLiteral
			Identifier[a]
			Identifier[b]
			Elision
			Identifier[c]
			ObjectLiteral
				Property
					Identifier[a]
					ArrayLiteral
						StringLiteral[hello]
			ArrayLiteral
				DecimalLiteral[2]
				BooleanLiteral[true]
			ArrayLiteral
				Elision
			SpreadElement
				Identifier[d]
	`

	assertParse(t, src, expected, false)
}

func TestParser_FunctionDeclaration(t *testing.T) {
	src := `
function bar(car, star) {
	bar()
}
	`

	expected := `
ModuleDeclaration
	FunctionDeclaration
		Name
			Identifier[bar]
		FormalParameterList
			Identifier[car]
			Identifier[star]
		FunctionBody
			ExpressionStatement
				Call
					Identifier[bar]
					Arguments
	`

	assertParse(t, src, expected, false)
}

func TestParser_FunctionDeclaration_Destructuring(t *testing.T) {
	src := `
function printName({id=0, DOB:[d,m,y], 'name':{'first':first, 'last':last}}, format, [a,b,c]) {}
	`

	expected := `
ModuleDeclaration
	FunctionDeclaration
		Name
			Identifier[printName]
		FormalParameterList
			ObjectPattern
				AssignmentPatternDefault
					Identifier[id]
					DecimalLiteral[0]
				AssignmentPattern
					Identifier[DOB]
					ArrayPattern
						Identifier[d]
						Identifier[m]
						Identifier[y]
				AssignmentPattern
					StringLiteral[name]
					ObjectPattern
						AssignmentPattern
							StringLiteral[first]
							Identifier[first]
						AssignmentPattern
							StringLiteral[last]
							Identifier[last]
			Identifier[format]
			ArrayPattern
				Identifier[a]
				Identifier[b]
				Identifier[c]
		FunctionBody
	`

	assertParse(t, src, expected, false)
}

func TestParser_FunctionExpression(t *testing.T) {
	src := `
let bar = function bar(car, star) {
	bar()
}

let car = function(car,star) {
	car()
}
	`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			Identifier[bar]
			FunctionExpression
				Name
					Identifier[bar]
				FormalParameterList
					Identifier[car]
					Identifier[star]
				FunctionBody
					ExpressionStatement
						Call
							Identifier[bar]
							Arguments
	VariableStatement
		VariableDeclaration
			Identifier[car]
			FunctionExpression
				Name
				FormalParameterList
					Identifier[car]
					Identifier[star]
				FunctionBody
					ExpressionStatement
						Call
							Identifier[car]
							Arguments
`

	assertParse(t, src, expected, false)
}

func TestParser_FunctionExpression_Destructuring(t *testing.T) {
	src := `
let printName = function({id=0, DOB:[d,m,y], 'name':{'first':first, 'last':last}}, format, [a,b,c]) {}
	`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			Identifier[printName]
			FunctionExpression
				Name
				FormalParameterList
					ObjectPattern
						AssignmentPatternDefault
							Identifier[id]
							DecimalLiteral[0]
						AssignmentPattern
							Identifier[DOB]
							ArrayPattern
								Identifier[d]
								Identifier[m]
								Identifier[y]
						AssignmentPattern
							StringLiteral[name]
							ObjectPattern
								AssignmentPattern
									StringLiteral[first]
									Identifier[first]
								AssignmentPattern
									StringLiteral[last]
									Identifier[last]
					Identifier[format]
					ArrayPattern
						Identifier[a]
						Identifier[b]
						Identifier[c]
				FunctionBody
	`

	assertParse(t, src, expected, false)
}

func TestParser_ObjectLiteral(t *testing.T) {
	src := `
let o = {
	a:[1,2],
	b: function(c){

	},
	[d]: e,
	'f': g,
	...obj,
	...obj1,
	[d]: e,
	'f': g,
	1:h,
	cls: class {},
}
	`

	expected := `
ModuleDeclaration
	VariableStatement
		VariableDeclaration
			Identifier[o]
			ObjectLiteral
				Property
					Identifier[a]
					ArrayLiteral
						DecimalLiteral[1]
						DecimalLiteral[2]
				Property
					Identifier[b]
					FunctionExpression
						Name
						FormalParameterList
							Identifier[c]
						FunctionBody
				Property
					ComputedProperty
						Identifier[d]
					Identifier[e]
				Property
					StringLiteral[f]
					Identifier[g]
				SpreadElement
					Identifier[obj]
				SpreadElement
					Identifier[obj1]
				Property
					ComputedProperty
						Identifier[d]
					Identifier[e]
				Property
					StringLiteral[f]
					Identifier[g]
				Property
					DecimalLiteral[1]
					Identifier[h]
				Property
					Identifier[cls]
					ClassExpression
						Name
						Extends
						ClassBody
	`

	assertParse(t, src, expected, false)
}

func TestParser_NewExpression(t *testing.T) {
	src := `
new Graph	
	`

	expected := `
ModuleDeclaration
	ExpressionStatement
		NewExpression
			Identifier[Graph]
			Arguments	
	`

	assertParse(t, src, expected, false)

}
