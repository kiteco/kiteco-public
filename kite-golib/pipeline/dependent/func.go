package dependent

import (
	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

type funcDependent struct {
	name string
	in   func(pipeline.Sample)
}

func (f *funcDependent) Name() string {
	return f.name
}

func (f *funcDependent) Clone() pipeline.Dependent {
	return NewFromFunc(f.name, f.in)
}

func (f *funcDependent) In(s pipeline.Sample) {
	f.in(s)
}

// NewFromFunc returns a pipeline.Dependent that is used to wrap a function which operates on a closed over state
// and may have any desired side effects.
func NewFromFunc(name string, in func(pipeline.Sample)) pipeline.Dependent {
	return &funcDependent{
		name: name,
		in:   in,
	}
}
