package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
	chart "github.com/wcharczuk/go-chart"
)

var (
	numSamplesCutoff = 1000
	numUsersCutoff   = 100
	platforms        = map[string]bool{
		"darwin":  true,
		"windows": true,
		"linux":   false, // not implemented yet
	}
)

type memorySample struct {
	val float64
	ts  time.Time
}

type uidType interface{}

type tracked struct {
	OS            string  `json:"os"`
	ClientVersion string  `json:"client_version"`
	SentAt        float64 `json:"sent_at"`
	MemoryUsage   float64 `json:"memory_usage"`
	PythonEvent   int     `json:"python_edit"`
	UserID        uidType `json:"user_id"`
}

func main() {
	var days int
	var userID, filterPlatform, versions, minVersion string
	var groupByMonth, split bool
	flag.IntVar(&days, "days", 3, "days of events to receive")
	flag.StringVar(&userID, "uid", "", "user id")
	flag.StringVar(&filterPlatform, "platform", "", "platform (windows, darwin, or linux)")
	flag.StringVar(&versions, "versions", "", "client versions")
	flag.StringVar(&minVersion, "minVersion", "", "minimum client version (YYYYMMDD or YYMM)")
	flag.BoolVar(&groupByMonth, "groupByMonth", false, "group client version by month")
	flag.BoolVar(&split, "split", false, "split memory usage into active and inactive")
	flag.Parse()

	if supported, exists := platforms[filterPlatform]; filterPlatform != "" {
		if !exists {
			log.Fatalf("Invalid platform %s", filterPlatform)
		}
		if !supported {
			log.Fatalf("Memory usage not implemented for %s\n", filterPlatform)
		}
	}

	// init
	clientVersions := make(map[string]bool)
	for _, v := range strings.Split(versions, ",") {
		if v == "" {
			continue
		}
		clientVersions[v] = true
	}

	userActiveSamples := make(map[string]map[string]map[uidType][]memorySample)
	userInactiveSamples := make(map[string]map[string]map[uidType][]memorySample)
	userSamples := make(map[string]map[string]map[uidType][]memorySample)
	for platform, supported := range platforms {
		if !supported {
			continue
		}
		userActiveSamples[platform] = make(map[string]map[uidType][]memorySample)
		userInactiveSamples[platform] = make(map[string]map[uidType][]memorySample)
		userSamples[platform] = make(map[string]map[uidType][]memorySample)
	}

	end := analyze.Today()
	start := end.Add(0, 0, -days)
	listing, err := analyze.ListRange(segmentsrc.Production, start, end)
	if err != nil {
		log.Fatalln(err)
	}

	var uris []string
	for _, day := range listing.Dates {
		uris = append(uris, day.URIs...)
	}

	analyze.Analyze(uris, 4, "kite_status", func(meta analyze.Metadata, track *tracked) bool {
		if meta.EventName != "kite_status" {
			return true
		}
		platform := track.OS
		if supported, exists := platforms[platform]; !supported || !exists {
			return true
		}
		if filterPlatform != "" && track.OS != filterPlatform {
			return true
		}
		clientVersion := tracks.VersionToDate(track.ClientVersion, groupByMonth)
		if clientVersion == "" {
			return true
		}
		if _, ok := clientVersions[clientVersion]; len(clientVersions) > 0 && !ok {
			return true
		}
		if minVersion != "" && clientVersion < minVersion {
			return true
		}

		sentAt := int64(track.SentAt)
		ts := time.Unix(sentAt, 0)

		memoryUsage := track.MemoryUsage

		if split {
			if userActiveSamples[platform][clientVersion] == nil {
				userActiveSamples[platform][clientVersion] = make(map[uidType][]memorySample)
			}
			if userInactiveSamples[platform][clientVersion] == nil {
				userInactiveSamples[platform][clientVersion] = make(map[uidType][]memorySample)
			}
		} else {
			if userSamples[platform][clientVersion] == nil {
				userSamples[platform][clientVersion] = make(map[uidType][]memorySample)
			}
		}

		if split {
			if track.PythonEvent > 0 {
				userActiveSamples[platform][clientVersion][track.UserID] = append(userActiveSamples[platform][clientVersion][track.UserID], memorySample{
					val: memoryUsage / (1024 * 1024), // convert bytes to Mb
					ts:  ts,
				})
			} else {
				userInactiveSamples[platform][clientVersion][track.UserID] = append(userInactiveSamples[platform][clientVersion][track.UserID], memorySample{
					val: memoryUsage / (1024 * 1024), // convert bytes to Mb
					ts:  ts,
				})
			}
		} else {
			userSamples[platform][clientVersion][track.UserID] = append(userSamples[platform][clientVersion][track.UserID], memorySample{
				val: memoryUsage / (1024 * 1024), // convert bytes to Mb
				ts:  ts,
			})
		}

		return true
	})

	ts := time.Now()
	year, month, day := ts.Date()
	for platform, supported := range platforms {
		if !supported {
			continue
		}
		suffix := fmt.Sprintf("%s-%d-%d-%d", platform, int(month), day, year)
		if split {
			activeSuffix := fmt.Sprintf("%s-%s", "active", suffix)
			graphMemoryPercentiles(userActiveSamples[platform], activeSuffix)
			graphPercentilesTimeseries(userActiveSamples[platform], activeSuffix)
			graphMemoryVariancePercentiles(userActiveSamples[platform], activeSuffix)
			if userID != "" {
				graphUserMemoryUsage(userActiveSamples[platform], userID, activeSuffix)
			}
			inactiveSuffix := fmt.Sprintf("%s-%s", "inactive", suffix)
			graphMemoryPercentiles(userInactiveSamples[platform], inactiveSuffix)
			graphPercentilesTimeseries(userInactiveSamples[platform], inactiveSuffix)
			graphMemoryVariancePercentiles(userInactiveSamples[platform], inactiveSuffix)
			if userID != "" {
				graphUserMemoryUsage(userInactiveSamples[platform], userID, inactiveSuffix)
			}
		} else {
			graphMemoryPercentiles(userSamples[platform], suffix)
			graphPercentilesTimeseries(userSamples[platform], suffix)
			graphMemoryVariancePercentiles(userSamples[platform], suffix)
			if userID != "" {
				graphUserMemoryUsage(userSamples[platform], userID, suffix)
			}
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

func computeVariance(samples []memorySample) float64 {
	var sum float64
	for _, s := range samples {
		sum += s.val
	}
	avg := sum / float64(len(samples))

	var sqSum float64
	for _, s := range samples {
		sqSum += math.Pow(avg-s.val, 2)
	}
	return sqSum / float64(len(samples))
}

func graphMemoryPercentiles(memorySamples map[string]map[uidType][]memorySample, suffix string) {
	if len(memorySamples) == 0 {
		return
	}

	var versions []string
	for version := range memorySamples {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	i := 0
	var series []chart.Series
	for _, clientVersion := range versions {
		users := memorySamples[clientVersion]
		var samples []float64
		for _, userSamples := range users {
			for _, sample := range userSamples {
				samples = append(samples, sample.val)
			}
		}
		if len(samples) < numSamplesCutoff {
			continue
		}
		memoryPercentiles := tracks.ComputePercentiles(samples, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    fmt.Sprintf("%s (n=%d)", clientVersion, len(users)),
			XValues: chartPercentiles,
			YValues: memoryPercentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      "Memory Usage Percentiles",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Usage (Mb)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	f, err := os.Create(fmt.Sprintf("memory-percentiles-%s.png", suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphPercentilesTimeseries(memorySamples map[string]map[uidType][]memorySample, suffix string) {
	if len(memorySamples) == 0 {
		return
	}

	var versions []string
	for version := range memorySamples {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	var fiftieth, ninetieth []float64
	var versionFloats []float64
	var ticks []chart.Tick
	for _, clientVersion := range versions {
		users := memorySamples[clientVersion]
		var samples []float64
		for _, userSamples := range users {
			for _, sample := range userSamples {
				samples = append(samples, sample.val)
			}
		}
		if len(samples) < numSamplesCutoff {
			continue
		}
		memoryPercentiles := tracks.ComputePercentiles(samples, chartPercentiles)
		versionInt, err := strconv.Atoi(clientVersion)
		if err != nil {
			continue
		}
		versionFloats = append(versionFloats, float64(versionInt))
		ticks = append(ticks, chart.Tick{Value: float64(versionInt), Label: clientVersion})
		fiftieth = append(fiftieth, memoryPercentiles[50])
		ninetieth = append(ninetieth, memoryPercentiles[90])
	}

	series := []chart.Series{
		chart.ContinuousSeries{
			Name:    "50th percentile",
			XValues: versionFloats,
			YValues: fiftieth,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.ColorBlue,
			},
		},
		chart.ContinuousSeries{
			Name:    "90th percentile",
			XValues: versionFloats,
			YValues: ninetieth,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.ColorRed,
			},
		},
	}

	graph := chart.Chart{
		Title:      "Memory Usage",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Version",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			Ticks:     ticks,
		},
		YAxis: chart.YAxis{
			Name:      "Usage (Mb)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	f, err := os.Create(fmt.Sprintf("memory-percentiles-by-version-%s.png", suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphMemoryVariancePercentiles(memorySamples map[string]map[uidType][]memorySample, suffix string) {
	if len(memorySamples) == 0 {
		return
	}

	var versions []string
	for version := range memorySamples {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	i := 0
	var series []chart.Series
	for _, clientVersion := range versions {
		users := memorySamples[clientVersion]
		var variances []float64
		for _, userSamples := range users {
			if len(userSamples) == 0 {
				continue
			}
			variances = append(variances, computeVariance(userSamples))
		}
		if len(variances) < numUsersCutoff {
			continue
		}
		variancePercentiles := tracks.ComputePercentiles(variances, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    fmt.Sprintf("%s (n=%d)", clientVersion, len(users)),
			XValues: chartPercentiles,
			YValues: variancePercentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      "Memory Variance Percentiles",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Usage (Mb)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	f, err := os.Create(fmt.Sprintf("memory-variance-percentiles-%s.png", suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphUserMemoryUsage(memSamples map[string]map[uidType][]memorySample, uid, suffix string) {
	if len(memSamples) == 0 {
		return
	}

	i := 0
	var series []chart.Series
	for clientVersion, userSamples := range memSamples {
		var cutoff float64
		samples := userSamples[uid]
		if len(samples) == 0 {
			continue
		}
		var values []float64
		for _, s := range samples {
			values = append(values, s.val)
		}
		if len(values) == 0 {
			continue
		}
		memoryPercentiles := tracks.ComputePercentiles(values, chartPercentiles)
		if len(memoryPercentiles) < 95 {
			continue
		}
		cutoff = memoryPercentiles[95]

		var highSamples []float64
		var highTimestamps []time.Time
		for _, sample := range samples {
			if sample.val >= cutoff {
				highSamples = append(highSamples, sample.val)
				highTimestamps = append(highTimestamps, sample.ts)
			}
		}
		series = append(series, chart.TimeSeries{
			Name:    clientVersion,
			XValues: highTimestamps,
			YValues: highSamples,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      fmt.Sprintf("Memory Usage for %s", uid),
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
			Name: "Usage (Mb)",
			Style: chart.Style{
				Show: true,
			},
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	f, err := os.Create(fmt.Sprintf("memory-usage-%s-%s.png", uid, suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
