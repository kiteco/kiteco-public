// +build darwin linux

package benchmark

import (
	"fmt"
	"io"
	"math/rand"
	"runtime"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// Stats ...
type Stats struct {
	Task      string
	TotalCPU  float64
	Durations []time.Duration
	Count     int
}

// Median durations ...
func (s Stats) Median() time.Duration {
	if len(s.Durations) == 0 {
		return time.Duration(0)
	}

	sort.Slice(s.Durations, func(i, j int) bool {
		return s.Durations[i] < s.Durations[j]
	})

	middle := len(s.Durations) / 2
	if len(s.Durations) > 0 && len(s.Durations)%2 == 0 {
		return (s.Durations[middle-1] + s.Durations[middle]) / 2
	}
	return s.Durations[middle]
}

// Avg durations ...
func (s Stats) Avg() time.Duration {
	var total time.Duration
	for _, d := range s.Durations {
		total += d
	}
	return time.Duration(float64(total) / float64(len(s.Durations)))
}

// Print ...
func (s Stats) Print(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Task: %s\n", s.Task)
	tabw := tabwriter.NewWriter(w, 16, 4, 4, ' ', 0)
	defer tabw.Flush()
	fmt.Fprintf(tabw, "avg latency (ms)\tmedian latency (ms)\tnum cores\tavg cpu utilization\t\n")
	fmt.Fprintf(tabw, "%.3f\t%.3f\t%v\t%v\n",
		float64(s.Avg())/1e6, float64(s.Median())/1e6,
		runtime.NumCPU(), s.TotalCPU/float64(s.Count),
	)
}

// StepFn ...
type StepFn func(predictor lexicalmodels.ModelBase, in predict.Inputs)

// InitFn ...
type InitFn func(predictor lexicalmodels.ModelBase, in predict.Inputs)

// Benchmarker ...
type Benchmarker struct {
	Iters     int
	Predictor predict.Predictor
	Search    predict.SearchConfig
	Buf       []byte
	Verbose   bool
}

// MustBenchmark ...
func (b Benchmarker) MustBenchmark(task string, init InitFn, step StepFn) Stats {
	stats, err := b.Benchmark(task, init, step)
	if err != nil {
		panic(err)
	}
	return stats
}

// Benchmark ...
func (b Benchmarker) Benchmark(task string, init InitFn, step StepFn) (Stats, error) {
	tokens, err := b.Predictor.GetEncoder().Lexer.Lex(b.Buf)
	if err != nil {
		return Stats{}, errors.Wrapf(err, "encoding error")
	}

	var count int
	var totalCPU float64
	var durations []time.Duration
	for count < b.Iters {
		in := predict.Inputs{
			Tokens:         tokens,
			CursorTokenIdx: rand.Intn(len(tokens)-1) + 1,
		}

		if init != nil {
			init(b.Predictor, in)
		}

		start := time.Now()
		step(b.Predictor, in)
		since := time.Since(start)
		usage := cpuUsage()

		// Throw away first result because model needs to warm up
		if count == 0 {
			if b.Verbose {
				fmt.Printf("%s iter %d: took: %s, cpu usage: %.02f (NOT COUNTED)\n", task, count, since, usage)
			}
			count++
			continue
		}

		if b.Verbose {
			fmt.Printf("%s iter %d: took: %s, cpu usage: %.02f\n", task, count, since, usage)
		}

		totalCPU += usage
		durations = append(durations, since)
		count++
	}

	return Stats{
		Task:      task,
		TotalCPU:  totalCPU,
		Durations: durations,
		Count:     count,
	}, nil
}
