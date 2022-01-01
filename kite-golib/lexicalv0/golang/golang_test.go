package golang

import (
	"go/token"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/stretchr/testify/assert"
)

func Test_GolangShouldBPEEncode(t *testing.T) {
	lexer := Lexer{}

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
	}

	for _, tc := range identTestCases {
		parts, ok := lexer.ShouldBPEEncode(makeGoIdentToken(t, tc.lit))
		assert.Equal(t, tc.should, ok)
		assert.Equal(t, tc.parts, parts)
	}

	stringTestCases := []testCase{
		{"fooBar", false, nil},
		{"foo_bar", false, nil},
	}

	for _, tc := range stringTestCases {
		parts, ok := lexer.ShouldBPEEncode(makeGoStringToken(t, tc.lit))
		assert.Equal(t, tc.should, ok)
		assert.Equal(t, tc.parts, parts)
	}
}

func makeGoIdentToken(t *testing.T, lit string) lexer.Token {
	return lexer.Token{
		Lit:   lit,
		Token: int(token.IDENT),
	}
}

func makeGoStringToken(t *testing.T, lit string) lexer.Token {
	return lexer.Token{
		Lit:   lit,
		Token: int(token.STRING),
	}
}
