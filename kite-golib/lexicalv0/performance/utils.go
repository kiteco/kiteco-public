package performance

import (
	"math/rand"
	"reflect"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
)

var errDepthExceedsSlots = errors.New("depth must not exceed numSlots")

func matchSeries(ss []predict.Predicted, label []int, topN int) Metric {
	if len(ss) == 0 {
		return Metric{
			Accurate: 0.0,
			InTopK:   0.0,
		}
	}

	var accurate float64
	if reflect.DeepEqual(ss[0].TokenIDs, label) {
		accurate = 1.0
		return Metric{
			Accurate: accurate,
			InTopK:   accurate,
		}
	}

	var inTopK float64
	if len(ss) < topN {
		topN = len(ss)
	}
	for _, s := range ss[:topN] {
		if reflect.DeepEqual(s.TokenIDs, label) {
			inTopK = 1.0
			break
		}
	}

	return Metric{
		Accurate: accurate,
		InTopK:   inTopK,
	}
}

func (current Summary) append(added Summary) {
	for msmt := range added {
		current[msmt] = append(current[msmt], added[msmt]...)
	}
}

// ValidForValueAdded applies filters on a window to see if we want to evaluate it for Token/Char Value Added
func ValidForValueAdded(l lexer.Lexer, tokens []lexer.Token) bool {
	if l.Lang() == lang.Text {
		return true
	}
	hasIdent := false
	noConstants := true
	noChangeLine := true
	noComments := true
	for _, tok := range tokens {
		if l.IsType(lexer.IDENT, tok) {
			hasIdent = true
		}
		// Filter out the strings that are not BPE encoded
		if l.IsType(lexer.LITERAL, tok) && !l.IsType(lexer.IDENT, tok) {
			_, ok := l.ShouldBPEEncode(tok)
			if !ok {
				noConstants = false
			}
		}
		if l.IsType(lexer.COMMENT, tok) {
			noComments = false
		}
		if l.IsType(lexer.SEMICOLON, tok) || tok.Lit == "\n" {
			noChangeLine = false
		}
	}
	return hasIdent && noConstants && noChangeLine && noComments
}

func getString(l lexer.Lexer, tok lexer.Token) string {
	if l.IsType(lexer.IDENT, tok) || tok.Lit != "" {
		return tok.Lit
	}
	return l.TokenName(tok.Token)
}

func numIdents(l lexer.Lexer, tokens []lexer.Token) int {
	var num int
	for _, t := range tokens {
		if l.IsType(lexer.IDENT, t) {
			num++
		}
	}
	return num
}

func predictionSitesForLang(measurements map[Measurement]float64, r *rand.Rand, ll lexer.Lexer, path string, buf []byte, maxCorrectAhead, seriesLength int) (PredictionSites, error) {
	tokens, err := ll.Lex(buf)
	if err != nil {
		return PredictionSites{}, errors.Wrapf(err, "unable to lex file")
	}

	sites := make(PredictionSites)
	for i, tok := range tokens {
		if ll.Lang() == lang.Text {
			if _, ok := ll.ShouldBPEEncode(tok); !ok {
				// filters out sof, eof, and unknown tokens
				continue
			}
			if strings.TrimSpace(tok.Lit) == "" {
				continue
			}
			if text.CursorInComment(data.NewBuffer(string(buf)).Select(data.Selection{Begin: tok.Start, End: tok.End}), lang.FromFilename(path)) {
				continue
			}
		}

		if i < len(tokens)-maxCorrectAhead {
			if r.Float64() < measurements[CorrectTokensAhead] {
				sites[CorrectTokensAhead] = append(sites[CorrectTokensAhead], PredictionSite{
					Path:   path,
					Tokens: tokens,
					Idx:    i,
				})
			}
		}

		if i <= len(tokens)-seriesLength {
			if r.Float64() < measurements[Series] {
				sites[Series] = append(sites[Series], PredictionSite{
					Path:   path,
					Tokens: tokens,
					Idx:    i,
				})
			}

			if window := tokens[i : i+seriesLength]; ValidForValueAdded(ll, window) {
				if r.Float64() < measurements[TokenValueAdded] {
					sites[TokenValueAdded] = append(sites[TokenValueAdded], PredictionSite{
						Path:   path,
						Tokens: tokens,
						Idx:    i,
					})
				}
			}
		}

		switch {
		case ll.IsType(lexer.IDENT, tok):
			if r.Float64() < measurements[Word] {
				sites[Word] = append(sites[Word], PredictionSite{
					Path:   path,
					Tokens: tokens,
					Idx:    i,
				})
			}
		case ll.IsType(lexer.STRING, tok):
			// Make things easier and trim leading and trailing spaces
			if _, ok := ll.ShouldBPEEncode(tok); !ok {
				break
			}
			if strings.Contains(tok.Lit, " ") {
				break
			}
			if r.Float64() < measurements[String] {
				sites[String] = append(sites[String], PredictionSite{
					Path:   path,
					Tokens: tokens,
					Idx:    i,
				})
			}
			if r.Float64() < measurements[StringCharValueAdded] {
				sites[StringCharValueAdded] = append(sites[StringCharValueAdded], PredictionSite{
					Path:   path,
					Tokens: tokens,
					Idx:    i,
				})
			}
		default:
			if r.Float64() < measurements[Lexical] {
				sites[Lexical] = append(sites[Lexical], PredictionSite{
					Path:   path,
					Tokens: tokens,
					Idx:    i,
				})
			}
		}
	}

	return sites, nil
}
