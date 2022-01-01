package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeCamel(t *testing.T) {
	text := "JSONEncoder"
	tokens := TokenizeCamel(text)
	require.Len(t, tokens, 2)

	assert.Equal(t, "JSON", tokens[0])
	assert.Equal(t, "Encoder", tokens[1])

	text = "1D"
	tokens = TokenizeCamel(text)
	require.Len(t, tokens, 1)
	assert.Equal(t, "1D", tokens[0])

	text = "mFooBar"
	tokens = TokenizeCamel(text)
	require.Len(t, tokens, 3)
	assert.Equal(t, "m", tokens[0])
	assert.Equal(t, "Foo", tokens[1])
	assert.Equal(t, "Bar", tokens[2])
}

func TestTokenizeCode(t *testing.T) {
	code := `
		def main():
		    # this is a test
		    a, b = "1323", "test"
		    tempFile,
		    TESTVAR_summer = {}
		    return TESTVAR`

	tokens := TokenizeCode(code)

	require.Len(t, tokens, 17)

	assert.Equal(t, "def", tokens[0])
	assert.Equal(t, "summer", tokens[len(tokens)-3])
}

func TestTokenizeCodeWithoutCamelPhrases(t *testing.T) {
	code := `
		def main():
		    # this is a test
		    a, b = "1323", "test"
		    tempFile,
		    TESTVAR_summer = {}
		    return TESTVAR`

	tokens := TokenizeCodeWithoutCamelPhrases(code)

	require.Len(t, tokens, 16)

	assert.Equal(t, "def", tokens[0])
	assert.Equal(t, "summer", tokens[len(tokens)-3])
}

func TestStem(t *testing.T) {
	test := []string{"lane", "parsing", "parse", "cookies", "beautiful", "Creating", "constructing", "setting"}
	test = Stem(test)
	exp := []string{"lane", "pars", "pars", "cooki", "beauti", "creat", "construct", "set"}
	assert.Equal(t, exp, test)
}

func TestSearchTermProcessor(t *testing.T) {
	test := []string{"parsing", "parse", "cookies", "beautiful", "Creating", "constructing", "setting", "construct", "a"}
	filter := SearchTermProcessor
	act := filter.Apply(test)

	exp := Tokens([]string{"pars", "cooki", "beauti", "creat", "construct", "set"})
	assert.Equal(t, exp, act)
}

func TestCleanTokens(t *testing.T) {
	test := []string{"<go>", "<python>", "path))", ","}
	test = CleanTokens(test)
	exp := []string{"go", "python", "path"}
	assert.Equal(t, exp, test)
}

func TestRemoveSpecialCharacterTokens(t *testing.T) {
	test := []string{"google_id=89", ",go", "<python>>"}
	test = RemoveSpecialCharacterTokens(test)
	exp := []string{"go", "python"}
	assert.Equal(t, exp, test)
}

func TestLowerCase(t *testing.T) {
	test := []string{"GO", "THERE"}
	test = Lower(test)
	exp := []string{"go", "there"}
	assert.Equal(t, exp, test)
}
func TestCodeTokenizer(t *testing.T) {
	code := `(void)parser:(PaymentTermsLibxmlParser *)parser encounteredError:(NSError *)error
			{
    			NSLog("error occured");
			}`

	tokenizer := CodeTokenizer{}
	tokens := tokenizer.Tokenize(code)

	require.Len(t, tokens, 9)

	assert.Equal(t, "void", tokens[0])
	assert.Equal(t, "\"error occured\"", tokens[len(tokens)-1])
}

func TestSpaceTokenizer(t *testing.T) {
	doc := "this  is a string with spaces   "
	tokenizer := SpaceTokenizer{}
	test := tokenizer.Tokenize(doc)
	exp := Tokens{"this", "is", "a", "string", "with", "spaces"}
	assert.Equal(t, exp, test)
}
