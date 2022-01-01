package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
)

type metricType string

const (
	encodingLengthType metricType = "encoding_length"
	byteLengthType     metricType = "byte_length"
	numSpacesType      metricType = "num_spaces"
)

var metricTypes = []metricType{encodingLengthType, byteLengthType, numSpacesType}

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
	case encodingLengthType, numSpacesType, byteLengthType:
		return []string{
			fmt.Sprintf("%v_median", t),
			fmt.Sprintf("%v_mean", t),
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

type siteMetricsByExt map[string]siteMetrics

func newSiteMetrics(sites sitesByExt, text encoderBundle) siteMetricsByExt {
	byExt := make(siteMetricsByExt)

	for ext, sites := range sites {
		byExt[ext] = make(siteMetrics)
		for st, ps := range sites {
			var encodingLength, byteLength, numSpaces []float64
			for _, p := range ps {
				encodingLength = append(encodingLength, float64(len(text.EncodeTokens(p.Window))))

				var space int
				for _, tok := range p.Window {
					if len(strings.TrimSpace(tok.Lit)) == 0 {
						space++
					}
				}
				numSpaces = append(numSpaces, float64(space))

				var byteLen int
				for _, tok := range p.Window {
					byteLen += len(tok.Lit)
				}

				byteLength = append(byteLength, float64(byteLen))
			}

			byExt[ext][st] = metrics{
				Count: len(encodingLength),
				Metrics: map[metricType]metric{
					encodingLengthType: newMetric(encodingLength),
					byteLengthType:     newMetric(byteLength),
					numSpacesType:      newMetric(numSpaces),
				},
			}
		}
	}

	return byExt
}

func computeMetrics(outPath string, sampleRates sampleRates, gen inspect.CodeGenerator, text encoderBundle, numSites int, windows []int) {
	start := time.Now()

	type metricsAndWindow struct {
		Window int
		Text   siteMetricsByExt
	}

	var ms []metricsAndWindow
	for _, windowSize := range windows {
		textSites := getSites(gen, text, sampleRates, windowSize, numSites)

		ms = append(ms, metricsAndWindow{
			Window: windowSize,
			Text:   newSiteMetrics(textSites, text),
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

	fmt.Fprintf(out, "Vocab: %s\n", text.Vocab)

	sep := "\t"

	header := []string{"site_type", "window_size", "count", "ext"}
	for _, mt := range metricTypes {
		header = append(header, metric{}.Header(mt)...)
	}
	fmt.Fprintf(out, "%s\n", strings.Join(header, sep))

	for _, mw := range ms {
		row := func(st siteType, metrics metrics, ext string) {
			row := []string{
				string(st),
				fmt.Sprintf("%d", mw.Window),
				fmt.Sprintf("%d", metrics.Count),
				ext,
			}
			for _, mt := range metricTypes {
				row = append(row, metrics.Metrics[mt].Row(mt)...)
			}
			fmt.Fprintf(out, "%s\n", strings.Join(row, sep))
		}

		var exts []string
		for ext := range mw.Text {
			exts = append(exts, ext)
		}
		sort.Strings(exts)

		for _, st := range siteTypes {
			for _, ext := range exts {
				row(st, mw.Text[ext][st], ext)
			}
		}
	}

	fmt.Fprintf(out, "Done! took %v to compute metrics\n", time.Since(start))
}
