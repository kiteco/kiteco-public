package tracks

import (
	"sort"
	"strings"

	analytics "gopkg.in/segmentio/analytics-go.v3"
)

// ParseUserID parses the UserId field of the track event
func ParseUserID(track *analytics.Track) string {
	return track.UserId
}

// ComputePercentiles buckets the samples by percentile
func ComputePercentiles(samples, percentiles []float64) []float64 {
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

// VersionToDate extracts the date portion from the client version string
func VersionToDate(version string, dropDay bool) string {
	// strip platform identifier and version iteration num from version string
	// version iterations from the same date will be grouped together
	first := strings.Index(version, ".")
	last := strings.LastIndex(version, ".")
	if first < 0 || last < 0 || first == last {
		return ""
	}
	if len(version)%2 != 0 {
		// day is double-digit
		version = strings.Replace(version[first+1:last], ".", "", -1)
	} else {
		// day is single-digit
		version = strings.Replace(version[first+1:last], ".", "0", -1)
	}

	// strip day
	if dropDay {
		version = version[:len(version)-2]
	}
	return version
}
