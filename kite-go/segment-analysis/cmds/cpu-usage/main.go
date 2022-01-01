package main

import (
	"flag"
	"fmt"
	"log"
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

type cpuSample struct {
	uid uidType
	val float64
	ts  time.Time
}

type byPercentage []cpuSample

func (p byPercentage) Len() int           { return len(p) }
func (p byPercentage) Less(i, j int) bool { return p[i].val < p[j].val }
func (p byPercentage) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

var (
	numSamplesCutoff = 1000
	platforms        = map[string]bool{
		"darwin":  true,
		"windows": true,
		"linux":   true,
	}

	// maps of platform -> client version -> samples
	userSamples        = make(map[string]map[string][]cpuSample)
	totalSamples       = make(map[string]map[string][]cpuSample)
	totalActiveSamples = make(map[string]map[string][]cpuSample)
)

type uidType interface{}

type tracked struct {
	OS                       string  `json:"os"`
	ClientVersion            string  `json:"client_version"`
	SentAt                   float64 `json:"sent_at"`
	CPUSamples               string  `json:"cpu_samples"`
	ActiveCPUSamples         string  `json:"active_cpu_samples"`
	PythonEvent              int     `json:"python_edit"`
	UserID                   uidType `json:"user_id"`
	UseIDCCForOldCompletions bool    `json:"use_idcc_for_old_completions"`
}

func main() {
	var days, minPercentile int
	var userID, filterPlatform, versions, minVersion string
	var groupByMonth, all, newActive, verbose bool
	flag.IntVar(&days, "days", 3, "days of events to receive")
	flag.IntVar(&minPercentile, "minPercentile", 0, "min percentile to graph")
	flag.StringVar(&userID, "uid", "0", "user id")
	flag.StringVar(&filterPlatform, "platform", "", "platform (windows, darwin, or linux)")
	flag.StringVar(&versions, "versions", "", "client versions")
	flag.StringVar(&minVersion, "minVersion", "", "minimum client version (YYYYMMDD or YYMM)")
	flag.BoolVar(&groupByMonth, "groupByMonth", false, "group client version by month")
	flag.BoolVar(&all, "all", false, "use all events instead of only active events")
	flag.BoolVar(&newActive, "newActive", false, "use ActiveCPUSamples instead of PythonEvents")
	flag.BoolVar(&verbose, "verbose", false, "print verbose debug info")
	flag.Parse()

	if _, exists := platforms[filterPlatform]; filterPlatform != "" && !exists {
		log.Fatalln("Invalid platform")
	}

	// init
	clientVersions := make(map[string]bool)
	for _, v := range strings.Split(versions, ",") {
		if v == "" {
			continue
		}
		clientVersions[v] = true
	}

	for platform := range platforms {
		userSamples[platform] = make(map[string][]cpuSample)
		totalSamples[platform] = make(map[string][]cpuSample)
		totalActiveSamples[platform] = make(map[string][]cpuSample)
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
		if _, exists := platforms[platform]; !exists {
			return true
		}
		if filterPlatform != "" && platform != filterPlatform {
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

		cpuSampleStr := track.CPUSamples
		if cpuSampleStr == "" {
			return true
		}
		cpuSamples := strings.Split(cpuSampleStr, ",")
		for i := range cpuSamples {
			sample, err := strconv.ParseFloat(cpuSamples[i], 64)
			if err != nil {
				log.Println(err)
				return true
			}
			minuteDiff := len(cpuSamples) - 1 - i // samples are taken every minute ending at ts
			sampleTs := ts.Add(time.Duration(-minuteDiff) * time.Minute)
			truncatedTs := sampleTs.Truncate(time.Minute)

			if track.UserID == userID {
				userSamples[platform][clientVersion] = append(userSamples[platform][clientVersion], cpuSample{
					uid: userID,
					val: sample,
					ts:  truncatedTs,
				})
			}

			if all {
				totalSamples[platform][clientVersion] = append(totalSamples[platform][clientVersion], cpuSample{
					uid: track.UserID,
					val: sample,
				})
			} else if !newActive {
				// define active as having any python events
				if track.PythonEvent > 0 {
					totalActiveSamples[platform][clientVersion] = append(totalActiveSamples[platform][clientVersion], cpuSample{
						uid: track.UserID,
						val: sample,
					})
				}
			}
		}

		if !all && newActive {
			// define active as the samples in active_cpu_samples
			cpuActiveSampleStr := track.ActiveCPUSamples
			if cpuActiveSampleStr == "" {
				return true
			}
			cpuActiveSamples := strings.Split(cpuActiveSampleStr, ",")
			for i := range cpuActiveSamples {
				sample, err := strconv.ParseFloat(cpuActiveSamples[i], 64)
				if err != nil {
					log.Println(err)
					return true
				}
				totalActiveSamples[platform][clientVersion] = append(totalActiveSamples[platform][clientVersion], cpuSample{
					uid: track.UserID,
					val: sample,
				})
			}
		}

		return true
	})

	if verbose {
		log.Println("Highest cpu usage:")
		for platform := range platforms {
			for clientVersion, samples := range totalSamples[platform] {
				log.Printf("%s, %s:\n", platform, clientVersion)
				sort.Sort(sort.Reverse(byPercentage(samples)))
				for i, sample := range samples {
					if i == 10 {
						break
					}
					log.Printf("\t%s: %f, %s\n", sample.uid, sample.val, sample.ts)
				}
			}
		}
		log.Println("Highest active cpu usage:")
		for platform := range platforms {
			for clientVersion, samples := range totalActiveSamples[platform] {
				log.Printf("%s, %s:\n", platform, clientVersion)
				sort.Sort(sort.Reverse(byPercentage(samples)))
				for i, sample := range samples {
					if i == 10 {
						break
					}
					log.Printf("\t%s: %f\n", sample.uid, sample.val)
				}
			}
		}
	}

	ts := time.Now()
	year, month, day := ts.Date()
	for platform := range platforms {
		suffix := fmt.Sprintf("%s-%d-%d-%d", platform, int(month), day, year)
		if userID != "" {
			graphCPUUsage(userSamples[platform], userID, suffix)
		}
		if all {
			graphCPUPercentiles(totalSamples[platform], suffix, minPercentile)
		} else {
			suffix = fmt.Sprintf("%s-%s", "active", suffix)
			graphCPUPercentiles(totalActiveSamples[platform], suffix, minPercentile)
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

func graphCPUPercentiles(cpuSamples map[string][]cpuSample, suffix string, index int) {
	if len(cpuSamples) == 0 {
		return
	}

	var versions []string
	for version := range cpuSamples {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	i := 0
	var series []chart.Series
	for _, clientVersion := range versions {
		values := cpuSamples[clientVersion]
		var samples []float64
		for _, sample := range values {
			samples = append(samples, sample.val)
		}
		if len(values) < numSamplesCutoff {
			continue
		}
		cpuPercentiles := tracks.ComputePercentiles(samples, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    fmt.Sprintf("%s (n=%d)", clientVersion, len(values)),
			XValues: chartPercentiles[index:],
			YValues: cpuPercentiles[index:],
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      "CPU Usage Percentiles",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Usage (%)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	f, err := os.Create(fmt.Sprintf("cpu-percentiles-%s.png", suffix))
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphCPUUsage(cpuSamples map[string][]cpuSample, uid, suffix string) {
	if len(cpuSamples) == 0 {
		return
	}

	for clientVersion, values := range cpuSamples {
		var samples []float64
		var timestamps []time.Time
		for _, sample := range values {
			samples = append(samples, sample.val)
			timestamps = append(timestamps, sample.ts)
		}
		series := []chart.Series{
			chart.TimeSeries{
				Name:    "kite cpu usage",
				XValues: timestamps,
				YValues: samples,
				Style: chart.Style{
					Show:        true,
					StrokeColor: chart.ColorRed,
				},
			},
		}

		graph := chart.Chart{
			Title:      fmt.Sprintf("CPU Usage for %s, %s", uid, clientVersion),
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

		f, err := os.Create(fmt.Sprintf("cpu-usage-%s-%s-%s.png", uid, clientVersion, suffix))
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
