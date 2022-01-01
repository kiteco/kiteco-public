package text

import (
	"bufio"
	"bytes"
	"go/scanner"
	"go/token"
	"strings"
	"unicode"

	porterstemmer "github.com/kiteco/go-porterstemmer"
	"github.com/kiteco/kiteco/kite-golib/bufutil"
	"golang.org/x/net/html"
)

// TokenFunc defines a type of function that takes in an array of tokens and
// returns an array of tokens.
type TokenFunc func(Tokens) Tokens

// Tokens represents a slice of strings
type Tokens []string

// Processor consists of a list of text processing rules.
type Processor struct {
	filters []TokenFunc
}

// SearchTermProcessor returns a processor that does the following three to an input token array:
// 1) remove stop words
// 2) stem each token
// 3) uniquify an array of tokens
var SearchTermProcessor = NewProcessor(Lower, RemoveStopWords, Stem, Uniquify)

// TFProcessor is the processor used to build tf counts.
var TFProcessor = NewProcessor(Lower, RemoveStopWords, Stem)

// NewProcessor takes a list of TokenFuncs to instantiate a Filter.
func NewProcessor(funcs ...TokenFunc) *Processor {
	f := &Processor{}
	for _, fn := range funcs {
		f.filters = append(f.filters, fn)
	}
	return f
}

// Apply applies a list of TokenFunc to transform the input tokens
func (f *Processor) Apply(ts Tokens) Tokens {
	for _, fn := range f.filters {
		ts = fn(ts)
	}
	return ts
}

// TokenizeWithoutCamelPhrases tokenizes a string and replaces
// all the camel case phrases.
// Examples:
// "fooBar" -> {"foo", "bar"}
func TokenizeWithoutCamelPhrases(s string) Tokens {
	return tokenizeWithCamelPhrasesReplaced(s, true)
}

// Tokenize tokenizes a text string and returns the word types that appear
// in the string (i.e., no repeated words are returned).
// It tokenizes a string by whitespace. It also tokenizes all
// snake case and camel case phrases. Note that it adds all the camel phrases
// (in lower case) in the input string to the return token stream.
// To tokenize a string without the original camel phrases, call
// TokenizeWithoutCamelPhrases.
// Examples:
// "fooBar" -> {"foo", "bar", "foobar"}
func Tokenize(s string) Tokens {
	return tokenizeWithCamelPhrasesReplaced(s, false)
}

// TokenizeNoCamel is like Tokenize, but it does not tokenize any camel case
// phrases.
func TokenizeNoCamel(s string) Tokens {
	s = Normalize(s)
	buf := bytes.NewBufferString(s)
	scanner := bufio.NewScanner(buf)
	scanner.Split(bufio.ScanWords)

	var tokens Tokens
	for scanner.Scan() {
		tok := scanner.Text()
		toks := strings.Split(tok, "_")
		for _, t := range toks {
			tokens = append(tokens, t)
		}
	}

	return tokens
}

func tokenizeWithCamelPhrasesReplaced(s string, replaceCamel bool) Tokens {
	s = Normalize(s)

	buf := bytes.NewBufferString(s)
	scanner := bufio.NewScanner(buf)
	scanner.Split(bufio.ScanWords)

	var tokens Tokens
	for scanner.Scan() {
		tok := scanner.Text()
		toks := strings.Split(tok, "_")
		for _, t := range toks {
			camToks := TokenizeCamel(t)
			for _, tt := range camToks {
				tokens = append(tokens, tt)
			}
			if len(camToks) > 1 && !replaceCamel {
				tokens = append(tokens, t)
			}
		}
	}

	return tokens
}

// TokenizeCamel tokenizes camel-case phrases. For example,
// "mFooBar" -> {"m", "Foo", "Bar"}
func TokenizeCamel(w string) Tokens {
	var tokens []string
	var singleCharPhrases []byte
	l := 0
	for s := w; s != ""; s = s[l:] {
		l = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
		if l <= 0 || allCap(s) {
			l = len(s)
		}
		if l == 1 {
			singleCharPhrases = append(singleCharPhrases, s[0])
		} else {
			if len(singleCharPhrases) > 0 {
				tokens = append(tokens, string(singleCharPhrases))
			}
			tokens = append(tokens, s[:l])
			singleCharPhrases = []byte{}
		}
	}
	if len(singleCharPhrases) > 0 {
		tokens = append(tokens, string(singleCharPhrases))
	}
	return tokens
}

var specialChars = []byte(" []{}()!?@#$%^&*()_-+=,'.\\/|:;<>\"`'")

