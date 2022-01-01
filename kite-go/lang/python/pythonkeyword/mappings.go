package pythonkeyword

import (
	"fmt"
	"reflect"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

// KeywordTokenToCat maps a keyword token to an integer in the range [1,30]
// The category is 1-indexed and define in the keywords map
// Category can be -1 for keyword ignored by the model
func KeywordTokenToCat(token pythonscanner.Token) int {
	if _, ok := AllKeywords[token]; !ok {
		panic(fmt.Sprintf("Keyword %d not registered in AllKeywords map", token))

	}
	return AllKeywords[token].Cat
}

// KeywordCatToToken maps an integer in [1,30] representing a keyword category to the corresponding keyword token.
func KeywordCatToToken(cat int) pythonscanner.Token {
	tok, ok := categoryToToken[cat]
	if !ok {
		panic(fmt.Sprintf("keyword category out of range: %d", cat))
	}
	return tok
}

// NodeToCat returns a category for a python AST node.
func NodeToCat(node pythonast.Node) int {
	if pythonast.IsNil(node) {
		return 0
	}
	typ := reflect.ValueOf(node).Type()
	cat, ok := nodeTypeCategories[typ]
	if !ok {
		panic(fmt.Sprintf("unrecognized node type: %v", typ))
	}
	return cat
}

// AllNodeTypes returns a slice of the names of all node types, in which the index of each type is its corresponding
// category.
func AllNodeTypes() []string {
	var max int
	for _, cat := range nodeTypeCategories {
		if cat > max {
			max = cat
		}
	}

	allTypes := make([]string, max+1)

	for typ, cat := range nodeTypeCategories {
		allTypes[cat] = typ.String()
	}

	return allTypes
}

// nodeCategories is a map of AST node structs to their corresponding categories. The absence of a node maps to a
// category of 0.
var nodeCategories = map[pythonast.Node]int{
	&pythonast.AnnotationStmt{}:        1,
	&pythonast.ArgsParameter{}:         2,
	&pythonast.Argument{}:              3,
	&pythonast.AssertStmt{}:            4,
	&pythonast.AssignStmt{}:            5,
	&pythonast.AttributeExpr{}:         6,
	&pythonast.AugAssignStmt{}:         7,
	&pythonast.BadExpr{}:               8,
	&pythonast.BadStmt{}:               9,
	&pythonast.BinaryExpr{}:            10,
	&pythonast.Branch{}:                11,
	&pythonast.BreakStmt{}:             12,
	&pythonast.CallExpr{}:              13,
	&pythonast.ClassDefStmt{}:          14,
	&pythonast.ComprehensionExpr{}:     15,
	&pythonast.ContinueStmt{}:          16,
	&pythonast.DelStmt{}:               17,
	&pythonast.DictComprehensionExpr{}: 18,
	&pythonast.DictExpr{}:              19,
	&pythonast.DottedAsName{}:          20,
	&pythonast.DottedExpr{}:            21,
	&pythonast.EllipsisExpr{}:          22,
	&pythonast.ExceptClause{}:          23,
	&pythonast.ExecStmt{}:              24,
	&pythonast.ExprStmt{}:              25,
	&pythonast.ForStmt{}:               26,
	&pythonast.FunctionDefStmt{}:       27,
	&pythonast.Generator{}:             28,
	&pythonast.GlobalStmt{}:            29,
	&pythonast.IfExpr{}:                30,
	&pythonast.IfStmt{}:                31,
	&pythonast.ImportAsName{}:          32,
	&pythonast.ImportFromStmt{}:        33,
	&pythonast.ImportNameStmt{}:        34,
	&pythonast.IndexExpr{}:             35,
	&pythonast.IndexSubscript{}:        36,
	&pythonast.KeyValuePair{}:          37,
	&pythonast.LambdaExpr{}:            38,
	&pythonast.ListComprehensionExpr{}: 39,
	&pythonast.ListExpr{}:              40,
	&pythonast.Module{}:                41,
	&pythonast.NameExpr{}:              42,
	&pythonast.NumberExpr{}:            43,
	&pythonast.Parameter{}:             44,
	&pythonast.PassStmt{}:              45,
	&pythonast.PrintStmt{}:             46,
	&pythonast.RaiseStmt{}:             47,
	&pythonast.ReprExpr{}:              48,
	&pythonast.ReturnStmt{}:            49,
	&pythonast.SetComprehensionExpr{}:  50,
	&pythonast.SetExpr{}:               51,
	&pythonast.SliceSubscript{}:        52,
	&pythonast.StringExpr{}:            53,
	&pythonast.TryStmt{}:               54,
	&pythonast.TupleExpr{}:             55,
	&pythonast.UnaryExpr{}:             56,
	&pythonast.WhileStmt{}:             57,
	&pythonast.WithItem{}:              58,
	&pythonast.WithStmt{}:              59,
	&pythonast.YieldExpr{}:             60,
	&pythonast.YieldStmt{}:             61,
	&pythonast.NonLocalStmt{}:          62,
	&pythonast.AwaitExpr{}:             63,
}

// nodeTypeCategories is a map from the type of each node (as determined by reflect.ValueOf().Type())
// to its corresponding category, computed from nodeCategories.
var nodeTypeCategories map[reflect.Type]int

var categoryToToken map[int]pythonscanner.Token

// numKeywords holds the number of keywords that can be predicted by the keyword model
// All ignored keywords (cat == -1) are not counted in numKeywords
var numKeywords uint

// NumKeywords is a getter for numKeywords, not exported to avoid writing it from another file
// numKeywords holds the number of keywords that can be predicted by the keyword models
// All ignored keywords (cat == -1) are not counted in numKeywords
func NumKeywords() uint {
	return numKeywords
}

func init() {
	nodeTypeCategories = make(map[reflect.Type]int, len(nodeCategories))

	for node, cat := range nodeCategories {
		nodeTypeCategories[reflect.ValueOf(node).Type()] = cat
	}
	numKeywords = 0
	categoryToToken = make(map[int]pythonscanner.Token)
	for tok, keyword := range AllKeywords {
		if keyword.Cat != -1 {
			categoryToToken[keyword.Cat] = tok
			numKeywords++
		}
	}
}
