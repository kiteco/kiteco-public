package pythonpipeline

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
)

var sourceByEventType = map[pythontracking.EventType]segmentsrc.Source{
	pythontracking.ServerSignatureFailureEvent:   segmentsrc.CalleeTracking,
	pythontracking.ServerCompletionsFailureEvent: segmentsrc.CompletionsTracking,
}

// DefaultTrackingEventsOpts ...
var DefaultTrackingEventsOpts = TrackingEventsOpts{
	NumReaders: 16,
}

// TrackingEventsOpts bundles the options for tracking events
type TrackingEventsOpts struct {
	NumReaders int
	MaxEvents  int
	Logger     io.Writer
	// ShardByUMF, if set, shards by user/machine/file instead of just by message ID. This is useful if we want to
	// do local deduping on the user/machine/file level.
	ShardByUMF bool

	// DedupByUMF use the dedup filter directly in the source. That allows to process a certain amount of dedup events
	// With the MaxEvents counter.
	DedupByUMF bool

	CacheRoot string
	// MaxDecodeErrors, if >= 0, defines the point at which the analyzer quits if that many decode errors are exceeded
	MaxDecodeErrors int64
}

type event struct {
	meta analyze.Metadata
	trk  *pythontracking.Event
}

// TrackingEvents is a Source that emits pythontracking events that have been stored in S3 for a given date range.
type TrackingEvents struct {
	startDate   analyze.Date
	endDate     analyze.Date
	eventType   pythontracking.EventType
	opts        TrackingEventsOpts
	dedupFilter transform.IncludeFn

	events      chan event
	shard       int
	totalShards int
}

// NewTrackingEvents for the specified dates and event type. If maxEvents > 0, the number of events returned is
// limited to that number.
func NewTrackingEvents(startDate analyze.Date, endDate analyze.Date, eventType pythontracking.EventType, opts TrackingEventsOpts) *TrackingEvents {
	if opts.DedupByUMF && !opts.ShardByUMF {
		opts.ShardByUMF = true
		fmt.Println("INFO: The option ShardByUMF is required when using DedupByUMF to ensure the dedup will be done correctly for sharded executions. It has been forced to true.")
	}

	var dedupFilter transform.IncludeFn
	if opts.DedupByUMF {
		dedupFilter = DedupeEvents()
	}

	return &TrackingEvents{
		startDate:   startDate,
		endDate:     endDate,
		eventType:   eventType,
		opts:        opts,
		dedupFilter: dedupFilter,
	}
}

// ForShard implements pipeline.Source
func (t *TrackingEvents) ForShard(shard, totalShards int) (pipeline.Source, error) {
	var maxEvents int
	if t.opts.MaxEvents > 0 {
		maxEvents = int(math.Ceil(float64(t.opts.MaxEvents) / float64(totalShards)))
	}

	var dedupFilter transform.IncludeFn
	if t.opts.DedupByUMF {
		dedupFilter = DedupeEvents()
	}

	c := &TrackingEvents{
		startDate:   t.startDate,
		endDate:     t.endDate,
		eventType:   t.eventType,
		events:      make(chan event, t.opts.NumReaders*10),
		shard:       shard,
		totalShards: totalShards,
		dedupFilter: dedupFilter,
		opts: TrackingEventsOpts{
			NumReaders: t.opts.NumReaders,
			MaxEvents:  maxEvents,
			Logger:     t.opts.Logger,
			CacheRoot:  t.opts.CacheRoot,
			DedupByUMF: t.opts.DedupByUMF,
		},
	}

	c.start()

	return c, nil
}

func (t *TrackingEvents) start() {
	t.logf("TrackingEvents shard %d/%d: getting events from %v to %v",
		t.shard, t.totalShards, t.startDate, t.endDate)

	listing, err := analyze.ListRange(sourceByEventType[t.eventType], t.startDate, t.endDate)

	if err != nil {
		log.Fatalf("error listing events in range: %v", err)
	}

	var URIs []string
	for _, d := range listing.Dates {
		URIs = append(URIs, d.URIs...)
	}

	t.logf("found %d URIs in the provided date range", len(URIs))
	if len(URIs) == 0 {
		log.Fatalf("no URIs found for data range %v to %v", t.startDate, t.endDate)
	}

	go func() {
		var count int
		opts := analyze.Opts{
			Logger:          t.opts.Logger,
			NumGo:           t.opts.NumReaders,
			Cache:           t.opts.CacheRoot,
			MaxDecodeErrors: t.opts.MaxDecodeErrors,
		}
		results := analyze.WithOpts(opts, URIs, string(t.eventType),
			func(metadata analyze.Metadata, track *pythontracking.Event) bool {
				if track == nil {
					return false
				}
				if t.opts.MaxEvents > 0 && count >= t.opts.MaxEvents {
					return false
				}

				// Hash the ID (or user/machine/file) and see whether the modulo of the hash falls into the range of
				// this shard
				hashSource := metadata.ID.String()
				if t.opts.ShardByUMF {
					hashSource = fmt.Sprintf("%d/%s/%s", track.UserID, track.MachineID, track.Filename)
				}
				h := fnv.New64()
				h.Write([]byte(hashSource))
				if h.Sum64()%uint64(t.totalShards) != uint64(t.shard) {
					return true
				}

				e := event{metadata, track}
				dedupEvent := Event{
					Event: *e.trk,
					Meta:  e.meta,
				}
				if !t.opts.DedupByUMF || t.dedupFilter(dedupEvent) {
					t.events <- e
					count++
				}

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
	}()
}

func (t *TrackingEvents) logf(fmtstr string, args ...interface{}) {
	if t.opts.Logger != nil {
		if !strings.HasSuffix(fmtstr, "\n") {
			fmtstr += "\n"
		}
		fmt.Fprintf(t.opts.Logger, fmtstr, args...)
	}
}

// SourceOut implements pipeline.Source
func (t *TrackingEvents) SourceOut() pipeline.Record {
	event, ok := <-t.events
	if !ok {
		return pipeline.Record{}
	}

	return pipeline.Record{
		Key: event.meta.ID.String(),
		Value: Event{
			Event: *event.trk,
			Meta:  event.meta,
		},
	}
}

// Name implements pipeline.Source
func (t *TrackingEvents) Name() string { return "TrackingEvents" }
