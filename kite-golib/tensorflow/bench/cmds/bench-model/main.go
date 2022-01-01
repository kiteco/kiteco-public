package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
	"github.com/kiteco/kiteco/kite-golib/tensorflow/bench"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// memUsage returns the resident set size of the process.
func memUsage() (int, error) {
	cmd := exec.Command("ps", "-o", "rss", "-p", strconv.Itoa(os.Getpid()))
	outBytes, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	out := string(outBytes)
	if !strings.Contains(out, "RSS") {
		return 0, fmt.Errorf("unexpected output: %s", out)
	}
	usage, err := strconv.Atoi(strings.TrimSpace(strings.Replace(out, "RSS", "", 1)))
	if err != nil {
		return 0, err
	}
	return usage, nil
}

func main() {
	args := struct {
		FrozenModel string `arg:"required"`
		FeedPath    string `arg:"required"`
		Loop        int
	}{}
	arg.MustParse(&args)

	recs, err := bench.LoadFeedRecords(args.FeedPath)
	fail(err)

	log.Printf("processed %d feed records from %s", len(recs), args.FeedPath)

	baseline, err := memUsage()
	fail(err)

	log.Printf("baseline mem usage: %d", baseline)

	model, err := tensorflow.NewModel(args.FrozenModel)
	fail(err)

	done := make(chan struct{})

	var infTimeSamples []float64

	go func() {
		loop := 1
		if args.Loop != 0 {
			loop = args.Loop
		}
		for i := 0; i < loop; i++ {
			for _, rec := range recs {
				start := time.Now()
				_, err = model.Run(rec.Feeds, rec.Fetches)
				fail(err)
				infTimeSamples = append(infTimeSamples, time.Since(start).Seconds())
			}
		}
		done <- struct{}{}
	}()

	ticker := time.Tick(100 * time.Millisecond)

	var memSamples []int

	var isDone bool
	for !isDone {
		select {
		case <-ticker:
			usage, err := memUsage()
			if err != nil {
				fail(err)
			}
			memSamples = append(memSamples, usage-baseline)
		case <-done:
			isDone = true
			break
		}
	}

	log.Printf("collected %d memory usage samples", len(memSamples))
	log.Printf("collected %d inference time samples", len(infTimeSamples))

	sort.Ints(memSamples)
	var memMax int
	for _, s := range memSamples {
		if s > memMax {
			memMax = s
		}
	}
	memMedian := memSamples[len(memSamples)/2]

	sort.Float64s(infTimeSamples)
	var infTimeMax float64
	for _, s := range infTimeSamples {
		if s > infTimeMax {
			infTimeMax = s
		}
	}
	infTimeMedian := infTimeSamples[len(infTimeSamples)/2]

	fmt.Printf("%d,%d,%f,%f\n", memMedian, memMax, infTimeMedian, infTimeMax)
}
