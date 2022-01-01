package source

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

// Func wraps a function that emits
// records as a source, f is assumed to be goroutine safe
func Func(name string, f func() pipeline.Record) pipeline.Source {
	return &funcSource{
		name: name,
		f:    f,
	}
}

type funcSource struct {
	name string
	f    func() pipeline.Record
}

// ForShard ...
func (f *funcSource) ForShard(shard int, total int) (pipeline.Source, error) {
	if total > 1 {
		return nil, fmt.Errorf("func source only works in non distributed environment (e.g total shards = 1, got %d)", total)
	}
	return f, nil
}

// Name ...
func (f *funcSource) Name() string {
	return f.name
}

// SourceOut ...
func (f *funcSource) SourceOut() pipeline.Record {
	return f.f()
}
