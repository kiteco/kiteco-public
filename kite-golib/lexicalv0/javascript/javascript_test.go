package javascript

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireLex(t *testing.T, src string) ([]string, []int) {
	lexer, err := NewLexer()
	require.NoError(t, err)

	toks, err := lexer.Lex([]byte(src))
	require.NoError(t, err)

	var ids []int
	var lits []string
	for _, tok := range toks {
		lits = append(lits, tok.Lit)
		ids = append(ids, tok.Token)
	}

	return lits, ids
}

func Test_JavascriptAutoSemicolon(t *testing.T) {
	src := `
export const LOAD_USER = 'load account/user data'
export const getUser
`

	lits, ids := requireLex(t, src)

	require.Equal(t,
		[]string{"export", "const", "LOAD_USER", "=", "'", "load account/user data", "'", "", "export", "const", "getUser", ""},
		lits,
	)

	require.Equal(
		t,
		[]int{3, 14, 1, 39, 96, 97, 96, 117, 3, 14, 1, 117},
		ids,
	)
}

func Test_JavascriptOnlyRealSemicolons(t *testing.T) {
	// make sure that only semi colons put by the user get mapped to anonSymSemi
	src := `
export const LOAD_USER = 'load account/user data';
if (someBool)
`

	lits, ids := requireLex(t, src)

	require.Equal(t,
		[]string{"export", "const", "LOAD_USER", "=", "'", "load account/user data", "'", ";", "if", "(", "someBool", ")", ""},
		lits,
	)

	require.Equal(
		t,
		[]int{3, 14, 1, 39, 96, 97, 96, 33, 15, 18, 1, 19, 117},
		ids,
	)
}

func Test_JavascriptInVueTokens(t *testing.T) {
	src := `
<template>
  <div class="split"></div>
</template>

<script>
export default {
  name: 'split'
}
</script>

<style lang="stylus" rel="stylesheet/stylus">

.split
  width: 100%
  height: 16px
  border-top: 1px solid rgba(7, 17, 27, .1)
  border-bottom: 1px solid rgba(7, 17, 27, .1)
  background: #f3f5f7

</style>`

	lits, ids := requireLex(t, src)

	require.Equal(t,
		[]string{"<", "template", ">", "\n  ", "<", "div", "class", "=", "\"", "split", "\"", ">", "<", "/", "div", ">", "\n", "<", "/", "template", ">", "<", "script", ">", "export", "default", "{", "name", ":", "'", "split", "'", "}", "<", "/", "script>", "<", "style", "lang", "=", "\"", "stylus", "\"", "rel", "=", "\"", "stylesheet/stylus", "\"", ">", "\n\n.split\n  width: 100%\n  height: 16px\n  border-top: 1px solid rgba(7, 17, 27, .1)\n  border-bottom: 1px solid rgba(7, 17, 27, .1)\n  background: #f3f5f7\n\n", "<", "/", "style", ">", ""},
		lits)

	require.Equal(t,
		[]int{42, 1, 43, 45, 42, 1, 227, 39, 94, 95, 94, 43, 42, 44, 1, 43, 45, 42, 44, 1, 43, 42, 1, 43, 1, 5, 6, 227, 34, 96, 97, 96, 8, 42, 44, 103, 42, 1, 227, 39, 94, 95, 94, 227, 39, 94, 95, 94, 43, 45, 42, 44, 1, 43, 117},
		ids)
}

func Test_JavascriptUnclosedString(t *testing.T) {
	// Issue #10491
	src := `var greeting = 'hello`
	lits, ids := requireLex(t, src)
	require.Equal(t, []string{"var", "greeting", "=", "'", "hello"}, lits)
	// Note: because of the remapping of global treesitter symbols in
	// ../lexer/treesitter.go, the ERROR token 65535 generated for "hello"
	// is converted to the max symbol id of javascript + 1, so 231.
	require.Equal(t, []int{anonSymVar, symIdentifier, anonSymEq, anonSymSquote, symIdentifier}, ids)
}

func Test_JavascriptShouldBPEEncode(t *testing.T) {
	lexer, err := NewLexer()
	require.NoError(t, err)

	type testCase struct {
		lit    string
		should bool
		parts  []string
	}

	identTestCases := []testCase{
		{"fooBar", true, []string{"fooBar$"}},
		{"foo_bar", true, []string{"foo_bar$"}},
		{"foo-bar", true, []string{"foo-bar$"}},
		{"foo_bar-Baz", true, []string{"foo_bar-Baz$"}},
		{"./foobar", false, nil},
		{"$", true, []string{"$$"}},
	}

	for _, tc := range identTestCases {
		parts, ok := lexer.ShouldBPEEncode(makeJSIdentToken(t, tc.lit))
		assert.Equal(t, tc.should, ok)
		assert.Equal(t, tc.parts, parts)
	}

	stringTestCases := []testCase{
		{"fooBar", true, []string{"fooBar$"}},
		{"foo_bar", true, []string{"foo_bar$"}},
		{"foo-bar", true, []string{"foo-bar$"}},
		{"foo_bar-Baz", true, []string{"foo_bar-Baz$"}},
		{".foo/bar", true, []string{".foo+", "/+", "bar$"}},
		{".foobar", true, []string{".foobar$"}},
		{"#foobar", true, []string{"#foobar$"}},
		{"#foo bar", true, []string{"#foo+", " +", "bar$"}},
		{"#foobar\n", true, []string{"#foobar+", "\n$"}},
		{"$", false, nil},
	}

	for _, tc := range stringTestCases {
		parts, ok := lexer.ShouldBPEEncode(makeJSStringToken(t, tc.lit))
		assert.Equal(t, tc.should, ok)
		assert.Equal(t, tc.parts, parts)
	}
}

func TestEmptyToken(t *testing.T) {
	src := []byte(`
import React from "react";

import AnswersPage from "@kiteco/kite-answers-renderer";

class AnswersContainer extends React.Component {
  render() {
    if (this.props.input && this.props.input.content)
`)
	lexer, err := NewLexer()
	require.NoError(t, err)

	toks, err := lexer.Lex(src)
	require.NoError(t, err)

	var ids []int
	var lits []string
	for _, tok := range toks {
		lits = append(lits, tok.Lit)
		ids = append(ids, tok.Token)
	}

	require.Equal(t, []string{
		"import", "React", "from", "\"", "react", "\"", ";", "import", "AnswersPage", "from", "\"", "@kiteco/kite-answers-renderer", "\"", ";", "class", "AnswersContainer", "extends", "React", ".", "Component", "{", "render", "(", ")", "{", "if", "(", "this", ".", "props", ".", "input", "&&", "this", ".", "props", ".", "input", ".", "content", ")", "",
	}, lits)

	require.Equal(t, []int{
		10, 1, 11, 94, 95, 94, 33, 10, 1, 11, 94, 95, 94, 33, 48, 1, 49, 1, 47, 227, 6, 227, 18, 19, 6, 15, 18, 107, 47, 227, 47, 227, 68, 107, 47, 227, 47, 227, 47, 227, 19, 117,
	}, ids)
}

func makeJSIdentToken(t *testing.T, lit string) lexer.Token {
	for tok := range jsIdentLike {
		return lexer.Token{
			Lit:   lit,
			Token: tok,
		}
	}
	require.FailNow(t, "jsIdentLike is empty")
	return lexer.Token{}
}

func makeJSStringToken(t *testing.T, lit string) lexer.Token {
	for tok := range jsStringBPE {
		return lexer.Token{
			Lit:   lit,
			Token: tok,
		}
	}
	require.FailNow(t, "jsStringBPE is empty")
	return lexer.Token{}
}
