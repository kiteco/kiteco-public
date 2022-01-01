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
)

var (
	knownFailureTypes = []string{
		"no files selected",
		"missing file hashes",
		"requested file path not found in file listing",
		"requested file path not found in dir",
		"requested file path excluded from index",
		"requested build path not in walked root directory",
		"start path not valid file",
		"stat",
		"kitectx.Context expired: context deadline exceeded",
		"expected path to begin",
	}
	// maps from uid -> bool
	seenFailureTypes      = make(map[string]bool)
	newFailureTypes       = make(map[string]bool)
	users                 = make(map[string]bool)
	usersWithIndexAdded   = make(map[string]bool)
	usersWithParseTimeout = make(map[string]bool)
	seenReasons           = make(map[string]bool)
	// map from uid -> failure type -> bool
	usersWithIndexFailure = make(map[string]map[string]bool)
	// maps from client version -> durations
	buildDurations = make(map[string][]float64)
	parseDurations = make(map[string][]float64)
	avgCPU         = make(map[string][]float64)
	maxCPU         = make(map[string][]float64)
)

type statsByDay struct {
	day time.Time

	// counts by unique uid
	totalUsers    map[string]bool
	failedUsers   map[string]bool
	successUsers  map[string]bool
	filteredUsers map[string]bool

	// counts by event
	totalEvents          int
	failedEvents         int
	jobFailureTypes      map[string]int
	jobFailureTypesUsers map[string]map[string]bool
	totalVersion         int
	olderVersion         int // how many events have version < minVersion
	olderVersionUsers    map[string]bool
	olderVersions        map[string]int
	filteredReasonsUsers map[string]map[string]int
}

var totalStats []statsByDay

