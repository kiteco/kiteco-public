package main

import (
	"errors"
	"fmt"
	"go/token"
	"log"
	"net/url"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
	"github.com/spf13/cobra"
)

var (
	numReadThreads     = 16
	numAnalysisThreads = 4
	start, end         analyze.Date
)

func init() {
	start = analyze.Today()
	end = analyze.Today()
	cmd.Flags().Var(&start, "start", "beginning of date range to analyze")
	cmd.Flags().Var(&end, "end", "end of date range to analyze, inclusive")
}

var cmd = cobra.Command{
	Use:   "sig-failure-report",
	Short: "analyze signature failure logs and produce a log amenable for post-analysis via grep",
	Args:  cobra.NoArgs,
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

	processJob := func() error {
		for w := range wc {
			process(w.meta, w.trk)
		}
		return nil
	}
	pool := workerpool.New(numAnalysisThreads)
	for i := 0; i < numAnalysisThreads; i++ {
		pool.Add([]workerpool.Job{processJob})
	}

	pool.Wait()
	pool.Stop()
}

func process(meta analyze.Metadata, track *pythontracking.Event) {
	if track.Callee == nil {
		return
	}

	buffer := []byte(track.Buffer)
	cursor := token.Pos(track.Offset)

	incrLexer := pythonscanner.NewIncrementalFromBuffer(buffer, pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	})

	parseOpts := pythonparser.Options{
		Approximate: true,
		Cursor:      &cursor,
	}
	parseOpts.ScanOptions.Label = track.Filename

	mod, _ := pythonparser.ParseWords(kitectx.Background(), buffer, incrLexer.Words(), parseOpts)

	trackingURL := fmt.Sprintf("http://test-6.kite.com:3031/event?uri=%s&message_id=%s", url.QueryEscape(meta.ID.URI), url.QueryEscape(meta.ID.ID))
	log.Printf("LIVE_FAILURE %s %s", track.Callee.Failure, trackingURL)
	if mod == nil {
		log.Printf("PARSE_FAILURE %s", trackingURL)
	} else {
		callExpr, outsideParens, _ := python.FindCallExpr(kitectx.Background(), mod, buffer, int64(cursor))
		if callExpr == nil {
			log.Printf("NO_CALL_EXPR %s", trackingURL)
		} else if outsideParens {
			log.Printf("OUTSIDE_PARENS %s", trackingURL)
		}
	}
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
