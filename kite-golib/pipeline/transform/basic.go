package transform

import (
	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

// OneInOneOutFn maps a single input sample to a single output sample.
type OneInOneOutFn func(pipeline.Sample) pipeline.Sample

// WrapOneInOneOutFnKeyed wraps a OneInOneOutFn in a pipeline.Keyed
func WrapOneInOneOutFnKeyed(f OneInOneOutFn) OneInOneOutFn {
	return func(s pipeline.Sample) pipeline.Sample {
		k := s.(pipeline.Keyed)
		res := f(k.Sample)
		if res == nil {
			return nil
		}

		return pipeline.Keyed{
			Key:    k.Key,
			Sample: res,
		}
	}
}

// OneInOneOut is a pipeline.Transform that outputs exactly
// one sample for each input.
type OneInOneOut struct {
	name string
	f    OneInOneOutFn

	s pipeline.Sample
}

// NewOneInOneOut returns a pipeline.Transform with `name` that
// applies f once to each input and outputs the output.
func NewOneInOneOut(name string, f OneInOneOutFn) *OneInOneOut {
	return &OneInOneOut{
		name: name,
		f:    f,
		s:    nil,
	}
}

// NewOneInOneOutKeyed is a convenience function that returns a OneInOneOut pipeline.Transform
// with inputs and outputs that are wrapped in a pipeline.Keyed
func NewOneInOneOutKeyed(name string, f OneInOneOutFn) *OneInOneOut {
	return NewOneInOneOut(name, WrapOneInOneOutFnKeyed(f))
}

// Name implements pipeline.Transform
func (t *OneInOneOut) Name() string {
	return t.name
}

// In implements pipeline.Transform
func (t *OneInOneOut) In(s pipeline.Sample) {
	t.s = t.f(s)
}

// TransformOut implements pipeline.Transform
func (t *OneInOneOut) TransformOut() pipeline.Sample {
	s := t.s
	t.s = nil
	return s
}

// Clone implements pipeline.Transform
func (t *OneInOneOut) Clone() pipeline.Dependent {
	return NewOneInOneOut(t.name, t.f)
}

// MapFn maps a single input sample to multiple output samples.
type MapFn func(pipeline.Sample) []pipeline.Sample

// Map wraps a function that inputs a sample and returns a slice of transformed samples.
type Map struct {
	name  string
	mapFn MapFn

	samples []pipeline.Sample
	pos     int
}

// NewMap returns a pipeline.Transform with `name` that applies
// `mapFn` to each input and returns the outputs.
func NewMap(name string, mapFn MapFn) *Map {
	return &Map{
		name:  name,
		mapFn: mapFn,
	}
}

// In implements pipeline.Transform
func (t *Map) In(s pipeline.Sample) {
	t.samples = t.mapFn(s)
	t.pos = 0
}

// TransformOut implements pipeline.Transform
func (t *Map) TransformOut() pipeline.Sample {
	if t.pos >= len(t.samples) {
		return nil
	}

	sample := t.samples[t.pos]
	if sample == nil {
		panic("nil is an invalid value for a sample")
	}
	t.pos++
	return sample
}

// Name implements pipeline.Transform
func (t *Map) Name() string { return t.name }

// Clone implements pipeline.Transform
func (t *Map) Clone() pipeline.Dependent {
	return NewMap(t.name, t.mapFn)
}

// MapFnChan maps a single input sample to a channel of output samples.
type MapFnChan func(pipeline.Sample) chan pipeline.Sample

// MapChan wraps a function that inputs a sample and returns a slice of transformed samples.
type MapChan struct {
	name  string
	mapFn MapFnChan

	samples chan pipeline.Sample
}

// NewMapChan returns a pipeline.Transform with `name` that applies
// `mapFn` to each input and returns the outputs.
func NewMapChan(name string, mapFn MapFnChan) *MapChan {
	return &MapChan{
		name:  name,
		mapFn: mapFn,
	}
}

// In implements pipeline.Transform
func (t *MapChan) In(s pipeline.Sample) {
	t.samples = t.mapFn(s)
}

// TransformOut implements pipeline.Transform
func (t *MapChan) TransformOut() pipeline.Sample {
	if t.samples == nil {
		return nil
	}
	res, ok := <-t.samples
	if !ok {
		return nil
	}
	return res
}

// Name implements pipeline.Transform
func (t *MapChan) Name() string { return t.name }

// Clone implements pipeline.Transform
func (t *MapChan) Clone() pipeline.Dependent {
	return NewMapChan(t.name, t.mapFn)
}

// IncludeFn returns true for samples that should be emitted
// from a Filter transform.
type IncludeFn func(pipeline.Sample) bool

// Filter wraps a function that inputs a sample and returns true if the sample should be included or not.
type Filter struct {
	name      string
	includeFn IncludeFn

	sample pipeline.Sample
}

// NewFilter is a transform that outputs a sample if includeFn returns true.
func NewFilter(name string, includeFn IncludeFn) *Filter {
	return &Filter{
		name:      name,
		includeFn: includeFn,
	}
}

// In implements pipeline.Transform.
func (f *Filter) In(s pipeline.Sample) {
	if f.includeFn(s) {
		f.sample = s
	} else {
		f.sample = nil
	}
}

// TransformOut implements pipeline.Transform.
func (f *Filter) TransformOut() pipeline.Sample {
	sample := f.sample
	f.sample = nil
	return sample
}

// Name implements pipeline.Transform
func (f *Filter) Name() string { return f.name }

// Clone implements pipeline.Transform.
func (f *Filter) Clone() pipeline.Dependent {
	return NewFilter(f.name, f.includeFn)
}

// NewKeyedNilFilter filters out pipeline.Keyed samples for whose value is nil.
func NewKeyedNilFilter(name string) pipeline.Transform {
	return NewFilter(name, func(s pipeline.Sample) bool {
		return s.(pipeline.Keyed).Sample != nil
	})
}
