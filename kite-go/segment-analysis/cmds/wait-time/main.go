package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	chart "github.com/wcharczuk/go-chart"
)

var (
	maxWaitTime float64
	cutoff      = 3 * time.Second
	aboveCutoff = make(map[string][]float64)

	sources = map[string]bool{
		"request": true,
		"change":  true,
	}
	platforms = map[string]bool{
		"windows": true,
		"darwin":  true,
	}

	// maps of platform -> source -> client version -> durations
	waitDurations     = make(map[string]map[string]map[string][]float64)
	longWaitDurations = make(map[string]map[string]map[string][]float64)
)

func main() {
	var days int
	var verbose bool
	flag.IntVar(&days, "days", 3, "days of events to receive")
	flag.BoolVar(&verbose, "verbose", false, "print details about users with long wait times")
	flag.Parse()

	// init
	for platform := range platforms {
		waitDurations[platform] = make(map[string]map[string][]float64)
		longWaitDurations[platform] = make(map[string]map[string][]float64)
		for source := range sources {
			waitDurations[platform][source] = make(map[string][]float64)
			longWaitDurations[platform][source] = make(map[string][]float64)
		}
	}

	listing, err := tracks.List(tracks.Bucket, tracks.ClientEventSource)
	if err != nil {
		log.Fatalln(err)
	}

	for idx, day := range listing.Days {
		if idx < len(listing.Days)-days {
			continue
		}
		r := tracks.NewReader(tracks.Bucket, day.Keys, 8)
		go r.StartAndWait()

		for track := range r.Tracks {
			if track.Event != "Index Build" {
				continue
			}
			if track.Properties["source"] == nil {
				continue
			}
			if track.Properties["wait_duration_ns"] == nil {
				continue
			}
			platform := track.Properties["platform"].(string)
			if _, exists := platforms[platform]; !exists {
				continue
			}
			source := track.Properties["source"].(string)
			if _, exists := sources[source]; !exists {
				continue
			}
			clientVersion := track.Properties["client_version"].(string)

			// store wait durations
			duration := track.Properties["wait_duration_ns"].(float64)
			waitDurations[platform][source][clientVersion] = append(waitDurations[platform][source][clientVersion], duration/float64(time.Millisecond))
			if duration > maxWaitTime {
				maxWaitTime = duration
			}
			if duration > float64(cutoff) {
				uid := tracks.ParseUserID(track)
				aboveCutoff[uid] = append(aboveCutoff[uid], duration)
				longWaitDurations[platform][source][clientVersion] = append(longWaitDurations[platform][source][clientVersion], duration/float64(time.Millisecond))
			}
		}
	}

	if verbose {
		log.Printf("Users with wait time gt %s: %d", cutoff, len(aboveCutoff))
		for platform := range platforms {
			for source := range sources {
				log.Printf("%s, %s:\n", platform, source)
				for uid, durations := range aboveCutoff {
					var secondDurations []float64
					for _, d := range durations {
						secondDurations = append(secondDurations, d/float64(time.Second))
					}
					log.Printf("\t%s: %v\n", uid, secondDurations)
				}
			}
		}
	}
	log.Printf("Max wait time (s): %f", maxWaitTime/float64(time.Second))

	for platform := range platforms {
		for source := range sources {
			ts := time.Now()
			year, month, day := ts.Date()
			suffix := fmt.Sprintf("%s-%s-%d-%d-%d", platform, source, int(month), day, year)
			graphWaitDurations(waitDurations[platform][source], suffix)
			graphWaitDurations(longWaitDurations[platform][source], fmt.Sprintf("%s-%s", "long", suffix))
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

func graphWaitDurations(durations map[string][]float64, suffix string) {
	if len(durations) == 0 {
		return
	}

	i := 0
	var series []chart.Series
	for clientVersion, clientDurations := range durations {
		waitPercentiles := computePercentiles(clientDurations, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    clientVersion,
			XValues: chartPercentiles,
			YValues: waitPercentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      "Wait Duration Percentiles",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Wait time (ms)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	f, err := os.Create(fmt.Sprintf("wait-time-percentiles-%s.png", suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
