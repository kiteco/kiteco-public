package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

type metricType string

const (
	durationType  metricType = "duration"
	depthType     metricType = "depth"
	numTokensType metricType = "num_tokens"
	numSpacesType metricType = "num_spaces"
)

var metricTypes = []metricType{durationType, depthType, numTokensType, numSpacesType}

type metric struct {
	Median float64
	Mean   float64
}

func newMetric(vs []float64) metric {
	var sum float64
	for _, v := range vs {
		sum += v
	}

	sort.Slice(vs, func(i, j int) bool {
		return vs[i] < vs[j]
	})

	var median float64
	middle := len(vs) / 2
	if len(vs)%2 == 0 {
		median = (vs[middle] + vs[middle-1]) / 2.
	} else {
		median = vs[middle]
	}

	return metric{
		Median: median,
		Mean:   sum / float64(len(vs)),
	}
}

func (m metric) Row(t metricType) []string {
	return []string{
		fmt.Sprintf("%.2f", m.Median),
		fmt.Sprintf("%.2f", m.Mean),
	}
}

func (metric) Header(t metricType) []string {
	switch t {
	case depthType, numTokensType, numSpacesType:
		return []string{
			fmt.Sprintf("%v_median", t),
			fmt.Sprintf("%v_mean", t),
		}
	case durationType:
		return []string{
			fmt.Sprintf("%v_median_ms", t),
			fmt.Sprintf("%v_mean_ms", t),
		}
	default:
		panic(fmt.Sprintf("unsupported  metric type %v", t))
	}
}

type metrics struct {
	Count   int
	Metrics map[metricType]metric
}

type siteMetrics map[siteType]metrics

func newSiteMetrics(sites predictionSites, pb predictorBundle) siteMetrics {
	var completedFirstPred bool
	siteMetrics := make(siteMetrics)
	for st, ps := range sites {
		var durations, depths, numTokens, numSpaces []float64
		for _, p := range ps {
			pb.Search.Depth = p.Depth
			in := predict.Inputs{
				FilePath:       p.Path,
				Tokens:         p.BeforeContext,
				CursorTokenIdx: len(p.BeforeContext),
				SearchConfig:   pb.Search,
			}
			start := time.Now()
			_, err := pb.Predict(kitectx.Background(), in)
			fail(err)

			if !completedFirstPred {
				completedFirstPred = true
				continue
			}

			durations = append(durations, float64(time.Since(start).Milliseconds()))
			depths = append(depths, float64(p.Depth))
			numTokens = append(numTokens, float64(len(p.Window)))

			var space int
			for _, tok := range p.Window {
				if len(strings.TrimSpace(tok.Lit)) == 0 && pb.GetEncoder().Lexer.Lang() == lang.Text {
					space++
				}
			}
			numSpaces = append(numSpaces, float64(space))
		}

		siteMetrics[st] = metrics{
			Count: len(durations),
			Metrics: map[metricType]metric{
				durationType:  newMetric(durations),
				depthType:     newMetric(depths),
				numTokensType: newMetric(numTokens),
				numSpacesType: newMetric(numSpaces),
			},
		}
	}
	return siteMetrics
}

func computeMetrics(outPath string, sampleRates sampleRates, files []string, native, text predictorBundle) {
	start := time.Now()

	// to make times comparable, set minp to 0 for both, otherwise the model with
	// a lower minp will take longer.
	// text.Search.MinP = 0
	// native.Search.MinP = 0

	type metricsAndWindow struct {
		Window int
		Native siteMetrics
		Text   siteMetrics
	}

	var ms []metricsAndWindow
	for _, windowSize := range []int{1, 3, 5} {
		nativeSites, textSites := getSites(files, native, text, sampleRates, windowSize)

		nativeMetrics := newSiteMetrics(nativeSites, native)
		textMetrics := newSiteMetrics(textSites, text)
		ms = append(ms, metricsAndWindow{
			Window: windowSize,
			Native: nativeMetrics,
			Text:   textMetrics,
		})
	}

	out := io.Writer(os.Stdout)
	if outPath != "" {
		f, err := os.Create(outPath)
		fail(err)
		defer f.Close()

		out = io.MultiWriter(out, f)
	}

	fmt.Fprintf(out, "Native Search: %+v\n", native.Search)
	fmt.Fprintf(out, "Text Search: %+v\n", text.Search)

	var siteTypes []siteType
	for st := range sampleRates {
		siteTypes = append(siteTypes, st)
	}
	sort.Slice(siteTypes, func(i, j int) bool {
		return siteTypes[i] < siteTypes[j]
	})

	sep := "\t"

	header := []string{"site_type", "native_window_size", "count", "is_native"}
	for _, mt := range metricTypes {
		header = append(header, metric{}.Header(mt)...)
	}
	fmt.Fprintf(out, "%s\n", strings.Join(header, sep))

	for _, mw := range ms {
		row := func(st siteType, metrics metrics, isNative string) {
			row := []string{
				string(st),
				fmt.Sprintf("%d", mw.Window),
				fmt.Sprintf("%d", metrics.Count),
				isNative,
			}
			for _, mt := range metricTypes {
				row = append(row, metrics.Metrics[mt].Row(mt)...)
			}
			fmt.Fprintf(out, "%s\n", strings.Join(row, sep))
		}

		for _, st := range siteTypes {
			row(st, mw.Native[st], "yes")
			row(st, mw.Text[st], "no")
		}
	}

	fmt.Fprintf(out, "Done! took %v to compute metrics\n", time.Since(start))
}
