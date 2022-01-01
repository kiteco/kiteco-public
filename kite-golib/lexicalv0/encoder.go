package lexicalv0

import (
	"fmt"
	"go/token"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/golang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

// LangTags ...
func LangTags() map[lang.Language]lexer.Token {
	var langs []lang.Language
	for l := range lang.LanguageTags {
		langs = append(langs, l)
	}

	sort.Slice(langs, func(i, j int) bool {
		return langs[i] < langs[j]
	})

	// sep token is -2
	tok := -3

	tags := make(map[lang.Language]lexer.Token)
	for _, l := range langs {
		tags[l] = lexer.Token{Token: tok, Lit: fmt.Sprintf("kite-langtag184-%s", l.Name())}
		tok--
	}
	return tags
}

// ExtraTokens ...
func ExtraTokens(g LangGroup) []lexer.Token {
	// SOF is not included here since we already include the
	// +1 from the SOF in FileEncoder.NumLexical()
	toks := []lexer.Token{
		{Token: lexer.SepTok, Lit: lexer.SepTokStr},
	}

	// second clause is for single language text models
	if g.Lexer == lang.Text && g.IsMultiLingual() {
		tags := LangTags()
		for _, ll := range g.Langs {
			toks = append(toks, tags[ll])
		}
	}
	return toks
}

// FileEncoder ...
type FileEncoder struct {
	BPE             *bpe.Encoder
	Lexer           lexer.Lexer
	IDToString      []string
	IDToStringLower []string
	IDToToken       map[int]int
	StringToID      map[string]int

	tags      map[lang.Language]lexer.Token
	langGroup LangGroup
}

// NewFileEncoderFromVocab ...
func NewFileEncoderFromVocab(vocab []bpe.Entry, g LangGroup) (*FileEncoder, error) {
	enc := bpe.NewEncoderFromVocab(vocab)
	return newFileEncoderFromBPE(enc, g)
}

// NewFileEncoder ...
func NewFileEncoder(vocab string, g LangGroup) (*FileEncoder, error) {
	enc, err := bpe.NewEncoder(vocab)
	if err != nil {
		return nil, err
	}
	return newFileEncoderFromBPE(enc, g)
}

func newFileEncoderFromBPE(bpe *bpe.Encoder, g LangGroup) (*FileEncoder, error) {
	langLexer, err := NewLexer(g.Lexer)
	if err != nil {
		return nil, err
	}

	itt := make(map[int]int)
	lookup := []string{"SOF"}
	for _, tok := range langLexer.Tokens() {
		itt[len(lookup)] = tok.Token
		lookup = append(lookup, tok.Lit)
	}

	for _, v := range bpe.Vocab() {
		if g.Lexer == lang.Golang {
			itt[len(lookup)] = int(token.IDENT)
		} else {
			itt[len(lookup)] = lexer.BPEEncodedTok
		}
		lookup = append(lookup, v)
	}

	for _, t := range ExtraTokens(g) {
		if t.Token == 0 || t.Token == lexer.SepTok {
			// backwards compat since we already include sof above and sep below
			continue
		}
		itt[len(lookup)] = int(t.Token)
		lookup = append(lookup, t.Lit)
	}

	// sep token is always the last token in the vocab
	itt[len(lookup)] = lexer.SepTok
	lookup = append(lookup, lexer.SepTokStr)

	sti := make(map[string]int)
	for i, s := range lookup {
		sti[s] = i
	}

	lookupLower := make([]string, 0, len(lookup))
	for _, l := range lookup {
		if l != lexer.SepTokStr { // works because sep is at the end
			lookupLower = append(lookupLower, strings.ToLower(l))
		}
	}

	return &FileEncoder{
		BPE:             bpe,
		Lexer:           langLexer,
		IDToString:      lookup,
		IDToStringLower: lookupLower,
		IDToToken:       itt,
		StringToID:      sti,
		tags:            LangTags(),
		langGroup:       g,
	}, nil
}

// SepVocabID ...
func (f *FileEncoder) SepVocabID() int {
	return len(f.IDToString) - 1
}

// Encode ...
func (f *FileEncoder) Encode(buf []byte, filename string) ([]string, error) {
	tokens, err := f.Lexer.Lex(buf)
	if err != nil {
		return nil, err
	}

	var encoded []string
	for _, t := range f.BeforeContextPrefix(filename) {
		encoded = append(encoded, f.IDToString[t])
	}
	for _, tok := range tokens {
		if subtokens, ok := f.Lexer.ShouldBPEEncode(tok); ok {
			encoded = append(encoded, f.BPE.Encode(subtokens)...)
		} else {
			encoded = append(encoded, f.Lexer.TokenName(tok.Token))
		}
	}

	return encoded, nil
}

// IsLexical returns true if the id represents a lexical token
func (f *FileEncoder) IsLexical(id int) bool {
	return id < f.Lexer.NumTokens()+1 // +1 is for the SOF token
}

// IsEncoderToken returns true if the id represents a special encoder token
func (f *FileEncoder) IsEncoderToken(id int) bool {
	if id == f.SepVocabID() {
		return true
	}
	for _, tag := range f.tags {
		if f.StringToID[tag.Lit] == id {
			return true
		}
	}
	return false
}

// NumLexical returns the number of lexical tokens (including SOF)
func (f *FileEncoder) NumLexical() int {
	return f.Lexer.NumTokens() + 1 // +1 is for the SOF token
}

// PrepareBeforeContext for prediction with the specific window size
// TODO: PrepareLeftContext ?
func (f *FileEncoder) PrepareBeforeContext(context []int, window int, filename string) []int {
	prefix := f.BeforeContextPrefix(filename)

	if window <= len(prefix) {
		// TODO: not clear what to do here
		return prefix[:window]
	}

	var hasPrefix bool
	if len(context) >= len(prefix) {
		hasPrefix = true
		for i, p := range prefix {
			if context[i] != p {
				hasPrefix = false
				break
			}
		}
	}

	if hasPrefix {
		prefix = nil
	}

	if len(context)+len(prefix) > window {
		context = context[len(context)+len(prefix)-window:]
	}

	final := make([]int, 0, len(prefix)+len(context))
	final = append(final, prefix...)
	final = append(final, context...)
	return final
}

// RemoveBeforeContextPrefix removes the prefix that is added to the context
// for prediction.
func (f *FileEncoder) RemoveBeforeContextPrefix(context []int, filename string) []int {
	prefix := f.BeforeContextPrefix(filename)
	if len(context) >= len(prefix) {
		for i, p := range prefix {
			if context[i] != p {
				return context
			}
		}
		return context[len(prefix):]
	}
	return context
}

// BeforeContextPrefix contains the prefix vocab elements that should be
// prepended to every "before" context that is used for prediction.
func (f *FileEncoder) BeforeContextPrefix(filename string) []int {
	// second clause is for single language text models
	if f.langGroup.Lexer != lang.Text || !f.langGroup.IsMultiLingual() {
		return []int{0}
	}

	nativeLang := lang.FromFilename(filename)
	tag, ok := f.tags[nativeLang]
	if !ok {
		// TODO: not clear what to do here
		return []int{0}
	}

	id := f.StringToID[tag.Lit]
	return []int{id, 0}
}

// EncodeIdx ...
func (f *FileEncoder) EncodeIdx(buf []byte, filename string) ([]int, error) {
	tokens, err := f.Lexer.Lex(buf)
	if err != nil {
		return nil, err
	}
	encoded := f.BeforeContextPrefix(filename)
	return append(encoded, f.EncodeTokens(tokens)...), nil
}

// EncodeTokens ...
func (f *FileEncoder) EncodeTokens(tokens []lexer.Token) []int {
	// SOF is indexed as 0
	var encoded []int
	for _, tok := range tokens {
		if subtokens, ok := f.Lexer.ShouldBPEEncode(tok); ok {
			encoded = append(encoded, f.EncodeSubtokens(subtokens)...)
		} else {
			// Nasty hack for backwards compatibility. We should get rid of this mapping
			// next time we retrain
			if f.Lexer.Lang() == lang.Golang {
				encoded = append(encoded, golang.TokenToIdx[token.Token(tok.Token)]+1) // +1 is for the SOF token
			} else {
				encoded = append(encoded, tok.Token+1) // +1 is for the SOF token
			}
		}
	}

	return encoded
}

// EncodeSubtokens ...
func (f *FileEncoder) EncodeSubtokens(subtokens []string) []int {
	return shiftByOffset(
		f.Lexer.NumTokens()+1, // +1 is for the SOF token
		f.BPE.EncodeIdx(subtokens))
}

// Size of the dictionary
func (f *FileEncoder) Size() int {
	return len(f.IDToString)
}

// DecodeToVocab gets vocab entries from []int
func (f *FileEncoder) DecodeToVocab(toks []int) []string {
	var entries []string
	for _, tok := range toks {
		if tok == 0 {
			// Handle SOF here otherwise IsLexical will not work
			entries = append(entries, "SOF")
			continue
		}
		if f.IsLexical(tok) {
			lexicalToken := tok - 1 // NOTE: -1 here to shift out SOF token

			// NOTE: This golang-specific branch mirrors the golang-specific portion
			// of EncodeTokens - to remove once we retrain
			if f.langGroup.Lexer == lang.Golang {
				lexicalToken = int(golang.IdxToToken[lexicalToken])
			}
			entries = append(entries, f.Lexer.TokenName(lexicalToken))
		} else {
			entries = append(entries, f.IDToString[tok])
		}
	}
	return entries
}

// DecodeToStrings ...
func (f *FileEncoder) DecodeToStrings(toks []int) []string {
	var ret []string
	tokens := f.Decode(toks)
	for _, tok := range tokens {
		ret = append(ret, tok.Lit)
	}
	return ret
}

// Decode ...
func (f *FileEncoder) Decode(toks []int) []lexer.Token {
	var bpes []string
	var tokens []lexer.Token

	handleBPE := func() {
		if len(bpes) == 0 {
			return
		}
		for _, s := range f.Lexer.MergeBPEEncoded(bpes) {
			tokens = append(tokens, lexer.Token{
				Token: lexer.BPEEncodedTok,
				Lit:   s,
			})
		}
		bpes = nil
	}

	for _, tok := range toks {
		// Handle SOF here otherwise IsLexical will not work
		if tok == 0 {
			continue
		}
		if f.IsLexical(tok) {
			handleBPE()
			lexicalToken := tok - 1 // NOTE: -1 here to shift out SOF token

			// NOTE: This golang-specific branch mirrors the golang-specific portion
			// of EncodeTokens - to remove once we retrain
			if f.langGroup.Lexer == lang.Golang {
				lexicalToken = int(golang.IdxToToken[lexicalToken])
			}

			tokens = append(tokens, lexer.Token{
				Token: lexicalToken,
				Lit:   f.Lexer.TokenName(lexicalToken),
			})
		} else {
			bpes = append(bpes, f.IDToString[tok])
		}
	}

	// In case we're ending in a BPE encoded token
	handleBPE()

	return tokens
}

// LangTagForPath ...
func (f *FileEncoder) LangTagForPath(path string) int {
	// second clause is for single language text models
	// TODO: third clause is nasty, we should clean this up, the large all lang model
	// doesn't use the lang tags
	if f.langGroup.Lexer != lang.Text || !f.langGroup.IsMultiLingual() || f.langGroup.Equals(AllLangsGroup) {
		return -1
	}
	return f.langGroup.LangTagFor(path)
}

func shiftByOffset(offset int, idx []int) []int {
	var shifted []int
	for _, val := range idx {
		shifted = append(shifted, val+offset)
	}
	return shifted
}