// hasSpecialChars returns whether any byte in buf matches specialChars
func hasSpecialChars(buf []byte) bool {
	for _, b := range buf {
		if bytes.Contains(specialChars, []byte{b}) {
			return true
		}
	}
	return false
}

// CodeTokensFromHTML returns the tokens that were found *within* a code snippet
func CodeTokensFromHTML(doc string) []string {
	var tokens []string
	z := html.NewTokenizer(bytes.NewBuffer([]byte(doc)))
	codeDepth := 0
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return tokens
		case html.TextToken:
			if codeDepth != 0 {
				scanner := bufio.NewScanner(bytes.NewBuffer(z.Text()))
				scanner.Split(bufio.ScanWords)
				for scanner.Scan() {
					token := scanner.Bytes()
					if len(token) > 0 {
						tokens = append(tokens, string(token))
					}
				}
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			if bytes.Equal(tn, []byte("pre")) || bytes.Equal(tn, []byte("code")) {
				if tt == html.StartTagToken {
					codeDepth++
				} else {
					codeDepth--
				}
			}
		}
	}
}

// Tokenizer is generic interface for an object which breaks an input
// string into Tokens.
type Tokenizer interface {
	Tokenize(string) Tokens
}

// HTMLTokenizer is an object for parsing the text components from
// the text components of an  HTML doc.
// Uses a bufutil.Pool to avoid using extra space for repeated tokens.
type HTMLTokenizer struct {
	pool      *bufutil.Pool
	stopWords map[string]interface{}
}

// NewHTMLTokenizer returns a HTMLTokenizer object
// No need for pointer becuase pool is a pointer anyways
func NewHTMLTokenizer() HTMLTokenizer {
	return HTMLTokenizer{
		pool:      bufutil.NewPool(),
		stopWords: StopWords(),
	}
}

// tokenizeHTML takes the provided buffer and returns a list of unigram
// tokens from the text components of the html.
// TODO: make this use strings
func tokenizeHTML(stopWords map[string]interface{}, buf []byte) [][]byte {
	var tokens [][]byte
	z := html.NewTokenizer(bytes.NewBuffer(buf))
	codeDepth := 0
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return tokens
		case html.TextToken:
			if codeDepth == 0 {
				scanner := bufio.NewScanner(bytes.NewBuffer(z.Text()))
				scanner.Split(bufio.ScanWords)
				for scanner.Scan() {
					token := scanner.Bytes()
					if len(token) > 0 {
						tokens = append(tokens, token)
					}
				}
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			if bytes.Equal(tn, []byte("pre")) || bytes.Equal(tn, []byte("code")) {
				if tt == html.StartTagToken {
					codeDepth++
				} else {
					codeDepth--
				}
			}
		}
	}
}

// Tokenize returns a list of unigram tokens taken from the text components of the html doc
func (t HTMLTokenizer) Tokenize(doc string) Tokens {
	// TODO: inefficient, lots of copying from byte arrays
	// to strings and vice versa!!
	tokenBytes := tokenizeHTML(t.stopWords, []byte(doc))
	var tokens Tokens
	for _, bytes := range tokenBytes {
		tokens = append(tokens, string(bytes))
	}
	return tokens
}

// allCap checks if the string is all cap.
func allCap(w string) bool {
	for _, c := range w {
		if !unicode.IsUpper(c) {
			return false
		}
	}
	return true
}

func cleanEnds(tok string) string {
	if len(tok) == 0 {
		return ""
	}
	if hasSpecialChars([]byte{tok[0]}) {
		if len(tok) > 1 {
			return cleanEnds(tok[1:])
		}
		return ""

	}
	if hasSpecialChars([]byte{tok[len(tok)-1]}) {
		if len(tok) > 1 {
			return cleanEnds(tok[:len(tok)-1])
		}
		return ""
	}
	return tok
}

// RemoveSpecialCharacterTokens removes tokens that
// have special characters that are not at the prefix or postfix of the token.
func RemoveSpecialCharacterTokens(ts Tokens) Tokens {
	var clean Tokens
	for _, t := range ts {
		t = cleanEnds(t)
		if len(t) > 0 && !hasSpecialChars([]byte(t)) {
			clean = append(clean, t)
		}
	}
	return clean
}

// CleanTokens emits a stream of tokens that
// have special characters located in the prfixes of postfixes of tokens removed.
// Note there still may be tokens emitted that have special characters in the middle
// of the token.
func CleanTokens(ts Tokens) Tokens {
	var clean Tokens
	for _, t := range ts {
		t = cleanEnds(t)
		if len(t) > 0 {
			clean = append(clean, t)
		}
	}
	return clean
}

