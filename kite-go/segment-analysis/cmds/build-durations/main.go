package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	chart "github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type stage struct {
	name     string
	duration float64
}

type byDuration []stage

func (d byDuration) Len() int           { return len(d) }
func (d byDuration) Less(i, j int) bool { return d[i].duration < d[j].duration }
func (d byDuration) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }

var (
	filterTypes = map[string]bool{
		"none":    true,
		"success": true,
		"timeout": true,
	}
	platforms = map[string]bool{
		"windows": true,
		"darwin":  true,
	}

	// map of platform -> filter type -> durations
	totalDurations = make(map[string]map[string][]float64)
	// map of platform -> filter type -> stage name -> durations
	durations = make(map[string]map[string]map[string][]float64)
	// map of platform -> filter type -> stage name -> count
	longest = make(map[string]map[string]map[string]int)
	// map of platform -> filter type -> client version -> file counts
	fileCounts = make(map[string]map[string]map[string][]float64)
)

func main() {
	var days int
	flag.IntVar(&days, "days", 3, "days of events to receive")
	flag.Parse()

	// init
	for platform := range platforms {
		totalDurations[platform] = make(map[string][]float64)
		durations[platform] = make(map[string]map[string][]float64)
		longest[platform] = make(map[string]map[string]int)
		fileCounts[platform] = make(map[string]map[string][]float64)
		for filter := range filterTypes {
			durations[platform][filter] = make(map[string][]float64)
			longest[platform][filter] = make(map[string]int)
			fileCounts[platform][filter] = make(map[string][]float64)
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

			if track.Properties["build_durations"] == nil {
				continue
			}

			var filter string
			if track.Properties["error"].(string) == "" {
				filter = "success"
			}

			errStr := track.Properties["error"].(string)
			if strings.Contains(errStr, "kitectx") {
				filter = "timeout"
			}
			platform := track.Properties["platform"].(string)
			if _, exists := platforms[platform]; !exists {
				continue
			}
			clientVersion := track.Properties["client_version"].(string)

			// store stage durations
			buildDurations := track.Properties["build_durations"].(map[string]interface{})
			var stages []stage
			for name, duration := range buildDurations {
				stages = append(stages, stage{
					name:     name,
					duration: duration.(float64) / float64(time.Second),
				})
			}
			sort.Sort(sort.Reverse(byDuration(stages)))
			for _, s := range stages {
				durations[platform]["none"][s.name] = append(durations[platform]["none"][s.name], s.duration)
				if filter != "" {
					durations[platform][filter][s.name] = append(durations[platform][filter][s.name], s.duration)
				}
			}
			// longest duration will be first in reverse order
			if len(stages) > 0 {
				longest[platform]["none"][stages[0].name]++
				if filter != "" {
					longest[platform][filter][stages[0].name]++
				}
			}

			totalDuration := track.Properties["since_start_ns"].(float64) / float64(time.Second)
			totalDurations[platform]["none"] = append(totalDurations[platform]["none"], totalDuration)
			if filter != "" {
				totalDurations[platform][filter] = append(totalDurations[platform][filter], totalDuration)
			}
			var fileCount float64
			if track.Properties["filtered_files"] == nil {
				fileCount = track.Properties["files"].(float64)
			} else {
				fileCount = track.Properties["filtered_files"].(float64)
			}
			fileCounts[platform]["none"][clientVersion] = append(fileCounts[platform]["none"][clientVersion], fileCount)
			if filter != "" {
				fileCounts[platform][filter][clientVersion] = append(fileCounts[platform][filter][clientVersion], fileCount)
			}
		}
	}

	log.Println("Number of times stage took longest:")
	for platform := range platforms {
		for filter := range filterTypes {
			log.Printf("%s, %s:\n", platform, filter)
			for name, count := range longest[platform][filter] {
				log.Printf("\t%s: %d\n", name, count)
			}
			log.Println("---")
		}
	}

	for platform := range platforms {
		for filter := range filterTypes {
			ts := time.Now()
			year, month, day := ts.Date()
			suffix := fmt.Sprintf("%s-%s-%d-%d-%d", platform, filter, int(month), day, year)
			graphStageDurations(durations[platform][filter], totalDurations[platform][filter], suffix)
			graphFileCounts(fileCounts[platform][filter], suffix)
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

var colors = []drawing.Color{
	chart.ColorBlue,
	chart.ColorCyan,
	chart.ColorGreen,
	chart.ColorOrange,
	chart.ColorAlternateGray,
}

func graphStageDurations(stageDurations map[string][]float64, total []float64, suffix string) {
	if len(durations) == 0 {
		return
	}

	var series []chart.Series
	i := 0
	for stage, durations := range stageDurations {
		percentiles := computePercentiles(durations, chartPercentiles)

		var dashes []float64
		if i >= len(colors) {
			dashes = []float64{5.0, 5.0}
		}

		series = append(series, chart.ContinuousSeries{
			Name:    stage,
			XValues: chartPercentiles,
			YValues: percentiles,
			Style: chart.Style{
				Show:            true,
				StrokeColor:     chart.GetAlternateColor(i),
				StrokeDashArray: dashes,
			},
		})
		i++
	}
	totalPercentiles := computePercentiles(total, chartPercentiles)
	series = append(series, chart.ContinuousSeries{
		Name:    "total",
		XValues: chartPercentiles,
		YValues: totalPercentiles,
		Style: chart.Style{
			Show:            true,
			StrokeColor:     chart.ColorRed,
			StrokeDashArray: []float64{1.0, 1.0},
		},
	})

	graph := chart.Chart{
		Title:      "Percentiles for Build Stage Durations",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Build time (s)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	f, err := os.Create(fmt.Sprintf("build-stage-percentiles-%s.png", suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphFileCounts(counts map[string][]float64, suffix string) {
	if len(counts) == 0 {
		return
	}

	i := 0
	var series []chart.Series
	for clientVersion, clientCounts := range counts {
		filePercentiles := computePercentiles(clientCounts, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    clientVersion,
			XValues: chartPercentiles,
			YValues: filePercentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      "Percentiles for Files in Index",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "num files",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	f, err := os.Create(fmt.Sprintf("file-count-percentiles-%s.png", suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
