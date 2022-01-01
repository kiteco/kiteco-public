package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
	chart "github.com/wcharczuk/go-chart"
)

var (
	// supported platforms
	platforms = map[string]bool{
		"darwin":  true,
		"windows": true,
		"linux":   true,
	}

	// map of platform -> client version -> event
	walkEvents = make(map[string]map[string][]walkEvent)
	// possible values recorded in a walk event
	dataTypes = map[string]bool{
		"duration": true,
		"walked":   true,
		"library":  true,
	}
)

type walkEvent struct {
	durationNs  float64
	walkedDirs  float64
	libraryDirs float64
}

type uidType interface{}

type tracked struct {
	Platform      string  `json:"platform"`
	ClientVersion string  `json:"client_version"`
	DurationNs    float64 `json:"since_start_ns"`
	WalkedDirs    float64 `json:"walked_dirs"`
	LibraryDirs   float64 `json:"library_dirs"`
	// deprecated
	ScannedDirs float64 `json:"scanned_dirs"`
}

func main() {
	var days, shift int
	var minVersion string
	var groupByMonth bool
	flag.IntVar(&days, "days", 3, "days of events to receive")
	flag.IntVar(&shift, "shift", 0, "how many days back to shift window of events")
	flag.StringVar(&minVersion, "minVersion", "", "minimum client version (YYYYMMDD or YYMM)")
	flag.BoolVar(&groupByMonth, "groupByMonth", false, "group client version by month")
	flag.Parse()

	for platform := range platforms {
		walkEvents[platform] = make(map[string][]walkEvent)
	}

	end := analyze.Today()
	start := end.Add(0, 0, -days)
	if shift > 0 {
		start = start.Add(0, 0, -shift)
		end = start.Add(0, 0, days)
	}
	listing, err := analyze.ListRange(segmentsrc.ClientEventsTrimmed, start, end)
	if err != nil {
		log.Fatalln(err)
	}

	var uris []string
	for _, day := range listing.Dates {
		uris = append(uris, day.URIs...)
	}

	analyze.Analyze(uris, 4, "Background Library Walk Completed", func(meta analyze.Metadata, track *tracked) bool {
		if meta.EventName != "Background Library Walk Completed" {
			return true
		}
		platform := track.Platform
		if _, exists := platforms[platform]; !exists {
			return true
		}

		clientVersion := tracks.VersionToDate(track.ClientVersion, groupByMonth)
		if clientVersion == "" {
			return true
		}
		if minVersion != "" && clientVersion < minVersion {
			return true
		}

		event := walkEvent{
			walkedDirs: track.WalkedDirs,
		}

		w := track.DurationNs
		if w > 0 {
			event.durationNs = w / float64(time.Second)
		}

		f := track.LibraryDirs
		if f == 0 {
			f = track.ScannedDirs
		}
		event.libraryDirs = f

		walkEvents[platform][clientVersion] = append(walkEvents[platform][clientVersion], event)

		return true
	})

	// create graphs for each platform
	ts := time.Now()
	year, month, day := ts.Date()
	for platform := range platforms {
		suffix := fmt.Sprintf("%s-%d-%d-%d", platform, int(month), day, year)
		graphPercentiles(walkEvents[platform], "duration", "Walk Duration(s)", "seconds", fmt.Sprintf("walk-duration-percentiles-%s.png", suffix))
		graphPercentiles(walkEvents[platform], "walked", "Num Directories Walked", "count", fmt.Sprintf("walked-dirs-percentiles-%s.png", suffix))
		graphPercentiles(walkEvents[platform], "library", "Num Library Directories Found", "count", fmt.Sprintf("lib-dirs-percentiles-%s.png", suffix))
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

func graphPercentiles(eventsByVersion map[string][]walkEvent, dataType, title, yAxisDescription, filename string) {
	if ok := dataTypes[dataType]; !ok {
		return
	}
	if len(eventsByVersion) < 0 {
		return
	}

	var versions []string
	for version := range eventsByVersion {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	i := 0
	var series []chart.Series
	for _, version := range versions {
		events := eventsByVersion[version]
		var samples []float64
		for _, ev := range events {
			switch dataType {
			case "duration":
				samples = append(samples, ev.durationNs)
			case "walked":
				samples = append(samples, ev.walkedDirs)
			case "library":
				samples = append(samples, ev.libraryDirs)
			default:
			}
		}
		if len(samples) == 0 {
			continue
		}

		percentiles := tracks.ComputePercentiles(samples, chartPercentiles)

		series = append(series, chart.ContinuousSeries{
			Name:    version,
			XValues: chartPercentiles,
			YValues: percentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      title,
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      yAxisDescription,
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
