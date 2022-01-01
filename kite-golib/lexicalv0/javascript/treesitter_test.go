package javascript

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TreeSitterRemapping(t *testing.T) {
	lexer, err := NewLexer()
	require.NoError(t, err)

	file := `
// we expect the initial # to be illegal
#target photoshop
scaleFactor=0.6667
app.bringToFront();
`

	tokens, err := lexer.Lex([]byte(file))
	require.NoError(t, err)

	secondToken := tokens[1]
	require.Equal(t, "KITE_ILLEGAL", lexer.TokenName(secondToken.Token))
	require.True(t, secondToken.Token < lexer.NumTokens())
}
