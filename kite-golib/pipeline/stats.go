package pipeline

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
)

// FeedStats describes how a particular feed was used during the pipeline's run.
type FeedStats = rundb.FeedStats

// runStats maintains live statistics for each feed of a pipeline.
type runStats struct {
	stats map[Feed]FeedStats
	m     sync.Mutex
}

func newRunStats() runStats {
	return runStats{
		stats: make(map[Feed]FeedStats),
	}
}

// IncrFeedIn should be called when a Feed receives data
func (r *runStats) IncrFeedIn(feed Feed) {
	r.m.Lock()
	defer r.m.Unlock()

	s := r.stats[feed]
	s.In++
	r.stats[feed] = s
}

// IncrFeedOut should be called when a Feed outputs data
func (r *runStats) IncrFeedOut(feed Feed) {
	r.m.Lock()
	defer r.m.Unlock()

	s := r.stats[feed]
	s.Out++
	r.stats[feed] = s
}

// AddFeedError should be called when a feed outputs a sampleError event
func (r *runStats) AddFeedError(feed Feed, sourceName string, sourceKey string, err sampleError) {
	r.m.Lock()
	defer r.m.Unlock()

	s := r.stats[feed]
	if s.ErrsByReason == nil {
		s.ErrsByReason = make(map[string]rundb.FeedErrors)
	}
	s.ErrsByReason[err.Reason] = s.ErrsByReason[err.Reason].AddError(sourceName, sourceKey, err)
	r.stats[feed] = s
}

// Stats returns a copy of the feed stats
func (r *runStats) Stats() map[string]FeedStats {
	r.m.Lock()
	defer r.m.Unlock()

	cpy := make(map[string]FeedStats, len(r.stats))
	for feed, s := range r.stats {
		cpy[feed.Name()] = s.DeepCopy()
	}

	return cpy
}

// AggregateStats returns an aggregation of multiple feed stats.
func AggregateStats(stats []map[string]FeedStats) map[string]FeedStats {
	agg := make(map[string]FeedStats)

	for _, s := range stats {
		for k, v := range s {
			agg[k] = agg[k].Add(v)
		}
	}

	return agg
}

func printStats(w io.Writer, stats map[string]FeedStats) {
	var feeds []string
	for f := range stats {
		feeds = append(feeds, f)
	}
	sort.Strings(feeds)

	for _, f := range feeds {
		fmt.Fprintf(w, "%s:\n", f)
		s := stats[f]

		maxVal := s.In
		if s.Out > maxVal {
			maxVal = s.Out
		}

		type reasonCount struct {
			reason string
			count  int64
		}
		var errs []reasonCount
		for r, c := range s.ErrsByReason {
			errs = append(errs, reasonCount{reason: r, count: c.Count})
			if c.Count > maxVal {
				maxVal = c.Count
			}
		}
		sort.Slice(errs, func(i, j int) bool { return errs[i].count > errs[j].count })

		// right-justify the counts when printing
		padding := len(strconv.Itoa(int(maxVal)))

		printCount := func(k string, v int64) {
			fmtStr := "    %" + strconv.Itoa(padding) + "d - %s\n"
			fmt.Fprintf(w, fmtStr, v, k)
		}

		printCount("in", s.In)
		printCount("out", s.Out)
		for _, rc := range errs {
			printCount(fmt.Sprintf("error (%s)", rc.reason), rc.count)
		}

		fmt.Fprint(w, "\n")
	}
}
