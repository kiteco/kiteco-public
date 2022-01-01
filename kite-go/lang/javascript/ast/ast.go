package ast

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
)

// Type of an ast node.
type Type string

const (
	// -- Declarations

	// FunctionDeclaration represents a function declaration.
	FunctionDeclaration = "FunctionDeclaration"
	// ClassDeclaration represents a class declaration.
	ClassDeclaration = "ClassDeclaration"
	// ImportDeclaration represents an import declaration.
	ImportDeclaration = "ImportDeclaration"
	// ExportDeclaration represents an export declaration.
	ExportDeclaration = "ExportDeclaration"
	// ModuleDeclaration represents a module declaration.
	ModuleDeclaration = "ModuleDeclaration"

	// -- Statements

	// VariableStatement represents a variable declaration.
	VariableStatement Type = "VariableStatement"
	// EmptyStatement represents an empty statement.
	EmptyStatement = "EmptyStatement"
	// ExpressionStatement represents an expression statement.
	ExpressionStatement = "ExpressionStatement"
	// IfStatement represents an if statement.
	IfStatement = "IfStatement"
	// WhileStatement represents a while statement.
	WhileStatement = "WhileStatement"
	// ForStatement represents a for statement.
	ForStatement = "ForStatement"
	// ForInStatement represents a for in statement.
	ForInStatement = "ForInStatement"
	// ForOfStatement represents a for of statement.
	ForOfStatement = "ForOfStatement"
	// ContinueStatement represents a continue statement.
	ContinueStatement = "ContinueStatement"
	// BreakStatement represents a break statement.
	BreakStatement = "BreakStatement"
	// ReturnStatement represents a return statement.
	ReturnStatement = "ReturnStatement"
	// WithStatement represents a with statement.
	WithStatement = "WithStatement"
	// LabelledStatement represents a labelled statement.
	LabelledStatement = "LabelledStatement"
	// SwitchStatement represents a switch statement.
	SwitchStatement = "SwitchStatement"
	// ThrowStatement represents a throw statement.
	ThrowStatement = "ThrowStatement"
	// TryStatement represents a try statement.
	TryStatement = "TryStatement"
	// DebuggerStatement represents a debug statement.
	DebuggerStatement = "DebuggerStatement"

	// -- Expressions

	// Identifier represents an identifier expression.
	Identifier = "Identifier"
	// NullLiteral represents a null literal expression.
	NullLiteral = "NullLiteral"
	// BooleanLiteral represents a boolean literal expression.
	BooleanLiteral = "BooleanLiteral"
	// DecimalLiteral represents a decimal literal expression.
	DecimalLiteral = "DecimalLiteral"
	// HexIntegerLiteral represents a hex integer literal expression.
	HexIntegerLiteral = "HexIntegerLiteral"
	// StringLiteral represents a string literal expression.
	StringLiteral = "StringLiteral"
	// RegularExpressionLiteral represents a regular expression literal.
	RegularExpressionLiteral = "RegularExpressionLiteral"
	// ArrayLiteral represents an array literal expression.
	ArrayLiteral = "ArrayLiteral"
	// ObjectLiteral represents an object literal expression.
	ObjectLiteral = "ObjectLiteral"
	// SuperCall represents a super call expression.
	SuperCall = "SuperCall"
	// SuperMember represents a super member expression.
	SuperMember = "SuperMember"
	// NewExpression represents a new expression.
	NewExpression = "NewExpression"
	// Call represents a call expression.
	Call = "Call"
	// MemberExpression represents a member expression.
	MemberExpression = "MemberExpression"
	// ThisExpression represents a this expression.
	ThisExpression = "ThisExpression"
	// UnaryExpression represents a unary expression.
	UnaryExpression = "UnaryExpression"
	// AssignmentExpression represents an assignment expression.
	AssignmentExpression = "AssignmentExpression"
	// ConditionalExpression represents a conditional expression.
	ConditionalExpression = "ConditionalExpression"
	// BinaryExpression represents a binary expression,
	// note we make no distinction between logical expression (&&,||)
	// and "true" binary expressions (+,-,|,&...etc) this means
	// that the precedence between logical and binary expressions
	// is ignored.
	BinaryExpression = "BinaryExpression"
	// YieldExpression represents a yield expression.
	YieldExpression = "YieldExpression"
	// AwaitExpression represents an await expression.
	AwaitExpression = "AwaitExpression"
	// SequenceExpression represents a sequence expression.
	SequenceExpression = "SequenceExpression"
	// FunctionExpression represents a function expression.
	FunctionExpression = "FunctionExpression"
	// ArrowFunction represents an arrow function expression.
	ArrowFunction = "ArrowFunction"
	// ClassExpression represents a class expression.
	ClassExpression = "ClassExpression"

	// -- Other nodes

	// Extends for a class declaration or expression.
	Extends = "Extends"

	// Name for a class declaration, class expression,
	// function declaration, function expression.
	Name = "Name"

	// ClassBody for a class declaration or expression.
	ClassBody = "ClassBody"

	// ComputedProperty represents a propery name that was computed
	// in an object literal or object pattern.
	// e.g `[key]` in let {[key]} = foo or let o = {[key]:1}
	ComputedProperty = "ComputedProperty"

	// SpreadElement represents a spread element.
	SpreadElement = "SpreadElement"

	// ObjectPattern represents an object pattern,
	// e.g let {a:b} = foo `{a:b}` is the object pattern.
	ObjectPattern = "ObjectPattern"

	// ArrayPattern represents an array pattern,
	// e.g let [a,b] = foo `[a,b]` is the array pattern.
	ArrayPattern = "ArrayPattern"

	// Elision represents an elision in an array literal or pattern,
	// e.g [a,,,].
	Elision = "Elision"

	// RestElement represents a rest element expression in an array literal or pattern,
	// e.g [...a].
	RestElement = "RestElement"

	// AssignmentPatternDefault represents an assignment pattern expression
	// that assigns a default value.
	// e.g `c=1` in let {c=1} = foo.
	AssignmentPatternDefault = "AssignmentPatternDefault"

	// AssignmentPattern represents an assignment pattern expression
	// that either reassigns a property name or does nested destructurin/
	// e.g `c:d` and `e:{f,g}` in let {c:d, e:{f,g}} = foo.
	AssignmentPattern = "AssignmentPattern"

	// Property represents a property assignment when initializing an object,
	// e.g var f = {a:1} `a:1` is the propery.
	Property = "Property"

	// Method represents a method assignment when initializing an object (or in a class declaration),
	// e.g var f = {a:function(){...}} `a:function(){...}` is the method.
	Method = "Method"

	// Arguments represents the arguments to a call.
	Arguments = "Arguments"

	// VariableDeclaration represents the "declaration" portion of a VariableStatement,
	// e.g let a = 1 `a = 1` is the variable declaration.
	VariableDeclaration = "VariableDeclaration"

	// CaseClause represents a case clause in a switch statement.
	CaseClause = "CaseClause"

	// DefaultClause represents a default clause in a switch statement.
	DefaultClause = "DefaultClause"

	// TryBlock represents a block in a try statement.
	TryBlock = "TryBlock"

	// CatchBlock represents a catch block.
	CatchBlock = "CatchBlock"

	// FinallyBlock represents a finally block in a try statement.
	FinallyBlock = "FinallyBlock"

	// FunctionBody represents the body of a function (declaration/expression/arrow).
	FunctionBody = "FunctionBody"

	// FormalParameterList represents the formal parameter list for a function (declaration/expression/arrow).
	FormalParameterList = "FormalParameterList"

	// NameAs represents a "name as" node in an import or export declaration,
	// e.g import foo as bar from "car" `foo as bar` is the name as.
	NameAs = "NameAs"

	// Star represents a "*" in an import declaration,
	// e.g import * as bar from "car" `*` is the star.
	Star = "Star"

	// NameAsList represents a list of "name as" nodes.
	NameAsList = "NameAsList"
)

