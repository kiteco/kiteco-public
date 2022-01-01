package aggregator

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
)

// sumAggregator aggregates samples that conform to the Addable interface.
type sumAggregator struct {
	name string
	// if shared is true, a single aggregator is shared between all the worker goroutines
	// if shared is false, each worker goroutine has its own aggregator, so aggregation is more quick but
	//   the memory usage is higher
	shared bool

	newFn func() sample.Addable
	inFn  func(pipeline.Sample) sample.Addable

	a sample.Addable
	// m is only used in shared mode
	m sync.Mutex
}

// NewSumAggregator returns a Aggregator which can aggregate Addable instances, given:
// - a newFn which returns an instance of the Addable to aggregate. This instance should be of the type itself
//   (i.e. not a pointer).
// - an inFn which converts from incoming Samples to instances of the Addable to aggregate. If inFn returns nil,
//   the sample is skipped.
func NewSumAggregator(name string, newFn func() sample.Addable, inFn func(pipeline.Sample) sample.Addable) pipeline.Aggregator {
	return &sumAggregator{
		name:   name,
		shared: false,
		newFn:  newFn,
		inFn:   inFn,
	}
}

// NewSharedSumAggregator returns an Aggregator which has the same behavior as the one that is returned by
// NewSumAggregator. The difference is that one aggregator is shared between all the worker goroutines, making
// memory usage more efficient but aggregation less efficient. This should be used if the aggregate is expected to
// occupy a large amount of memory.
func NewSharedSumAggregator(name string, newFn func() sample.Addable, inFn func(pipeline.Sample) sample.Addable) pipeline.Aggregator {
	return &sumAggregator{
		name:   name,
		shared: true,
		newFn:  newFn,
		inFn:   inFn,
	}
}

// Name implements pipeline.Aggregator
func (s *sumAggregator) Name() string {
	return s.name
}

// Clone implements pipeline.Aggregator
func (s *sumAggregator) Clone() pipeline.Dependent {
	if s.shared {
		s.a = s.newFn()
		return s
	}

	return &sumAggregator{
		name:  s.name,
		newFn: s.newFn,
		inFn:  s.inFn,
		a:     s.newFn(),
	}
}

// In implements pipeline.Sample
func (s *sumAggregator) In(sam pipeline.Sample) {
	addable := s.inFn(sam)

	if addable != nil {
		if s.shared {
			// If there's a single aggregator shared between all the workers, we need to lock on the aggregation
			s.m.Lock()
			defer s.m.Unlock()
		}
		s.a = s.a.Add(addable)
	}
}

// ForShard implements pipeline.Sample
func (s *sumAggregator) ForShard(shard, totalShards int) (pipeline.Aggregator, error) {
	return s, nil
}

// AggregateLocal implements pipeline.Aggregator
func (s *sumAggregator) AggregateLocal(clones []pipeline.Aggregator) (pipeline.Sample, error) {
	if s.shared {
		return s.a, nil
	}

	agg := s.newFn()

	for _, c := range clones {
		c := c.(*sumAggregator)
		agg = agg.Add(c.a)
	}

	return agg, nil
}

// FromJSON implements pipeline.Aggregator
func (s *sumAggregator) FromJSON(data []byte) (pipeline.Sample, error) {
	res := s.newFn()

	// Use reflection to get a pointer to the Addable so that we can unmarshal the data to it
	val := reflect.ValueOf(res)
	ptrVal := reflect.New(val.Type())
	ptrVal.Elem().Set(val)

	if err := json.Unmarshal(data, ptrVal.Interface()); err != nil {
		return nil, err
	}

	return res, nil
}

// AggregateFromShard implements pipeline.Aggregator
func (s *sumAggregator) AggregateFromShard(agg pipeline.Sample, shardSample pipeline.Sample, endpoint string) (pipeline.Sample, error) {
	if agg == nil {
		return shardSample, nil
	}
	return agg.(sample.Addable).Add(shardSample.(sample.Addable)), nil
}

// Finalize implements pipeline.Aggregator
func (s *sumAggregator) Finalize() error {
	return nil
}
