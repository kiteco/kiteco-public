package pipeline

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"
)

func (s *server) HandleRoot(w http.ResponseWriter, r *http.Request) {
	status := s.runStatus()

	var runTime time.Duration
	if status.StartedAt != (time.Time{}) {
		if status.FinishedAt != (time.Time{}) {
			runTime = status.FinishedAt.Sub(status.StartedAt)
		} else {
			runTime = time.Now().UTC().Sub(status.StartedAt)
		}
	}

	var feedStatsErr error
	feedStats, feedStatsErr := s.feedStats()
	if feedStatsErr != nil {
		log.Printf("error getting feed stats: %v", feedStatsErr)
	}

	type renderStatErr struct {
		Reason string
		Count  int64
	}

	type renderStat struct {
		Feed   string
		In     int64
		Out    int64
		Errors []renderStatErr

		Rows int // number of rows the entry spans in the view
	}

	stats := make([]renderStat, 0, len(feedStats))
	for feed, stat := range feedStats {
		errs := make([]renderStatErr, 0, len(stat.ErrsByReason))
		for reason, ebr := range stat.ErrsByReason {
			errs = append(errs, renderStatErr{Reason: reason, Count: ebr.Count})
		}
		sort.Slice(errs, func(i, j int) bool { return errs[i].Count > errs[j].Count })
		stats = append(stats, renderStat{
			Feed:   feed,
			In:     stat.In,
			Out:    stat.Out,
			Errors: errs,
			Rows:   len(errs) + 2,
		})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Feed < stats[j].Feed })

	type renderParam struct {
		Name  string
		Value interface{}
	}
	params := make([]renderParam, 0, len(s.pipe.Params))
	for k, v := range s.pipe.Params {
		params = append(params, renderParam{Name: k, Value: v})
	}
	sort.Slice(params, func(i, j int) bool { return params[i].Name < params[j].Name })

	err := s.templates.Render(w, "root.html", map[string]interface{}{
		"PipeName":     s.pipe.Name,
		"Role":         s.opts.Role,
		"StartedAt":    status.StartedAt,
		"FinishedAt":   status.FinishedAt,
		"FeedStats":    stats,
		"Params":       params,
		"Status":       status,
		"RunTime":      runTime,
		"FeedStatsErr": feedStatsErr,
	})
	if err != nil {
		s.internalError(w, err)
		return
	}
}

func (s *server) HandleFeedErrors(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	feedName := params.Get("feed")
	reason := params.Get("reason")

	if feedName == "" || reason == "" {
		s.badRequest(w, fmt.Errorf("need 'feed' and 'reason' params"))
		return
	}

	feedStats, err := s.feedStats()
	if err != nil {
		s.internalError(w, err)
		return
	}

	fs, ok := feedStats[feedName]
	if !ok {
		s.badRequest(w, fmt.Errorf("no stats for feed: %s", feedName))
		return
	}

	errs, ok := fs.ErrsByReason[reason]
	if !ok {
		s.badRequest(w, fmt.Errorf("no stats for feed: %s and reason: %s", feedName, reason))
		return
	}
	sort.Slice(errs.Samples, func(i, j int) bool { return errs.Samples[i].Timestamp.After(errs.Samples[j].Timestamp) })

	err = s.templates.Render(w, "feed-errors.html", map[string]interface{}{
		"PipeName":    s.pipe.Name,
		"Feed":        feedName,
		"Reason":      reason,
		"Errors":      errs,
		"SampleCount": len(errs.Samples),
	})
	if err != nil {
		s.internalError(w, err)
		return
	}
}

func (s *server) badRequest(w http.ResponseWriter, err error) {
	err = fmt.Errorf("bad request: %v", err)
	log.Printf("%v", err)
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func (s *server) internalError(w http.ResponseWriter, err error) {
	err = fmt.Errorf("internal error: %v", err)
	log.Printf("%v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