func main() {
	var days int
	var old, verbose, groupByMonth bool
	var platform, minVersion string
	flag.IntVar(&days, "days", 3, "days of events to receive")
	flag.BoolVar(&old, "old", false, "use old Client Event source")
	flag.StringVar(&platform, "platform", "darwin", "platform (windows, darwin, or linux)")
	flag.StringVar(&minVersion, "minVersion", "", "minimum client version (0.YYYYMMDD.X for darwin, 1.YYYY.MMDD.X for windows, 2.YYYYMMDD.X for linux)")
	flag.BoolVar(&verbose, "verbose", false, "print uids for users with parse timeouts and no successful index builds")
	flag.BoolVar(&groupByMonth, "groupByMonth", false, "group client version by month")
	flag.Parse()

	source := tracks.ClientEventSource
	if old {
		source = tracks.OldClientEventSource
	}
	listing, err := tracks.List(tracks.Bucket, source)
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
			totalUsers:           make(map[string]bool),
			failedUsers:          make(map[string]bool),
			successUsers:         make(map[string]bool),
			filteredUsers:        make(map[string]bool),
			jobFailureTypes:      make(map[string]int),
			jobFailureTypesUsers: make(map[string]map[string]bool),
			olderVersionUsers:    make(map[string]bool),
			olderVersions:        make(map[string]int),
			filteredReasonsUsers: make(map[string]map[string]int),
		}
		for track := range r.Tracks {
			switch track.Event {
			case "Local Index Added":
				uid := tracks.ParseUserID(track)
				usersWithIndexAdded[uid] = true
				continue
			case "Index Build Filtered":
				if track.Properties["platform"].(string) != platform {
					continue
				}
				uid := tracks.ParseUserID(track)
				reason := track.Properties["reason"].(string)
				if _, ok := currentStats.filteredReasonsUsers[reason]; !ok {
					currentStats.filteredReasonsUsers[reason] = make(map[string]int)
				}
				currentStats.filteredReasonsUsers[reason][uid]++
				seenReasons[reason] = true
				currentStats.filteredUsers[uid] = true
				continue
			case "Index Build":
				if track.Properties["platform"].(string) != platform {
					continue
				}

				uid := tracks.ParseUserID(track)
				clientVersion := track.Properties["client_version"].(string)
				currentStats.totalVersion++
				if minVersion != "" && clientVersion < minVersion {
					currentStats.olderVersion++
					currentStats.olderVersionUsers[uid] = true
					currentStats.olderVersions[clientVersion]++
					continue
				}
				if groupByMonth {
					lastDot := strings.LastIndex(clientVersion, ".")
					if lastDot-2 < 0 || len(clientVersion) <= lastDot-2 {
						continue
					}
					// strip end from version which includes day
					clientVersion = clientVersion[:lastDot-2]
				}

				currentStats.totalEvents++
				currentStats.totalUsers[uid] = true
				users[uid] = true

				// determine if job did not succeed and why
				errStr := track.Properties["error"].(string)
				var errMsg string
				if errStr != "" {
					currentStats.failedEvents++

					errMsg = errStr
					var exists bool
					for _, ft := range knownFailureTypes {
						// concatenated failure type that includes only error type, not
						// job and user-specific details
						if strings.Contains(errStr, ft) {
							errMsg = ft
							exists = true
							seenFailureTypes[ft] = true
							break
						}
					}
					if !exists {
						// track in order to update known failure types
						newFailureTypes[errMsg] = true
						errMsg = "other"
					}
					if currentStats.jobFailureTypesUsers[errMsg] == nil {
						currentStats.jobFailureTypesUsers[errMsg] = make(map[string]bool)
					}
					currentStats.jobFailureTypes[errMsg]++
					currentStats.jobFailureTypesUsers[errMsg][uid] = true
					currentStats.failedUsers[uid] = true
					if _, ok := usersWithIndexFailure[uid]; !ok {
						usersWithIndexFailure[uid] = make(map[string]bool)
					}
					usersWithIndexFailure[uid][errMsg] = true
				} else {
					currentStats.successUsers[uid] = true
				}

				// track build durations for graphing percentiles
				b := track.Properties["since_start_ns"].(float64)
				if b > 0 {
					buildDurations[clientVersion] = append(buildDurations[clientVersion], b/float64(time.Millisecond))
				}

				// track max and average cpu usage for graphing percentiles
				if track.Properties["cpu_info"] != nil {
					cpuInfo := track.Properties["cpu_info"].(map[string]interface{})
					count := cpuInfo["count"]
					sum := cpuInfo["sum"]
					if count != nil && sum != nil && count.(float64) != 0 {
						avgCPU[clientVersion] = append(avgCPU[clientVersion], sum.(float64)/count.(float64))
					}
					max := cpuInfo["max"]
					if max != nil {
						maxCPU[clientVersion] = append(maxCPU[clientVersion], max.(float64))
					}
				}

				if track.Properties["parse_info"] == nil {
					continue
				}
				parseInfo := track.Properties["parse_info"].(map[string]interface{})
				parseTimeouts := parseInfo["parse_failures"].(float64)
				if parseTimeouts > 0 {
					usersWithParseTimeout[uid] = true
				}
				parses := parseInfo["parse_durations"]
				if parses == nil {
					continue
				}
				for _, d := range parses.([]interface{}) {
					parseDurations[clientVersion] = append(parseDurations[clientVersion], d.(float64)/float64(time.Millisecond))
				}

			default:
				continue

			}
		}
		// update stats for given day
		totalStats = append(totalStats, currentStats)
	}

	log.Printf("%s REPORTS", strings.ToUpper(platform))
	for _, stats := range totalStats {
		// generate daily report
		log.Println("building report for ", stats.day.Format("2006-01-02"))

		log.Printf("Total Events: %d", stats.totalEvents)
		log.Printf("Failed Events: %d", stats.failedEvents)
		log.Printf("Total Users: %d", len(stats.totalUsers))
		log.Printf("Failed Users: %d", len(stats.failedUsers))
		log.Println("---")

		// user stats
		log.Printf("Percentage of users that have a failed job: %f", float64(len(stats.failedUsers))/float64(len(stats.totalUsers))*100)
		log.Println("---")

		// event stats
		log.Printf("Percentage of jobs that failed: %f", float64(stats.failedEvents)/float64(stats.totalEvents)*100)
		log.Println("---")

		// failure stats
		log.Printf("Failure types (%% failures, unique user count):")
		for reason, count := range stats.jobFailureTypes {
			if count > 0 {
				log.Printf("%s: %f, %d", reason, float64(count)/float64(stats.failedEvents)*100, len(stats.jobFailureTypesUsers[reason]))
			}
		}
		log.Println("---")

		// filtered stats
		total := len(stats.filteredUsers) + len(stats.totalUsers)
		log.Printf("Percentage of users with index builds filtered: %f (%d/%d)", float64(len(stats.filteredUsers))/float64(total), len(stats.filteredUsers), total)
		log.Println("---")

		// version stats
		log.Printf("Versions older than: %s", minVersion)
		for v, n := range stats.olderVersions {
			log.Printf("%s: %d", v, n)
		}
		log.Printf("\n\n\n")
	}
	log.Println("---")

	// notify of new failure types
	log.Println("New failure types:")
	for ft := range newFailureTypes {
		log.Println(ft)
	}
	log.Printf("\n\n\n")

	// verbose info
	if verbose {
		log.Println("---")
		neverAdded := make(map[string]map[string]bool)
		for uid, failures := range usersWithIndexFailure {
			if _, ok := usersWithIndexAdded[uid]; !ok {
				for failure := range failures {
					if _, ok := neverAdded[failure]; !ok {
						neverAdded[failure] = make(map[string]bool)
					}
					neverAdded[failure][uid] = true
				}
			}
		}
		log.Println("Users with no Local Index Added event:")
		for failure, users := range neverAdded {
			log.Printf("%s: %d\n", failure, len(users))
			for uid := range users {
				log.Printf("\t%s\n", uid)
			}
		}

		log.Println("---")
		log.Println("Users with parse timeout:")
		for uid := range usersWithParseTimeout {
			log.Printf("\t%s\n", uid)
		}
	}

	// create graphs with data from all days
	graphBuildDurations(platform, minVersion)
	graphParseDurations(platform, minVersion)
	graphCPUPercentiles(platform, minVersion, "max")
	graphCPUPercentiles(platform, minVersion, "avg")
	graphFailures(platform, minVersion)
	graphFailureUserCounts(platform, minVersion)
	graphPercentJobsFailed(platform, minVersion)
	graphPercentUsersFailed(platform, minVersion)
	graphPercentOlderVersion(platform, minVersion)
	graphPercentUsersNeverBuilt(platform, minVersion)
	graphNumUsersOlderVersion(platform, minVersion)
	graphFilteredReasons(platform, minVersion)
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

