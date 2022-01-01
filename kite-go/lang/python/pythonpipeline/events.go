package pythonpipeline

import (
	"log"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

// Event represents a pythontracking event
type Event struct {
	pythontracking.Event
	Meta analyze.Metadata
}

// SampleTag implements pipeline.Sample
func (Event) SampleTag() {}

// AnalyzedEvent represents an Event that has been analyzed.
type AnalyzedEvent struct {
	Event   Event
	Context *python.Context
}

// SampleTag implements pipeline.Sample
func (AnalyzedEvent) SampleTag() {}

// EventExpr represents a pythonast.EventExpr that appeared in some analyzed event
type EventExpr struct {
	AnalyzedEvent AnalyzedEvent
	Expr          pythonast.Expr
}

// SampleTag implements pipeline.Sample
func (EventExpr) SampleTag() {}

// AnalyzeEvents analyzes input source.Event samples and emits source.AnalyzedEvents.
func AnalyzeEvents(recreator *servercontext.Recreator, buildLocalCodeIndex bool) transform.OneInOneOutFn {
	return func(s pipeline.Sample) pipeline.Sample {
		ev := s.(Event)

		ctx, err := recreator.RecreateContext(&ev.Event, buildLocalCodeIndex)
		if err != nil {
			log.Printf("error recreating context: %v", err)
			return pipeline.WrapError("context recreation error", err)
		}

		return AnalyzedEvent{
			Event:   ev,
			Context: ctx,
		}
	}
}

// Exprs returns a MapFn that takes as input a source.AnalyzedEvent and outputs
// a slice of source.EventExpr events, one per pythonast.EventExpr in the analyzed event ast.
func Exprs(s pipeline.Sample) []pipeline.Sample {
	ev := s.(AnalyzedEvent)

	var exprs []pythonast.Expr

	pythonast.Inspect(ev.Context.AST, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		if expr, ok := n.(pythonast.Expr); ok {
			exprs = append(exprs, expr)
		}
		return true
	})

	var samples []pipeline.Sample

	for _, expr := range exprs {
		samples = append(samples, EventExpr{
			AnalyzedEvent: ev,
			Expr:          expr,
		})
	}

	return samples
}

// DedupeEvents returns a transform.IncludeFn that de-dupes events by user/machine/filename. Note that it uses
// a closed-over sync.Map to do that, and thus cannot de-dupe across multiple shards.
func DedupeEvents() transform.IncludeFn {
	type umf struct {
		User     int64
		Machine  string
		Filename string
	}

	var m sync.Map

	return func(s pipeline.Sample) bool {
		ev := s.(Event)

		key := umf{User: ev.UserID, Machine: ev.MachineID, Filename: ev.Filename}

		if _, ok := m.Load(key); ok {
			return false
		}

		m.Store(key, struct{}{})
		return true
	}
}
