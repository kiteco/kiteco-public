package sample

import "github.com/kiteco/kiteco/kite-golib/pipeline"

// Addable describes a Sample that can be aggregated with other Samples of the same type to produce another Sample
// of the same type.
// Addable must be JSON-serializable, since the aggregation needs to cross instance boundaries.
type Addable interface {
	pipeline.Sample
	// Add returns an Addable containing the sum of this one and other. It is OK for Add to mutate the receiver.
	Add(other Addable) Addable
}
