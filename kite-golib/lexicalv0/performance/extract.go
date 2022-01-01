package performance

import (
	"math/rand"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

// Extractor extracts prediction sites that will be used in Eval
type Extractor struct {
	Encoder      *lexicalv0.FileEncoder
	Rand         *rand.Rand
	RandomSeed   int64
	Measurements map[Measurement]float64
	SeriesLength int
}

// PredictionSite ...
type PredictionSite struct {
	Path   string
	Tokens []lexer.Token
	Idx    int
}

// PredictionSites ...
type PredictionSites map[Measurement][]PredictionSite

// ExtractPredictionSites ...
func (e *Extractor) ExtractPredictionSites(buf []byte, filename string, maxCorrectAhead int) (PredictionSites, error) {
	e.Rand.Seed(e.RandomSeed)

	nativeLexer := e.Encoder.Lexer
	if nativeLexer.Lang() == lang.Text {
		// Only use the Go/Py/JS lexers since the other ones are not fully implemented
		langForFile := lexicalmodels.LanguageGroupDeprecated(lang.FromFilename(filename))
		switch langForFile {
		case lang.Golang, lang.JavaScript, lang.Python:
			var err error
			nativeLexer, err = lexicalv0.NewLexer(langForFile)
			if err != nil {
				nativeLexer = e.Encoder.Lexer
			}
		}
	}

	sites, err := predictionSitesForLang(
		e.Measurements,
		e.Rand,
		nativeLexer,
		filename,
		buf,
		maxCorrectAhead,
		e.SeriesLength,
	)

	if err != nil {
		return PredictionSites{}, err
	}

	if e.Encoder.Lexer.Lang() == nativeLexer.Lang() {
		return sites, nil
	}

	textTokens, err := e.Encoder.Lexer.Lex(buf)
	if err != nil {
		return PredictionSites{}, errors.New("unable to use text lexer for %s", filename)
	}

	updateSites := func(m Measurement, ps []PredictionSite) ([]PredictionSite, error) {
		var updated []PredictionSite
		for _, p := range ps {
			tokOrig := p.Tokens[p.Idx]

			var iNew int
			var tokNew lexer.Token
			for iNew, tokNew = range textTokens {
				if tokNew.Start == tokOrig.Start {
					break
				}
			}

			if tokNew == (lexer.Token{}) {
				return nil, errors.New("unable to find new token for measurement %v, %v", m, tokOrig)
			}

			var checkLen int
			switch m {
			case Series, TokenValueAdded:
				checkLen = e.SeriesLength
			case CorrectTokensAhead:
				checkLen = maxCorrectAhead
			}
			if iNew+checkLen > len(textTokens) {
				continue
			}

			updated = append(updated, PredictionSite{
				Tokens: textTokens,
				Idx:    iNew,
			})
		}
		return updated, nil
	}

	updatedSites := make(PredictionSites)
	for m, ss := range sites {
		updated, err := updateSites(m, ss)
		if err != nil {
			return nil, errors.Wrapf(err, "error updating sites for %v", m)
		}
		updatedSites[m] = updated
	}

	return updatedSites, nil
}
