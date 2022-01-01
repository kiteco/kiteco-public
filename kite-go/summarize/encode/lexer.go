package encode

import "github.com/kiteco/kiteco/kite-golib/lexicalv0/text"

// Lex ...
func Lex(buf string) []string {
	return text.SplitWithOpts(buf, true)
}
