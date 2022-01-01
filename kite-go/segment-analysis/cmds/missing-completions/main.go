package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	chart "github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type statsByDay struct {
	day time.Time

	endpointStatusCounts map[string]map[string]float64
}

var (
	totalStats       []statsByDay
	seenEndpoints    = make(map[string]bool)
	seenStatuses     = make(map[string]bool)
	seenUserStatuses = make(map[string]map[string]int)
)

func main() {
	var days int
	flag.IntVar(&days, "days", 3, "days of events to retrieve")
	flag.Parse()

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

		currentStats := statsByDay{
			day:                  day.Day,
			endpointStatusCounts: make(map[string]map[string]float64),
		}

		for track := range r.Tracks {
			if track.Event != "Client HTTP Batch" {
				continue
			}
			if !track.Properties["kite_local"].(bool) {
				continue
			}

			uid := tracks.ParseUserID(track)
			if _, ok := seenUserStatuses[uid]; !ok {
				seenUserStatuses[uid] = make(map[string]int)
			}

			endpoints := track.Properties["requests"].(map[string]interface{})
			for endpoint, vals := range endpoints {
				seenEndpoints[endpoint] = true
				statuses := vals.(map[string]interface{})
				if _, ok := currentStats.endpointStatusCounts[endpoint]; !ok {
					currentStats.endpointStatusCounts[endpoint] = make(map[string]float64)
				}
				for status, val := range statuses {
					seenStatuses[status] = true
					if strings.Contains(endpoint, "completions") {
						seenUserStatuses[uid][status]++
					}
					count := val.(float64)
					currentStats.endpointStatusCounts[endpoint][status] += count
				}
			}
		}
		totalStats = append(totalStats, currentStats)
	}

	log.Println("Users with no completions 200s:")
	noSuccess := 0
	for uid, statuses := range seenUserStatuses {
		if _, exists := statuses["200"]; !exists {
			log.Printf("\t%s\n", uid)
			noSuccess++
			for status, count := range statuses {
				if status == "200" {
					continue
				}
				log.Printf("\t\t%s: %d\n", status, count)
			}
		}
	}
	log.Printf("Num users with no completions 200s: %d/%d (%f%%)", noSuccess, len(seenUserStatuses), float64(noSuccess)/float64(len(seenUserStatuses))*100)

	for endpoint := range seenEndpoints {
		graphEndpointStats(endpoint)
	}
}

var colors = []drawing.Color{
	chart.ColorBlue,
	chart.ColorCyan,
	chart.ColorGreen,
	chart.ColorRed,
	chart.ColorOrange,
	chart.ColorYellow,
	chart.ColorAlternateGray,
}

func graphEndpointStats(endpoint string) {
	var days []time.Time
	statusCountsByDay := make(map[string][]float64)
	for _, stats := range totalStats {
		days = append(days, stats.day)
		var total float64
		for status := range seenStatuses {
			total += stats.endpointStatusCounts[endpoint][status]
		}
		for status := range seenStatuses {
			count := stats.endpointStatusCounts[endpoint][status]
			var percentage float64
			if count != 0 {
				percentage = count / total * 100
			}
			statusCountsByDay[status] = append(statusCountsByDay[status], percentage)
		}
	}

	var series []chart.Series
	i := 0
	for status, counts := range statusCountsByDay {
		if status == "200" {
			continue
		}
		var dashes []float64
		if i >= len(colors) {
			dashes = []float64{5.0, 5.0}
		}
		series = append(series, chart.TimeSeries{
			Name:    status,
			XValues: days,
			YValues: counts,
			Style: chart.Style{
				Show:            true,
				StrokeColor:     colors[i%len(colors)],
				StrokeDashArray: dashes,
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      fmt.Sprintf("Status Percentages for %s", endpoint),
		TitleStyle: chart.StyleShow(),
		Background: chart.Style{
			Padding: chart.Box{
				Top:  20,
				Left: 400,
			},
		},
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
			},
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	name := fmt.Sprintf("%s-status-percentages.png", endpoint[strings.LastIndex(endpoint, "/")+1:])
	f, err := os.Create(name)
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	if err := f.Close(); err != nil {
		log.Fatalln(err)
	}
}
