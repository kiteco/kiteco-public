package sample

import "github.com/kiteco/kiteco/kite-golib/segment/analyze"

// SegmentEvent contains an event deserialized from Segment logs, along with its associated metadata.
type SegmentEvent struct {
	Metadata analyze.Metadata
	Event    interface{}
}

// SampleTag implements pipeline.Sample
func (SegmentEvent) SampleTag() {}
