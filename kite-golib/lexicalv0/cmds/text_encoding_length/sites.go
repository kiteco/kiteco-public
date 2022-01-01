package main

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
)

type sampleRates map[siteType]float64

type predictionSite struct {
	Path   string
	Window []lexer.Token
}

type predictionSites map[siteType][]predictionSite

type sitesByExt map[string]predictionSites

func (byExt sitesByExt) HaveMinSites(minSites int) bool {
	if len(byExt) == 0 {
		return false
	}

	for _, sites := range byExt {
		for _, ps := range sites {
			if len(ps) < minSites {
				return false
			}
		}
	}
	return true
}

type siteType string

const (
	atleast2Idents siteType = "atleast_2_idents"
	atleast1Ident  siteType = "atleast_1_ident"
)

func (s siteType) ValidWindow(window []lexer.Token, ll lexer.Lexer, buf, path string) bool {
	var numComments int
	for _, tok := range window {
		if _, ok := ll.ShouldBPEEncode(tok); !ok {
			// filters out sof, eof, and unknown tokens
			return false
		}
		if strings.TrimSpace(tok.Lit) == "" {
			// skip windows that contain whitespace only tokens
			return false
		}
		if text.CursorInComment(data.NewBuffer(buf).Select(data.Selection{Begin: tok.Start, End: tok.End}), lang.FromFilename(path)) {
			numComments++
		}
	}

	// for small windows we don't allow comments,
	// as window gets larger we make sure comments are
	// less than half ... TODO: hacky
	switch len(window) {
	case 0, 1, 2, 3:
		if numComments > 0 {
			return false
		}
	default:
		if numComments > 3 {
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

const maxFiles = int(1e5)

func getSites(gen inspect.CodeGenerator, text encoderBundle, sampleRates sampleRates, windowSize, numSites int) sitesByExt {
	byExt := make(sitesByExt)

	for i := 0; i < maxFiles; i++ {
		if i == maxFiles-1 {
			fmt.Printf("HIT MAX NUM FILES LIMIT OF %d\n", maxFiles)
		}
		code, path, err := gen.Next()
		fail(err)

		ext := filepath.Ext(path)
		if byExt[ext] == nil {
			byExt[ext] = make(predictionSites)
		}

		toks, err := text.Lexer.Lex([]byte(code))
		fail(err)

		for j := 0; j < len(toks)-windowSize; j++ {
			window := toks[j : j+windowSize]
			for st, rate := range sampleRates {
				if !st.ValidWindow(window, text.Lexer, code, path) {
					continue
				}
				if rand.Float64() >= rate {
					continue
				}

				byExt[ext][st] = append(byExt[ext][st], predictionSite{
					Path:   path,
					Window: window,
				})
			}
		}

		if byExt.HaveMinSites(numSites) {
			break
		}
	}
	return byExt
}
