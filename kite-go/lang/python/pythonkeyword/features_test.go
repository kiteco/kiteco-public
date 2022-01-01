package pythonkeyword

import (
	"go/token"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireCursorAndSource(t *testing.T, src string) (int64, []byte) {
	parts := strings.Split(src, "$")
	require.Len(t, parts, 2)

	return int64(len(parts[0])), []byte(strings.Join(parts, ""))
}

func requireLexParse(t *testing.T, src []byte, cursor int64) ([]pythonscanner.Word, *pythonast.Module) {
	cursorPos := token.Pos(cursor)

	words, err := pythonscanner.Lex(src, pythonscanner.Options{
		ScanComments:  true,
		ScanNewLines:  true,
		KeepEOFIndent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, words)

	mod, _ := pythonparser.ParseWords(kitectx.Background(), src, words, pythonparser.Options{
		Approximate: true,
		Cursor:      &cursorPos,
	})
	require.NotNil(t, mod)

	return words, mod
}

func assertFeatures(t *testing.T, tstSrc string, expected Features) {
	cursor, src := requireCursorAndSource(t, tstSrc)
	words, ast := requireLexParse(t, src, cursor)

	inputs := ModelInputs{
		Buffer:    src,
		Cursor:    cursor,
		AST:       ast,
		Words:     words,
		ParentMap: pythonast.ConstructParentTable(ast, pythonast.CountNodes(ast)),
	}

	features, err := NewFeatures(kitectx.Background(), inputs, ModelLookback)
	require.NoError(t, err)
	features.CodeSnippet = "" // Code Snippet if for debug purpose and just contains the initial source code
	// No need to test it

	assert.Equal(t, expected, features)
}

func prefixTokens(toks ...pythonscanner.Token) []int64 {
	prefs := make([]int64, NumKeywords())
	for _, kw := range pythonscanner.KeywordTokens {
		cat := KeywordTokenToCat(kw)
		for _, tok := range toks {
			if kw == tok {
				prefs[cat-1] = 1
			}
		}
	}
	return prefs
}

func TestFeaturesImport(t *testing.T) {
	src := `i$`
	features := Features{
		FirstToken:  pythonscanner.NewLine,
		LastSibling: 0,
		ParentNode:  NodeToCat(&pythonast.Module{}),
		RelIndent:   0,
		Previous: []pythonscanner.Token{
			pythonscanner.Illegal,
			pythonscanner.Illegal,
			pythonscanner.Illegal,
			pythonscanner.Illegal,
			pythonscanner.Illegal,
		},
		FirstChar:        buildFirstChar('i'),
		PreviousKeywords: make([]int64, NumKeywords()),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesFunctionDefWithoutPrefix(t *testing.T) {
	src := `import json

def foo(bar):
    $`
	features := Features{
		LastSibling: 0,
		ParentNode:  NodeToCat(&pythonast.FunctionDefStmt{}),
		FirstToken:  pythonscanner.NewLine,
		RelIndent:   2,
		Previous: []pythonscanner.Token{
			pythonscanner.Ident,  // foo
			pythonscanner.Lparen, // (
			pythonscanner.Ident,  // bar
			pythonscanner.Rparen, // )
			pythonscanner.Colon,  // :
		},
		FirstChar:        -1,
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.Def}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesFunctionDef(t *testing.T) {
	src := `import json

def foo(bar):
    b$`
	features := Features{
		LastSibling: 0,
		ParentNode:  NodeToCat(&pythonast.FunctionDefStmt{}),
		FirstToken:  pythonscanner.NewLine,
		RelIndent:   2,
		Previous: []pythonscanner.Token{
			pythonscanner.Ident,  // foo
			pythonscanner.Lparen, // (
			pythonscanner.Ident,  // bar
			pythonscanner.Rparen, // )
			pythonscanner.Colon,  // :
		},
		FirstChar:        buildFirstChar('b'),
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.Def}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesEOFSameIndent(t *testing.T) {
	src := `import json

def foo(bar):
    b = 5
    $`
	features := Features{
		LastSibling: NodeToCat(&pythonast.AssignStmt{}),
		ParentNode:  NodeToCat(&pythonast.FunctionDefStmt{}),
		FirstToken:  pythonscanner.NewLine,
		RelIndent:   0,
		Previous: []pythonscanner.Token{
			pythonscanner.Colon,   // :
			pythonscanner.NewLine, // \n
			pythonscanner.Ident,   // b
			pythonscanner.Assign,  // =
			pythonscanner.Int,     // 5
		},
		FirstChar:        -1,
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.Def}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesEOFLessIndent(t *testing.T) {
	src := `import json

def foo(bar):
    b = 5
$`
	features := Features{
		LastSibling: NodeToCat(&pythonast.FunctionDefStmt{}),
		ParentNode:  NodeToCat(&pythonast.Module{}),
		FirstToken:  pythonscanner.NewLine,
		RelIndent:   1,
		Previous: []pythonscanner.Token{
			pythonscanner.Colon,   // :
			pythonscanner.NewLine, // \n
			pythonscanner.Ident,   // b
			pythonscanner.Assign,  // =
			pythonscanner.Int,     // 5
		},
		FirstChar:        -1,
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.Def}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesEOFMoreIndent(t *testing.T) {
	src := `import json

def foo(bar):
    b = 5
		$`
	features := Features{
		LastSibling: 0,
		ParentNode:  NodeToCat(&pythonast.AssignStmt{}),
		FirstToken:  pythonscanner.NewLine,
		RelIndent:   2,
		Previous: []pythonscanner.Token{
			pythonscanner.Colon,   // :
			pythonscanner.NewLine, // \n
			pythonscanner.Ident,   // b
			pythonscanner.Assign,  // =
			pythonscanner.Int,     // 5
		},
		FirstChar:        -1,
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.Def}),
	}
	assertFeatures(t, src, features)
}

func buildPreviousKeywords(keywordList []pythonscanner.Token) []int64 {
	result := make([]int64, NumKeywords())
	for _, k := range keywordList {
		cat := KeywordTokenToCat(k)
		if cat > 0 {
			result[cat-1] = 1
		}
	}
	return result
}

func buildFirstChar(r rune) int64 {
	return int64(r - 'a' + 1)
}

func TestFeaturesForLoop(t *testing.T) {
	src := `import json

if __name__ == "__main__":
    foo = 1
    if json.loads(bar):
        foo = 2
    for x in ra$`
	features := Features{
		LastSibling: NodeToCat(&pythonast.IfStmt{}),
		ParentNode:  NodeToCat(&pythonast.IfStmt{}),
		FirstToken:  pythonscanner.For,
		RelIndent:   1,
		Previous: []pythonscanner.Token{
			pythonscanner.Int,
			pythonscanner.NewLine, // \n
			pythonscanner.For,     // for
			pythonscanner.Ident,   // x
			pythonscanner.In,      // in
		},
		FirstChar:        buildFirstChar('r'),
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.If, pythonscanner.For, pythonscanner.In}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesElseBranch(t *testing.T) {
	src := `import json

if __name__ == "__main__":
    foo = 1
    if json.loads(bar):
        foo = 2
    els$`
	features := Features{
		LastSibling: NodeToCat(&pythonast.IfStmt{}),
		ParentNode:  NodeToCat(&pythonast.IfStmt{}),
		FirstToken:  pythonscanner.NewLine,
		RelIndent:   1,
		Previous: []pythonscanner.Token{
			pythonscanner.Colon,   // :
			pythonscanner.NewLine, // \n
			pythonscanner.Ident,   // foo
			pythonscanner.Assign,  // =
			pythonscanner.Int,     // 2
		},
		FirstChar:        buildFirstChar('e'),
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.If}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesAfterElse(t *testing.T) {
	src := `import json

if __name__ == "__main__":
    foo = 1
    if json.loads(bar):
        foo = 2
    else:
        foo = 3
    print("B$`
	features := Features{
		LastSibling: NodeToCat(&pythonast.IfStmt{}),
		ParentNode:  NodeToCat(&pythonast.IfStmt{}),
		FirstToken:  pythonscanner.Ident,
		RelIndent:   1,
		Previous: []pythonscanner.Token{
			pythonscanner.Assign,  // ==
			pythonscanner.Int,     // 3
			pythonscanner.NewLine, // \n
			pythonscanner.Ident,   // print
			pythonscanner.Lparen,  // (
		},
		FirstChar:        0,
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import, pythonscanner.If, pythonscanner.Else}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesSimpleImportAssign(t *testing.T) {
	src := `import json

s = json$
`
	features := Features{
		LastSibling: NodeToCat(&pythonast.ImportNameStmt{}),
		ParentNode:  NodeToCat(&pythonast.Module{}),
		FirstToken:  pythonscanner.Ident,
		RelIndent:   0,
		Previous: []pythonscanner.Token{
			pythonscanner.Import,  // import
			pythonscanner.Ident,   // json
			pythonscanner.NewLine, // \n
			pythonscanner.Ident,   // s
			pythonscanner.Assign,  // =
		},
		FirstChar:        buildFirstChar('j'),
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Import}),
	}
	assertFeatures(t, src, features)
}

func TestFeaturesClassFirstLine(t *testing.T) {
	src := `class Foo:
	p$`

	// TODO(Moe): This test is an example of the issue #8037
	// The parent should be a ClassDefStmt but it currently returns the Module (as both their begins exactly match
	// the begin of the containing block
	features := Features{
		LastSibling: 0,
		ParentNode:  NodeToCat(&pythonast.Module{}),
		//  This should be: ParentNode:  NodeToCat(&pythonast.ClassDefStmt{}),
		FirstToken: pythonscanner.NewLine,
		RelIndent:  2,
		Previous: []pythonscanner.Token{
			pythonscanner.Illegal, //
			pythonscanner.Illegal, //
			pythonscanner.Class,   // Class
			pythonscanner.Ident,   // Foo
			pythonscanner.Colon,   // :
		},
		FirstChar:        buildFirstChar('p'),
		PreviousKeywords: buildPreviousKeywords([]pythonscanner.Token{pythonscanner.Class}),
	}
	assertFeatures(t, src, features)
}
