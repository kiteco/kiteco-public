package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	chart "github.com/wcharczuk/go-chart"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

func main() {
	listing, err := tracks.List(tracks.Bucket, tracks.ClientEventSource)
	if err != nil {
		log.Fatalln(err)
	}

	filter := func(t *analytics.Track) bool {
		return t.Event == "Editor Event"
	}

	for idx, day := range listing.Days {
		if idx < len(listing.Days)-3 {
			continue
		}

		log.Println("building report for", day.Day.Format("2006-01-02"))

		r := tracks.NewFilteredReader(tracks.Bucket, day.Keys, filter, 32)
		go r.StartAndWait()

		var delay, rtt, backend, total, threshold []float64
		for track := range r.Tracks {
			switch track.Event {
			case "Editor Event":
				d := track.Properties["client_delay_ns"].(float64)
				rt := track.Properties["duration_ns"].(float64)
				b := track.Properties["backend_duration_ns"].(float64)

				if d > 0 && rt > 0 && b > 0 {
					delay = append(delay, d/float64(time.Millisecond))
					rtt = append(rtt, rt/float64(time.Millisecond))
					backend = append(backend, b/float64(time.Millisecond))
					total = append(total, (d+rt)/float64(time.Millisecond))
					threshold = append(threshold, 100.0)
				}
			}
		}

		delayPercentiles := computePercentiles(delay, chartPercentiles)
		rttPercentiles := computePercentiles(rtt, chartPercentiles)
		backendPercentiles := computePercentiles(backend, chartPercentiles)
		totalPercentiles := computePercentiles(total, chartPercentiles)

		names := map[string][]float64{
			"delay":   delayPercentiles,
			"rtt":     rttPercentiles,
			"backend": backendPercentiles,
			"total":   totalPercentiles,
		}

		var series []chart.Series
		for name, values := range names {
			series = append(series, chart.ContinuousSeries{
				Name:    fmt.Sprintf("%s", name),
				XValues: chartPercentiles,
				YValues: values,
			})
		}
		series = append(series, chart.ContinuousSeries{
			Name:    "100ms threshold",
			XValues: chartPercentiles,
			YValues: threshold,
			Style: chart.Style{
				Show:            true,
				StrokeColor:     chart.ColorRed,
				StrokeDashArray: []float64{5.0, 5.0},
			},
		})

		graph := chart.Chart{
			XAxis: chart.XAxis{
				Name:      "Percentiles",
				NameStyle: chart.StyleShow(),
				Style:     chart.StyleShow(),
			},
			YAxis: chart.YAxis{
				Name:      "Latency (ms)",
				NameStyle: chart.StyleShow(),
				Style:     chart.StyleShow(),
			},
			Series: series,
		}

		graph.Elements = []chart.Renderable{
			chart.Legend(&graph),
		}

		f, err := os.Create(fmt.Sprintf("event-latency-%s.png", day.Day.Format("2006-01-02")))
		if err != nil {
			log.Fatalln(err)
		}

		graph.Render(chart.PNG, f)

		err = f.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}
}

var (
	chartPercentiles []float64
)

func init() {
	for i := float64(0); i <= 1.0; i += 0.01 {
		chartPercentiles = append(chartPercentiles, i)
	}
}

type userLatency struct {
	uid int64

	delay            []float64
	delayPercentiles []float64

	rtt            []float64
	rttPercentiles []float64

	total            []float64
	totalPercentiles []float64

	backend            []float64
	backendPercentiles []float64
}

func computePercentiles(samples, percentiles []float64) []float64 {
	sort.Float64s(samples)

	var ret []float64
	for _, p := range percentiles {
		idx := int(float64(len(samples)) * p)
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		ret = append(ret, samples[idx])
	}

	return ret
}
