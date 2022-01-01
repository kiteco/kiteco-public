package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

type sampleRates map[siteType]float64

type predictionSite struct {
	Path          string
	BeforeContext []lexer.Token
	Window        []lexer.Token
	Depth         int
}

type predictionSites map[siteType][]predictionSite

type siteType string

const (
	atleast2Idents siteType = "atleast_2_idents"
	atleast1Ident  siteType = "atleast_1_ident"
)

func (s siteType) ValidWindow(window []lexer.Token, ll lexer.Lexer) bool {
	// - never allow comments or literals since the lexer based models
	//   cannot predict these
	// - don't allow line changes just to make our lives easier
	for _, tok := range window {
		if ll.IsType(lexer.COMMENT, tok) {
			return false
		}
		if ll.IsType(lexer.LITERAL, tok) && !ll.IsType(lexer.IDENT, tok) {
			return false
		}

		if ll.IsType(lexer.SEMICOLON, tok) || strings.Contains(tok.Lit, "\n") {
			return false
		}
	}

	var identCount int
	for _, tok := range window {
		if ll.IsType(lexer.IDENT, tok) {
			identCount++
		}
	}

	switch s {
	case atleast2Idents:
		return identCount >= 2
	case atleast1Ident:
		return identCount >= 1
	default:
		panic(fmt.Sprintf("unsupported site type %v", s))
	}
}

const limitTextContextSize = true

func getSites(paths []string, native, text predictorBundle, sampleRates sampleRates, windowSize int) (predictionSites, predictionSites) {
	getTextTokens := func(contents []byte, start, end int) []lexer.Token {
		tokens, err := text.GetEncoder().Lexer.Lex(contents[start:end])
		fail(err)
		// remove eof
		tokens = tokens[:len(tokens)-1]

		return tokens
	}

	trimTextLeft := func(toks []lexer.Token, limit int) []lexer.Token {
		if !limitTextContextSize {
			return toks
		}
		var count int
		for i := len(toks) - 1; i >= 0; i-- {
			bp := len(native.GetEncoder().EncodeTokens([]lexer.Token{toks[i]}))
			if bp+count >= limit {
				return toks[i+1:]
			}
			count += bp
		}
		return toks
	}

	nativeSites := make(predictionSites)
	textSites := make(predictionSites)
	for _, path := range paths {
		contents, err := ioutil.ReadFile(path)
		fail(err)

		nativeTokens, err := native.GetEncoder().Lexer.Lex(contents)
		fail(err)

		for iNative, tokNative := range nativeTokens {
			if iNative+windowSize >= len(nativeTokens)-1 {
				break
			}
			nativeWindow := nativeTokens[iNative : iNative+windowSize]
			nativeDepth := len(native.GetEncoder().EncodeTokens(nativeWindow))
			if nativeDepth > native.GetHParams().NumPredictionSlots {
				continue
			}

			nativeNumBPsBefore := len(native.GetEncoder().EncodeTokens(nativeTokens[:iNative]))
			if nativeNumBPsBefore > native.Search.Window {
				nativeNumBPsBefore = native.Search.Window
			}

			textBeforeTokens := trimTextLeft(getTextTokens(contents, 0, tokNative.Start), nativeNumBPsBefore)

			textWindowTokens := getTextTokens(contents, tokNative.Start, nativeWindow[len(nativeWindow)-1].End)

			textDepth := len(text.GetEncoder().EncodeTokens(textWindowTokens))
			if textDepth > text.GetHParams().NumPredictionSlots {
				continue
			}

			for st, rate := range sampleRates {
				if !st.ValidWindow(nativeWindow, native.GetEncoder().Lexer) {
					continue
				}
				if rand.Float64() >= rate {
					continue
				}
				nativeSites[st] = append(nativeSites[st], predictionSite{
					Path:          path,
					BeforeContext: nativeTokens[:iNative],
					Window:        nativeWindow,
					Depth:         nativeDepth,
				})
				textSites[st] = append(textSites[st], predictionSite{
					Path:          path,
					BeforeContext: textBeforeTokens,
					Window:        textWindowTokens,
					Depth:         textDepth,
				})
			}
		}
	}
	return nativeSites, textSites
}
