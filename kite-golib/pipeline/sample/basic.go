package sample

// ByteSlice is a sample that wraps []byte
type ByteSlice []byte

// SampleTag implements pipeline.Sample
func (ByteSlice) SampleTag() {}

// StringSlice is a slice of strings
type StringSlice []string

// SampleTag implements pipeline.Sample
func (StringSlice) SampleTag() {}

// placeholderSample sample that indicates that the output of a Feed
// should be ignored, it is used to distinguish
// from a nil sample which is used for control flow
type placeholderSample int

// SampleTag implements pipeline.Sample
func (placeholderSample) SampleTag() {}

// Placeholder sample indicates that the output of a Feed
// should be ignored, it is used to distinguish
// from a nil sample which is used for control flow
const Placeholder = placeholderSample(0)

// String wraps a string
type String string

// SampleTag ...
func (String) SampleTag() {}
