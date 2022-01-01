package main

// Digests signature failure logs into a summarized JSON representation, recreating the context for each log.

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
	"github.com/spf13/cobra"
)

var (
	// map of server regions to corresponding localfiles S3 buckets. Omitting non-US regions to minimize latency.
	// TODO(damian): get the bucket name from the log once it contains that
	bucketsByRegion = map[string]string{
		"us-west-1": "kite-local-content",
		"us-west-2": "kite-local-content-us-west-2",
		"us-east-1": "kite-local-content-us-east-1",
		"eastus":    "kite-local-content-us-east-1",
		"westus2":   "kite-local-content",
	}
	numReadThreads       = 16
	numAnalysisThreads   = 4
	sitePackagesCheckPct = 10
	start, end           analyze.Date
)

func init() {
	start = analyze.Today()
	end = analyze.Today()
	cmd.Flags().Var(&start, "start", "beginning of date range to analyze")
	cmd.Flags().Var(&end, "end", "end of date range to analyze, inclusive")
}

var cmd = cobra.Command{
	Use:   "sig-failure-report OUT.JSON",
	Short: "analyze signature failure logs and produce a log amenable for post-analysis in e.g. pandas",
	Args:  cobra.ExactArgs(1),
	Run:   run,
}

func main() {
	cmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	// get URIs
	listing, err := analyze.ListRange(segmentsrc.CalleeTracking, start, end)
	fail(err)
	var URIs []string
	for _, d := range listing.Dates {
		URIs = append(URIs, d.URIs...)
	}
	log.Printf("found %d URIs within the provided date range", len(URIs))
	if len(URIs) == 0 {
		fail(errors.New("nothing to do"))
	}

	// open the file before doing heavy lifting so we can fail early if necessary
	w, err := os.Create(args[0])
	fail(err)
	defer w.Close()

	// start loading logs into the `wc` channel for analysis
	type work struct {
		meta analyze.Metadata
		trk  *pythontracking.Event
	}
	wc := make(chan work, numReadThreads*10)
	go func() {
		results := analyze.Analyze(URIs, numReadThreads, string(pythontracking.ServerSignatureFailureEvent),
			func(metadata analyze.Metadata, track *pythontracking.Event) bool {
				if track == nil {
					return false
				}
				wc <- work{metadata, track}
				return true
			})
		if results.Err != nil {
			log.Printf("error(s) encountered in loading logs: %v", err)
		}
		log.Printf("loaded events: %d", results.ProcessedEvents)
		log.Printf("event decode failures: %d", results.DecodeErrors)
		close(wc)
	}()

	// load the context recreator
	recreator, err := servercontext.NewRecreator(bucketsByRegion)
	if err != nil {
		log.Fatal("error creating context recreator", err)
	}

	// start concurrently analyzing loaded logs and put the output into the `ac` channel for writing
	analyzer := newAnalyzer(recreator)
	ac := make(chan *Analyzed, numAnalysisThreads*10)
	var filteredCount, analyzedCount uint32
	processJob := func() error {
		for w := range wc {
			if !shouldProcessEvent(w.trk) {
				atomic.AddUint32(&filteredCount, 1)
				continue
			}

			analyzed, err := analyzer.analyze(w.meta, w.trk)
			if err != nil {
				log.Printf("error in recreating context: %v", err)
				continue
			}

			ac <- analyzed
			atomic.AddUint32(&analyzedCount, 1)
		}
		return nil
	}
	pool := workerpool.New(numAnalysisThreads)
	for i := 0; i < numAnalysisThreads; i++ {
		pool.Add([]workerpool.Job{processJob})
	}
	go func() {
		pool.Wait()
		pool.Stop()
		close(ac)
		log.Printf("filtered events: %d", filteredCount)
		log.Printf("analyzed events: %d", analyzedCount)
	}()

	// write logs synchronously
	enc := json.NewEncoder(w)
	for a := range ac {
		err := enc.Encode(a)
		fail(err)
	}
}

func shouldProcessEvent(track *pythontracking.Event) bool {
	_, ok := bucketsByRegion[track.Region]
	return ok
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
