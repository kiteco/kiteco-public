package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/status"
	"github.com/kiteco/kiteco/kite-golib/tensorflow/bench"
)

type memRecorder struct {
	t *time.Ticker
}

func newMemRecorder(sampler *status.SampleInt64) *memRecorder {
	t := time.NewTicker(time.Second)

	go func() {
		for range t.C {
			var s runtime.MemStats
			runtime.ReadMemStats(&s)
			sampler.Record(int64(s.Alloc))
		}
	}()

	return &memRecorder{
		t: t,
	}
}

func (m *memRecorder) Stop() {
	m.t.Stop()
}

func printSampleDurations(name string, ps []float64, vs []int64) {
	fmt.Println()
	fmt.Println(name)
	fmt.Println("=============================")

	for i, p := range ps {
		fmt.Printf("Percentile: %.2f  %v\n", p, time.Duration(vs[i]))
	}
}

func printSampleInt64s(name string, ps []float64, vs []int64) {
	fmt.Println()
	fmt.Println(name)
	fmt.Println("=============================")

	for i, p := range ps {
		fmt.Printf("Percentile: %.2f  %d\n", p, vs[i])
	}
}

// feedWriter is used to write the raw tensorflow feeds/fetches to JSON so they can be replayed later
type feedWriter struct {
	Filename string
	Count    int

	m sync.Mutex

	recs []bench.FeedRecord
}

func (m *feedWriter) Callback(feeds map[string]interface{}, fetches []string, res map[string]interface{}, err error) {
	m.m.Lock()
	defer m.m.Unlock()

	if len(m.recs) >= m.Count {
		return
	}

	if err != nil {
		log.Printf("Tensorflow model returned error: %v", err)
		return
	}

	rec := bench.FeedRecord{
		Feeds:   feeds,
		Fetches: fetches,
	}

	m.recs = append(m.recs, rec)

	if len(m.recs)%20 == 0 {
		log.Printf("num feed records: %d", len(m.recs))
	}

	if len(m.recs) == m.Count {
		if err := bench.SaveFeedRecords(m.Filename, m.recs); err != nil {
			log.Printf("error writing feed records: %v", err)
			return
		}
		log.Printf("wrote feed records to %s", m.Filename)
	}
}
