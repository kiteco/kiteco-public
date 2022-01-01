package githubdata

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

// PredictionSite ...
type PredictionSite struct {
	PullNumber int    `json:"pull"`
	PullTime   string `json:"time"`
	FilePath   string `json:"filepath"`

	DstContextBefore string `json:"dst_context_before"`
	DstContextAfter  string `json:"dst_context_after"`
	DstWindow        string `json:"dst_window"`

	SrcContextBefore string `json:"src_context_before"`
	SrcContextAfter  string `json:"src_context_after"`
	SrcWindow        string `json:"src_window"`
}

// PredictionSiteWithMetrics ...
// TODO: rename this to prediction site
type PredictionSiteWithMetrics struct {
	PredictionSite

	AdditionSize int `json:"addition"`
	// Some metrics calculated for the purpose of filtering/grouping
	// Levenshtein distance / average length treating each token a character
	RelativeLevenshtein float64 `json:"levenshtein"`

	NumTokensBefore int `json:"tokens_before"`
	NumTokensAfter  int `json:"tokens_after"`

	NumIdentsBefore int `json:"idents_before"`
	NumIdentsAfter  int `json:"idents_after"`

	NumTokensInsertion int `json:"tokens_insertion"`
	NumTokensDeletion  int `json:"tokens_deletion"`

	NumIdentsInsertion int `json:"idents_insertion"`
	NumIdentsDeletion  int `json:"idents_deletion"`
}

// NewPredictionSiteWithMetrics ...
func NewPredictionSiteWithMetrics(p PredictionSite, langLexer lexer.Lexer) (PredictionSiteWithMetrics, error) {
	sm, leven, err := extractSiteMetric(p.SrcWindow, p.DstWindow, langLexer)
	if err != nil {
		return PredictionSiteWithMetrics{}, err
	}

	return PredictionSiteWithMetrics{
		PredictionSite:      p,
		RelativeLevenshtein: leven,
		NumTokensBefore:     sm.tokensB,
		NumTokensAfter:      sm.tokensA,
		NumIdentsBefore:     sm.identsB,
		NumIdentsAfter:      sm.identsA,
		NumTokensInsertion:  sm.tokensI,
		NumIdentsDeletion:   sm.tokensD,
		AdditionSize:        sm.tokensA,
	}, nil
}

type siteMetric struct {
	tokensB int
	tokensA int
	identsB int
	identsA int

	tokensD int
	tokensI int
	identsD int
	identsI int
}

func extractSiteMetric(beforeSite, afterSite string, langLexer lexer.Lexer) (siteMetric, float64, error) {
	wordsB, numTokensB, numIdentsB, err := toWords(beforeSite, langLexer)
	if err != nil {
		return siteMetric{}, 0, errors.Wrapf(err, "unable to convert before to words")
	}

	wordsA, numTokensA, numIdentsA, err := toWords(afterSite, langLexer)
	if err != nil {
		return siteMetric{}, 0, errors.Wrapf(err, "unable to convert after to words")
	}

	sm := siteMetric{
		tokensB: numTokensB,
		tokensA: numTokensA,
		identsB: numIdentsB,
		identsA: numIdentsA,
	}

	dmp := diffmatchpatch.New()
	charsB, charsA, lineArray := dmp.DiffLinesToChars(wordsB, wordsA)
	diffs := dmp.DiffMain(charsB, charsA, true)

	levenshtein := dmp.DiffLevenshtein(diffs)
	var relativeLeven float64
	if len(wordsA)+len(wordsB) > 0 {
		// TODO: what should we do here?
		relativeLeven = 2 * float64(levenshtein) / float64(len(wordsB)+len(wordsA))
	}

	reverted := dmp.DiffCharsToLines(diffs, lineArray)
	for _, diff := range reverted {
		// Put words back to line
		formatted := strings.ReplaceAll(diff.Text, "\n", " ")
		// Re-create new lines
		formatted = strings.ReplaceAll(formatted, ";;", "\n")

		numTokens, numIdents, err := lexAndCount(langLexer, formatted)
		if err != nil {
			return siteMetric{}, 0, errors.Wrapf(err, "unable to lex and count")
		}

		if diff.Type == diffmatchpatch.DiffInsert {
			sm.tokensI += numTokens
			sm.identsI += numIdents
		}
		if diff.Type == diffmatchpatch.DiffDelete {
			sm.tokensD += numTokens
			sm.identsD += numIdents
		}
	}
	return sm, relativeLeven, nil
}

func lexAndCount(l lexer.Lexer, s string) (int, int, error) {
	tokens, err := l.Lex([]byte(s))
	if err != nil {
		return 0, 0, errors.Wrapf(err, "unable to lex")
	}

	tokens = trimRightTokens(tokens, l)

	var numIdents int
	for _, t := range tokens {
		if l.IsType(lexer.IDENT, t) {
			numIdents++
		}
	}
	return len(tokens), numIdents, nil
}

func toWords(raw string, langLexer lexer.Lexer) (string, int, int, error) {
	tokens, err := langLexer.Lex([]byte(raw))
	if err != nil {
		return "", 0, 0, errors.Wrapf(err, "unable to lex")
	}

	tokens = trimRightTokens(tokens, langLexer)

	var words []string
	var numIdents int
	for _, t := range tokens {
		if langLexer.IsType(lexer.SEMICOLON, t) && t.Lit != ";" {
			words = append(words, ";;")
			continue
		}
		if langLexer.IsType(lexer.COMMENT, t) {
			words = append(words, t.Lit+";;")
			continue
		}
		if langLexer.IsType(lexer.IDENT, t) {
			numIdents++
		}
		words = append(words, getString(langLexer, t))
	}
	return strings.Join(words, "\n"), len(words), numIdents, nil
}

func getString(l lexer.Lexer, tok lexer.Token) string {
	if _, ok := l.ShouldBPEEncode(tok); ok || tok.Lit != "" {
		return tok.Lit
	}
	return l.TokenName(tok.Token)
}

func trimRightTokens(tokens []lexer.Token, langLexer lexer.Lexer) []lexer.Token {
	var count int
	for len(tokens) > 0 {
		count++
		if count > 2 {
			break
		}
		last := tokens[len(tokens)-1]
		if langLexer.IsType(lexer.EOF, last) || langLexer.IsType(lexer.SEMICOLON, last) {
			tokens = tokens[:len(tokens)-1]
			continue
		}
		break
	}
	return tokens
}
