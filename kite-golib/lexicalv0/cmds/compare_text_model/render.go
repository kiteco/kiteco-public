package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
)

type renderDiff struct {
	Expected           string
	ExpectedNormalized string
	Actual             string
	ActualNormalized   string
}

// expected and actual contain text tokens
func newRenderDiff(expected, actual []lexer.Token, nativeLexer lexer.Lexer) renderDiff {
	// if expected and actual are identical, we must have rendered properly
	if reflect.DeepEqual(expected, actual) {
		return renderDiff{}
	}

	// skip newline completions
	for _, toks := range [][]lexer.Token{expected, actual} {
		for _, tok := range toks {
			if strings.Contains(tok.Lit, "\n") {
				return renderDiff{}
			}
		}
	}

	expectedNormalized, err := Normalize(expected, nativeLexer)
	fail(err)

	actualNormalized, err := Normalize(actual, nativeLexer)
	fail(err)

	expectedRendered, _ := text.Render(nativeLexer.Lang(), []lexer.Token{}, expected)
	actualRendered, _ := text.Render(nativeLexer.Lang(), []lexer.Token{}, actual)

	if expectedNormalized != actualNormalized {
		return renderDiff{
			Expected:           expectedRendered.ForFormat(),
			ExpectedNormalized: expectedNormalized,
			Actual:             actualRendered.ForFormat(),
			ActualNormalized:   actualNormalized,
		}
	}

	return renderDiff{}
}

func (d renderDiff) Print(out io.Writer) {
	if d == (renderDiff{}) {
		return
	}

	fmt.Fprintln(out, "---")
	fmt.Fprintf(out, "Expected: %s\n", d.Expected)
	fmt.Fprintf(out, "Actual: %s\n", d.Actual)
	fmt.Fprintf(out, "ExpectedNormalized: %s\n", d.ExpectedNormalized)
	fmt.Fprintf(out, "ActualNormalized: %s\n", d.ActualNormalized)
}

type renderDiffs map[siteType][]renderDiff

func newRenderDiffs(sites predictionSites, native, text predictorBundle) renderDiffs {
	renderDiffs := make(renderDiffs)
	for st, ps := range sites {
		var diffs []renderDiff
		for _, p := range ps {
			text.Search.Depth = p.Depth
			in := predict.Inputs{
				FilePath:       p.Path,
				Tokens:         p.BeforeContext,
				CursorTokenIdx: len(p.BeforeContext),
				SearchConfig:   text.Search,
			}
			preds, err := text.Predict(kitectx.Background(), in)
			fail(err)

			for _, pred := range preds.Preds {
				diffs = append(diffs, newRenderDiff(p.Window, pred.Tokens, native.GetEncoder().Lexer))
			}

			if len(diffs) > 10 {
				break
			}
		}

		renderDiffs[st] = diffs
	}
	return renderDiffs
}

func computeRenderDiffs(outPath string, sampleRates sampleRates, files []string, native, text predictorBundle) {
	start := time.Now()

	type diffsAndWindow struct {
		Window int
		Diffs  renderDiffs
	}

	var diffs []diffsAndWindow
	for _, windowSize := range []int{3, 5} {
		_, textSites := getSites(files, native, text, sampleRates, windowSize)

		diffs = append(diffs, diffsAndWindow{
			Window: windowSize,
			Diffs:  newRenderDiffs(textSites, native, text),
		})
	}

	out := io.Writer(os.Stdout)
	if outPath != "" {
		f, err := os.Create(outPath)
		fail(err)
		defer f.Close()

		out = io.MultiWriter(out, f)
	}

	var siteTypes []siteType
	for st := range sampleRates {
		siteTypes = append(siteTypes, st)
	}
	sort.Slice(siteTypes, func(i, j int) bool {
		return siteTypes[i] < siteTypes[j]
	})

	for _, diff := range diffs {
		fmt.Fprintf(out, "Window: %d\n", diff.Window)
		for _, st := range siteTypes {
			fmt.Fprintf(out, "SiteType: %v\n", st)
			for _, d := range diff.Diffs[st] {
				d.Print(out)
			}
		}
	}

	fmt.Fprintf(out, "Done! took %v to compute diffs\n", time.Since(start))
}
