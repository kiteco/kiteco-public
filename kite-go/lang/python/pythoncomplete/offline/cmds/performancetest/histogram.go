package main

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"
)

// this is based on Golang's src/cmd/trace/annotations.go implementation

type durationHistogram struct {
	Count                int
	Buckets              []int
	MinBucket, MaxBucket int
}

// Five buckets for every power of 10.
var logDiv = math.Log(math.Pow(10, 1.0/5))

func (h *durationHistogram) add(d time.Duration) {
	var bucket int
	if d > 0 {
		bucket = int(math.Log(float64(d)) / logDiv)
	}
	if len(h.Buckets) <= bucket {
		h.Buckets = append(h.Buckets, make([]int, bucket-len(h.Buckets)+1)...)
		h.Buckets = h.Buckets[:cap(h.Buckets)]
	}
	h.Buckets[bucket]++
	if bucket < h.MinBucket || h.MaxBucket == 0 {
		h.MinBucket = bucket
	}
	if bucket > h.MaxBucket {
		h.MaxBucket = bucket
	}
	h.Count++
}

func (h *durationHistogram) BucketMin(bucket int) time.Duration {
	return time.Duration(math.Exp(float64(bucket) * logDiv))
}

func niceDuration(d time.Duration) string {
	var rnd time.Duration
	var unit string
	switch {
	case d < 10*time.Microsecond:
		rnd, unit = time.Nanosecond, "ns"
	case d < 10*time.Millisecond:
		rnd, unit = time.Microsecond, "µs"
	case d < 10*time.Second:
		rnd, unit = time.Millisecond, "ms"
	default:
		rnd, unit = time.Second, "s "
	}
	return fmt.Sprintf("%d%s", d/rnd, unit)
}

func (h *durationHistogram) ToHTML(urlmaker func(min, max time.Duration) string) template.HTML {
	if h == nil || h.Count == 0 {
		return template.HTML("")
	}

	const barWidth = 400

	maxCount := 0
	for _, count := range h.Buckets {
		if count > maxCount {
			maxCount = count
		}
	}

	w := new(bytes.Buffer)
	fmt.Fprintf(w, `<table>`)
	for i := h.MinBucket; i <= h.MaxBucket; i++ {
		// Tick label.
		if h.Buckets[i] > 0 {
			fmt.Fprintf(w, `<tr><td class="histoTime" align="right"><a href=%s>%s</a></td>`, urlmaker(h.BucketMin(i), h.BucketMin(i+1)), niceDuration(h.BucketMin(i)))
		} else {
			fmt.Fprintf(w, `<tr><td class="histoTime" align="right">%s</td>`, niceDuration(h.BucketMin(i)))
		}
		// Bucket bar.
		width := h.Buckets[i] * barWidth / maxCount
		fmt.Fprintf(w, `<td><div style="width:%dpx;background:blue;position:relative">&nbsp;</div></td>`, width)
		// Bucket count.
		fmt.Fprintf(w, `<td align="right"><div style="position:relative">%d</div></td>`, h.Buckets[i])
		fmt.Fprintf(w, "</tr>\n")

	}
	// Final tick label.
	fmt.Fprintf(w, `<tr><td align="right">%s</td></tr>`, niceDuration(h.BucketMin(h.MaxBucket+1)))
	fmt.Fprintf(w, `</table>`)
	return template.HTML(w.String())
}

func (h *durationHistogram) String() string {
	if h.Count == 0 {
		return ""
	}

	const barWidth = 40

	labels := []string{}
	maxLabel := 0
	maxCount := 0
	for i := h.MinBucket; i <= h.MaxBucket; i++ {
		// TODO: This formatting is pretty awful.
		label := fmt.Sprintf("[%-8s%-8s)", niceDuration(h.BucketMin(i))+",", niceDuration(h.BucketMin(i+1)))
		labels = append(labels, label)
		if len(label) > maxLabel {
			maxLabel = len(label)
		}
		count := h.Buckets[i]
		if count > maxCount {
			maxCount = count
		}
	}

	w := new(bytes.Buffer)
	for i := h.MinBucket; i <= h.MaxBucket; i++ {
		count := h.Buckets[i]
		bar := count * barWidth / maxCount
		countLabel := ""
		if count > 0 {
			countLabel = fmt.Sprintf("%d", count)
		}
		fmt.Fprintf(w, "%*s %-*s %s\n", maxLabel, labels[i-h.MinBucket], barWidth, strings.Repeat("█", bar), countLabel)
	}
	return w.String()
}