// Node in a javascript AST
type Node struct {
	// Begin is the byte offset (from the start of the file)
	// to the beginning of the node (inclusive, 0 based).
	Begin token.Pos
	// End is the byte offset (from the start of the file)
	// to the end of the node (exclusive, 0 based).
	End token.Pos
	// Type of the ast node.
	Type Type
	// Literal source of the node.
	Literal []byte
	// Children ast nodes.
	Children []*Node
}

// String representation of the node.
func (n *Node) String() string {
	var lit string
	switch n.Type {
	case Identifier:
		lit = "[" + string(n.Literal) + "]"
	default:
		if lit = Literal(n); lit != "" {
			lit = "[" + lit + "]"
		}
	}
	return fmt.Sprintf("%s%s", n.Type, lit)
}

// IsStatement returns true if the provided
// node represents a statement.
func IsStatement(node *Node) bool {
	if node == nil {
		return false
	}
	switch node.Type {
	case VariableStatement, EmptyStatement, ExpressionStatement,
		IfStatement, WhileStatement, ForStatement, ForInStatement,
		ForOfStatement, ContinueStatement, BreakStatement, ReturnStatement,
		WithStatement, LabelledStatement, SwitchStatement, ThrowStatement,
		TryStatement, DebuggerStatement:
		return true
	default:
		return false
	}
}

// IsDeclaration returns true if the provided node
// represents a declaration.
func IsDeclaration(node *Node) bool {
	if node == nil {
		return false
	}

	switch node.Type {
	case ModuleDeclaration, FunctionDeclaration, ClassDeclaration,
		ImportDeclaration, ExportDeclaration:
		return true
	default:
		return false
	}
}

// NameFor a ClassDeclaration, FunctionDeclaration, ClassExpression, FunctionExpression.
func NameFor(node *Node) *Node {
	switch node.Type {
	case ClassDeclaration, ClassExpression,
		FunctionDeclaration, FunctionExpression:
		if len(node.Children[0].Children) > 1 {
			return node.Children[0].Children[0]
		}
		return nil
	default:
		return nil
	}
}

// Literal from a ast literal node
func Literal(node *Node) string {
	switch node.Type {
	case StringLiteral:
		return strings.TrimFunc(string(node.Literal), func(r rune) bool {
			if unicode.IsSpace(r) {
				return true
			}
			if unicode.IsOneOf([]*unicode.RangeTable{unicode.Quotation_Mark}, r) {
				return true
			}
			return r == '`'
		})
	case DecimalLiteral:
		return strings.TrimFunc(string(node.Literal), unicode.IsSpace)
	case BooleanLiteral:
		return strings.TrimFunc(string(node.Literal), unicode.IsSpace)
	case HexIntegerLiteral:
		return strings.TrimFunc(string(node.Literal), unicode.IsSpace)
	case RegularExpressionLiteral:
		return strings.TrimFunc(string(node.Literal), unicode.IsSpace)
	default:
		return ""
	}
}
