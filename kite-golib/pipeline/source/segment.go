package source

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
)

// DefaultSegmentEventsOpts ...
var DefaultSegmentEventsOpts = SegmentEventsOpts{
	NumReaders: 16,
}

// SegmentEventsOpts bundles the options for tracking events
type SegmentEventsOpts struct {
	NumReaders int
	MaxEvents  int
	Logger     io.Writer
	CacheRoot  string
	ResultsDir string // If set, each shard will write a JSON summary of Segment log stats to this directory.
	// MaxDecodeErrors, if >= 0, defines the point at which the analyzer quits if that many decode errors are exceeded
	MaxDecodeErrors int64
}

// SegmentEvents is a Source that emits pythontracking events that have been stored in S3 for a given date range.
type SegmentEvents struct {
	name       string
	startDate  analyze.Date
	endDate    analyze.Date
	source     segmentsrc.Source
	prototypes map[string]interface{}
	opts       SegmentEventsOpts

	events      chan sample.SegmentEvent
	shard       int
	totalShards int
	uris        []string
}

// NewSegmentEvents for the specified dates and event type. If maxEvents > 0, the number of events returned is
// limited to that number.
// prototypes should contain a map of event names to objects of the same types that the messages should be deserialized
// to. If prototypes has a blank key and an event name does not match any of the other keys, it is deserialized to that
// corresponding value.
func NewSegmentEvents(name string, startDate analyze.Date, endDate analyze.Date,
	source segmentsrc.Source, prototypes map[string]interface{}, opts SegmentEventsOpts) *SegmentEvents {
	return &SegmentEvents{
		name:       name,
		startDate:  startDate,
		endDate:    endDate,
		source:     source,
		prototypes: prototypes,
		opts:       opts,
	}
}

// ForShard implements pipeline.Source
func (t *SegmentEvents) ForShard(shard, totalShards int) (pipeline.Source, error) {
	var maxEvents int
	if t.opts.MaxEvents > 0 {
		maxEvents = int(math.Ceil(float64(t.opts.MaxEvents) / float64(totalShards)))
	}

	opts := t.opts
	opts.MaxEvents = maxEvents

	listing, err := analyze.ListRange(t.source, t.startDate, t.endDate)
	if err != nil {
		return nil, fmt.Errorf("error getting URIs: %v", err)
	}

	var allURIs []string
	for _, d := range listing.Dates {
		allURIs = append(allURIs, d.URIs...)
	}

	t.logf("found %d URIs in the provided date range", len(allURIs))
	if len(allURIs) == 0 {
		return nil, fmt.Errorf("no URIs found for data range %v to %v", t.startDate, t.endDate)
	}

	var shardURIs []string
	for _, u := range allURIs {
		// use of the hash of the URI to determine which shard should handle it
		h := fnv.New64()
		h.Write([]byte(u))
		if h.Sum64()%uint64(totalShards) == uint64(shard) {
			shardURIs = append(shardURIs, u)
		}
	}

	t.logf("%d URIs for shard %d/%d", len(shardURIs), shard, totalShards)

	c := &SegmentEvents{
		name:       t.name,
		startDate:  t.startDate,
		endDate:    t.endDate,
		source:     t.source,
		prototypes: t.prototypes,
		opts:       opts,

		events:      make(chan sample.SegmentEvent, t.opts.NumReaders*10),
		shard:       shard,
		totalShards: totalShards,
		uris:        shardURIs,
	}

	c.start()

	return c, nil
}

// results that are saved to JSON files for debugging if opts.ResultsDir is set
type segmentResults struct {
	Err                string
	DecodeErrors       int64
	DecodeErrorsSample []string
	URIs               int
	CompletedURIs      int64
	ProcessedEvents    int64
	SkippedEvents      int64
}

func (t *SegmentEvents) start() {
	t.logf("SegmentEvents shard %d/%d: getting events from %v to %v",
		t.shard, t.totalShards, t.startDate, t.endDate)

	go func() {
		var count int
		opts := analyze.Opts{
			Logger:          t.opts.Logger,
			NumGo:           t.opts.NumReaders,
			Cache:           t.opts.CacheRoot,
			MaxDecodeErrors: t.opts.MaxDecodeErrors,
		}

		results := analyze.WithPrototypes(opts, t.uris, t.prototypes,
			func(metadata analyze.Metadata, ev interface{}) bool {
				if ev == nil {
					return true
				}

				if t.opts.MaxEvents > 0 && count >= t.opts.MaxEvents {
					return false
				}

				t.events <- sample.SegmentEvent{Metadata: metadata, Event: ev}
				count++

				return true
			})

		if results.Err != nil {
			panic(fmt.Errorf("error loading logs: %v", results.Err))
		}
		t.logf("loaded events: %d, event decode failures: %d", results.ProcessedEvents, results.DecodeErrors)
		if opts.MaxDecodeErrors >= 0 && results.DecodeErrors > opts.MaxDecodeErrors {
			// if we're panicking from too many decode errors, log to both the log and stderr to make debugging
			// more straightforward
			for _, e := range results.DecodeErrorsSample {
				log.Printf("sampled decode error: %v", e)
				t.logf("sampled decode error: %v", e)
			}
			panic("too many decode errors, panicking")
		}
		close(t.events)

		// TODO: perhaps the source should emit a sample that corresponds to the results?
		if t.opts.ResultsDir != "" {
			var errStr string
			if results.Err != nil {
				errStr = results.Err.Error()
			}

			des := make([]string, 0, len(results.DecodeErrorsSample))
			for _, e := range results.DecodeErrorsSample {
				des = append(des, e.Error())
			}

			res := segmentResults{
				Err:                errStr,
				DecodeErrors:       results.DecodeErrors,
				DecodeErrorsSample: des,
				URIs:               len(t.uris),
				CompletedURIs:      results.CompletedURIs,
				ProcessedEvents:    results.ProcessedEvents,
				SkippedEvents:      results.SkippedEvents,
			}
			outf, err := fileutil.NewBufferedWriter(
				fmt.Sprintf("%s/shard-%d-of-%d.json", t.opts.ResultsDir, t.shard, t.totalShards))
			if err != nil {
				t.logf("could not create results file: %v", err)
				return
			}
			defer outf.Close()

			if err := json.NewEncoder(outf).Encode(res); err != nil {
				t.logf("could not write results file: %v", err)
				return
			}
		}
	}()
}

func (t *SegmentEvents) logf(fmtstr string, args ...interface{}) {
	if t.opts.Logger != nil {
		if !strings.HasSuffix(fmtstr, "\n") {
			fmtstr += "\n"
		}
		fmt.Fprintf(t.opts.Logger, fmtstr, args...)
	}
}

// SourceOut implements pipeline.Source
func (t *SegmentEvents) SourceOut() pipeline.Record {
	event, ok := <-t.events
	if !ok {
		return pipeline.Record{}
	}

	return pipeline.Record{
		Key:   event.Metadata.ID.String(),
		Value: event,
	}
}

// Name implements pipeline.Source
func (t *SegmentEvents) Name() string { return t.name }