func graphBuildDurations(platform, minVersion string) {
	if len(buildDurations) == 0 {
		return
	}

	i := 0
	var series []chart.Series
	for clientVersion, durations := range buildDurations {
		buildPercentiles := computePercentiles(durations, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    clientVersion,
			XValues: chartPercentiles,
			YValues: buildPercentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      "Build Duration(ns) Percentiles",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Build time (ms)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	name := fmt.Sprintf("index-build-percentiles-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphParseDurations(platform, minVersion string) {
	if len(parseDurations) == 0 {
		return
	}

	i := 0
	var series []chart.Series
	for clientVersion, durations := range parseDurations {
		parsePercentiles := computePercentiles(durations, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    clientVersion,
			XValues: chartPercentiles,
			YValues: parsePercentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	graph := chart.Chart{
		Title:      "Parse Duration(ns) Percentiles",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "Parse time (ms)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	name := fmt.Sprintf("parse-durations-percentiles-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphCPUPercentiles(platform, minVersion, statistic string) {
	var cpuSamples map[string][]float64
	if statistic == "max" {
		cpuSamples = maxCPU
	} else if statistic == "avg" {
		cpuSamples = avgCPU
	} else {
		return
	}
	if len(cpuSamples) == 0 {
		return
	}

	i := 0
	var series []chart.Series
	for clientVersion, samples := range cpuSamples {
		cpuPercentiles := computePercentiles(samples, chartPercentiles)
		series = append(series, chart.ContinuousSeries{
			Name:    clientVersion,
			XValues: chartPercentiles,
			YValues: cpuPercentiles,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}

	title := fmt.Sprintf("%s CPU Usage(%%) Percentiles", strings.Title(statistic))
	graph := chart.Chart{
		Title:      title,
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "Percentiles",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Name:      "CPU usage (%)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}

	name := fmt.Sprintf("%s-cpu-usage-percentiles-%s", statistic, platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphFailures(platform, minVersion string) {
	var days []time.Time
	var total []float64
	failureCounts := make(map[string][]float64)
	for _, stats := range totalStats {
		days = append(days, stats.day)

		failed := 0
		for ft := range seenFailureTypes {
			count := stats.jobFailureTypes[ft]
			failed += count
			failureCounts[ft] = append(failureCounts[ft], float64(count))
		}
		total = append(total, float64(failed))
	}

	if len(total) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(total), len(days))
	}

	var series []chart.Series
	i := 0
	for ft, counts := range failureCounts {
		series = append(series, chart.TimeSeries{
			Name:    ft,
			XValues: days,
			YValues: counts,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}
	series = append(series, chart.TimeSeries{
		Name:    "total",
		XValues: days,
		YValues: total,
		Style: chart.Style{
			Show:            true,
			StrokeColor:     chart.ColorRed,
			StrokeDashArray: []float64{5.0, 5.0},
		},
	})

	graph := chart.Chart{
		Title:      "Failure Type Counts",
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

	name := fmt.Sprintf("failed-jobs-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphFailureUserCounts(platform, minVersion string) {
	var days []time.Time
	var total []float64
	failureCounts := make(map[string][]float64)
	for _, stats := range totalStats {
		days = append(days, stats.day)

		failed := 0
		for ft := range seenFailureTypes {
			users := stats.jobFailureTypesUsers[ft]
			failed += len(users)
			failureCounts[ft] = append(failureCounts[ft], float64(len(users)))
		}
		total = append(total, float64(failed))
	}

	if len(total) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(total), len(days))
	}

	var series []chart.Series
	i := 0
	for ft, counts := range failureCounts {
		series = append(series, chart.TimeSeries{
			Name:    ft,
			XValues: days,
			YValues: counts,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}
	series = append(series, chart.TimeSeries{
		Name:    "total",
		XValues: days,
		YValues: total,
		Style: chart.Style{
			Show:            true,
			StrokeColor:     chart.ColorRed,
			StrokeDashArray: []float64{5.0, 5.0},
		},
	})

	graph := chart.Chart{
		Title:      "User Counts for Failure Types",
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

	name := fmt.Sprintf("failed-jobs-users-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphPercentJobsFailed(platform, minVersion string) {
	var days []time.Time
	var failureRates []float64
	for _, stats := range totalStats {
		days = append(days, stats.day)

		var percentFailed float64
		if stats.totalEvents == 0 {
			percentFailed = 0
		} else {
			percentFailed = float64(stats.failedEvents) / float64(stats.totalEvents) * 100
		}
		failureRates = append(failureRates, percentFailed)
	}

	if len(failureRates) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(failureRates), len(days))
	}

	var series []chart.Series
	series = append(series, chart.TimeSeries{
		Name:    platform,
		XValues: days,
		YValues: failureRates,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorRed,
		},
	})

	graph := chart.Chart{
		Title:      "Index Build Job Failure Rate",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "percent",
			NameStyle: chart.StyleShow(),
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

	name := fmt.Sprintf("percent-build-failed-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphPercentUsersFailed(platform, minVersion string) {
	var days []time.Time
	var failureRates []float64
	for _, stats := range totalStats {
		days = append(days, stats.day)

		var percentFailed float64
		if stats.totalEvents == 0 {
			percentFailed = 0
		} else {
			percentFailed = float64(len(stats.failedUsers)) / float64(len(stats.totalUsers)) * 100
		}
		failureRates = append(failureRates, percentFailed)
	}

	if len(failureRates) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(failureRates), len(days))
	}

	var series []chart.Series
	series = append(series, chart.TimeSeries{
		Name:    platform,
		XValues: days,
		YValues: failureRates,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorRed,
		},
	})

	graph := chart.Chart{
		Title:      "% Users With Index Build Failure",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "percent",
			NameStyle: chart.StyleShow(),
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

	name := fmt.Sprintf("percent-users-failed-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphPercentOlderVersion(platform, minVersion string) {
	var days []time.Time
	var olderRates []float64
	for _, stats := range totalStats {
		days = append(days, stats.day)

		var percentOlder float64
		if stats.totalVersion == 0 {
			percentOlder = 0
		} else {
			percentOlder = float64(stats.olderVersion) / float64(stats.totalVersion) * 100
		}
		olderRates = append(olderRates, percentOlder)
	}

	if len(olderRates) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(olderRates), len(days))
	}

	var series []chart.Series
	series = append(series, chart.TimeSeries{
		Name:    platform,
		XValues: days,
		YValues: olderRates,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorRed,
		},
	})

	graph := chart.Chart{
		Title:      fmt.Sprintf("%% Events Older Than Version %s", minVersion),
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "percent",
			NameStyle: chart.StyleShow(),
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

	name := fmt.Sprintf("percent-older-version-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphNumUsersOlderVersion(platform, minVersion string) {
	var days []time.Time
	var olderCounts []float64
	for _, stats := range totalStats {
		days = append(days, stats.day)

		olderCounts = append(olderCounts, float64(len(stats.olderVersionUsers)))
	}

	if len(olderCounts) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(olderCounts), len(days))
	}

	var series []chart.Series
	series = append(series, chart.TimeSeries{
		Name:    platform,
		XValues: days,
		YValues: olderCounts,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorRed,
		},
	})

	graph := chart.Chart{
		Title:      fmt.Sprintf("Num Users with Version Older than %s", minVersion),
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "count",
			NameStyle: chart.StyleShow(),
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

	name := fmt.Sprintf("num-older-version-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphPercentUsersNeverBuilt(platform, minVersion string) {
	var days []time.Time
	var neverBuiltRates []float64
	for _, stats := range totalStats {
		days = append(days, stats.day)

		// find users who never had an index built
		neverBuilt := make(map[string]bool)
		for uid := range stats.failedUsers {
			success := stats.successUsers[uid]
			if !success {
				neverBuilt[uid] = true
			}
		}

		var percentNeverBuilt float64
		if stats.totalEvents == 0 {
			percentNeverBuilt = 0
		} else {
			percentNeverBuilt = float64(len(neverBuilt)) / float64(len(stats.totalUsers)) * 100
		}
		neverBuiltRates = append(neverBuiltRates, percentNeverBuilt)
	}

	if len(neverBuiltRates) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(neverBuiltRates), len(days))
	}

	var series []chart.Series
	series = append(series, chart.TimeSeries{
		Name:    platform,
		XValues: days,
		YValues: neverBuiltRates,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorRed,
		},
	})

	graph := chart.Chart{
		Title:      "% Users With No Index Build Success",
		TitleStyle: chart.StyleShow(),
		XAxis: chart.XAxis{
			Name:      "percent",
			NameStyle: chart.StyleShow(),
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

	name := fmt.Sprintf("percent-users-never-built-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func graphFilteredReasons(platform, minVersion string) {
	var days []time.Time
	var total []float64
	filteredCounts := make(map[string][]float64)
	for _, stats := range totalStats {
		days = append(days, stats.day)

		filtered := 0
		for r := range seenReasons {
			count := len(stats.filteredReasonsUsers[r])
			filtered += count
			filteredCounts[r] = append(filteredCounts[r], float64(count))
		}
		total = append(total, float64(filtered))
	}

	if len(total) != len(days) {
		log.Fatalf("Expected number of values to match: %d, %d", len(total), len(days))
	}

	var series []chart.Series
	i := 0
	for ft, counts := range filteredCounts {
		series = append(series, chart.TimeSeries{
			Name:    ft,
			XValues: days,
			YValues: counts,
			Style: chart.Style{
				Show:        true,
				StrokeColor: chart.GetAlternateColor(i),
			},
		})
		i++
	}
	series = append(series, chart.TimeSeries{
		Name:    "total",
		XValues: days,
		YValues: total,
		Style: chart.Style{
			Show:            true,
			StrokeColor:     chart.ColorRed,
			StrokeDashArray: []float64{5.0, 5.0},
		},
	})

	graph := chart.Chart{
		Title:      "Filtered Reason Counts",
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

	name := fmt.Sprintf("filtered-reasons-%s", platform)
	if minVersion != "" {
		name = fmt.Sprintf("%s-%s", name, minVersion)
	}
	f, err := os.Create(name + ".png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