// RemoveStopWords removes stop words from a TokenStream
func RemoveStopWords(ts Tokens) Tokens {
	var filteredTokens Tokens
	for _, t := range ts {
		if !skip(t) {
			filteredTokens = append(filteredTokens, t)
		}
	}
	return filteredTokens
}

// RemoveStopWordsExt removes stop words from a TokenStream
func RemoveStopWordsExt(ts Tokens) Tokens {
	var filteredTokens Tokens
	for _, t := range ts {
		if !skipExt(t) {
			filteredTokens = append(filteredTokens, t)
		}
	}
	return filteredTokens
}

// Lower converts all tokens to lower case
func Lower(ts Tokens) Tokens {
	for i, t := range ts {
		ts[i] = strings.ToLower(t)
	}
	return ts
}

// Stem extracts and returns the stems of each token in the input token stream
func Stem(ts Tokens) Tokens {
	for i, t := range ts {
		ts[i] = porterstemmer.StemString(t)
	}
	return ts
}

// Uniquify returns the set of unique tokens in a token stream
func Uniquify(ts Tokens) Tokens {
	var uniqueTokens Tokens
	seen := make(map[string]struct{})
	for _, t := range ts {
		if _, exists := seen[t]; !exists {
			uniqueTokens = append(uniqueTokens, t)
			seen[t] = struct{}{}
		}
	}
	return uniqueTokens
}

// TokenizeCodeWithoutCamelPhrases is the same as TokenizeCode except
// that it doesn't expand the return token stream with the camel phreases
// in the original code snippet.
func TokenizeCodeWithoutCamelPhrases(code string) Tokens {
	var tokens Tokens
	for _, tok := range lexCode(code) {
		tokens = append(tokens, TokenizeWithoutCamelPhrases(tok)...)
	}
	return tokens
}

// TokenizeCodeNoCamel is like TokenizeCode except that it doesn't
// tokenize any camel-case phrases in the input string.
func TokenizeCodeNoCamel(code string) Tokens {
	var tokens Tokens
	for _, tok := range lexCode(code) {
		tokens = append(tokens, TokenizeNoCamel(tok)...)
	}
	return tokens
}

// TokenizeCode tokens a piece of code.
func TokenizeCode(code string) Tokens {
	var tokens Tokens
	for _, tok := range lexCode(code) {
		tokens = append(tokens, Tokenize(tok)...)
	}
	return tokens
}

// CodeTokenizer tokenizes code using the Go lexer.
type CodeTokenizer struct{}

// Tokenize returns a stream of code tokens.
func (c CodeTokenizer) Tokenize(code string) Tokens {
	buf := []byte(code)
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(buf))
	var lexer scanner.Scanner
	lexer.Init(file, buf, nil, 0)
	var words []string
	for {
		_, t, lit := lexer.Scan()
		if t == token.EOF {
			break
		}
		if strings.TrimSpace(lit) == "" {
			continue
		}
		if strings.Trim(lit, string(specialChars)) == "" {
			continue
		}
		words = append(words, lit)
	}
	return words
}

// SpaceTokenizer tokenizers code based on whitespace.
type SpaceTokenizer struct{}

// Tokenize satisfies the Tokenizer interface.
// TODO(juan): optimize
func (st SpaceTokenizer) Tokenize(doc string) Tokens {
	var tokens Tokens
	for _, tok := range strings.Split(strings.TrimSpace(doc), " ") {
		tok = strings.TrimSpace(tok)
		if len(tok) > 0 {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}

func lexCode(code string) Tokens {
	buf := []byte(code)
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(buf))
	var lexer scanner.Scanner
	lexer.Init(file, buf, nil, 0)
	var words []string
	for {
		_, t, lit := lexer.Scan()
		if t == token.EOF {
			break
		}
		words = append(words, tokenToWord(t, lit))
	}
	return words
}

func tokenToWord(t token.Token, lit string) string {
	switch t {
	case token.COMMENT, token.IDENT:
		return lit
	case token.STRING, token.CHAR:
		return "STR"
	case token.LBRACE:
		return "dictionary"
	default:
		return ""
	}
}

var stopWords = StopWords()

// skipExt determines whether a word should be removed (or skipped).
func skipExt(w string) bool {
	_, skip := stopWords[w]
	return skip
}

// skip determines whether a word should be removed (or skipped).
func skip(w string) bool {
	switch w {
	case "a", "an", "the", "in", "of", "with", "to", "how", "what", "from", "and", "can", "is", "here", "there", "hello", "world", "try":
		return true
	}
	return false
}
