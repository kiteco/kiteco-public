package pipeline

import (
	"fmt"
	"io"
	"log"
)

// worker maintains clones of the dependent feeds in a pipeline, which lets it run the pipeline concurrently with other
// workers.
type worker struct {
	clone PipeClone

	stats *runStats

	logger io.Writer
}

func newWorker(r *runner) (worker, error) {
	newClone, err := r.clone.CloneForWorker()
	if err != nil {
		return worker{}, err
	}

	return worker{
		clone:  newClone,
		stats:  &r.stats,
		logger: r.opts.Logger,
	}, nil
}

// Run the pipeline for the given record originating from the given source.
func (w worker) Run(s Source, rec Record) {
	w.logf("running %s", rec.Key)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic for source: %s, key: %v", s.Name(), rec.Key)
			panic(r)
		}
	}()

	for _, dep := range w.clone.Dependents[s] {
		w.runDependent(s, rec, dep, rec.Value)
	}
}

// ClonedAggregator for the given original one
func (w worker) ClonedAggregator(agg Aggregator) Aggregator {
	return w.clone.OrigToClone[agg].(Aggregator)
}

func (w worker) runDependent(s Source, rec Record, d Dependent, in Sample) {
	w.stats.IncrFeedIn(w.clone.CloneToOrig[d])

	d.In(in)

	if d, ok := d.(Transform); ok {
		for {
			sample := d.TransformOut()
			if ks, ok := sample.(Keyed); ok {
				if se, ok := ks.Sample.(sampleError); ok {
					w.stats.AddFeedError(w.clone.CloneToOrig[d], s.Name(), rec.Key, se)
					continue
				}
			}

			if se, ok := sample.(sampleError); ok {
				w.stats.AddFeedError(w.clone.CloneToOrig[d], s.Name(), rec.Key, se)
			} else if sample == nil {
				return
			} else {
				w.stats.IncrFeedOut(w.clone.CloneToOrig[d])

				for _, dep := range w.clone.Dependents[d] {
					w.runDependent(s, rec, dep, sample)
				}
			}
		}
	}
}

func (w worker) logf(fstr string, args ...interface{}) {
	if w.logger != nil {
		fmt.Fprintf(w.logger, fstr+"\n", args...)
	}
}
