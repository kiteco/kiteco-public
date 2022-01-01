package data

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
)

// Sample for type induction
type Sample struct {
	Pkg    string
	Func   Symbol
	Return []Symbol
	Attrs  []string
}

// SampleTag implements pipeline.Sample
func (Sample) SampleTag() {}

// Symbol is a version of pythonresource.Symbol
// that is suitable for serialization and deserialization
type Symbol struct {
	Dist keytypes.Distribution
	Path pythonimports.DottedPath
}

// NewSymbol from the specified pythonresource.Symbol
func NewSymbol(s pythonresource.Symbol) Symbol {
	return Symbol{
		Dist: s.Dist(),
		Path: s.Canonical().Path(),
	}
}

// SampleByPkg is aggregated examples by package name
type SampleByPkg map[string][]Sample

// SampleTag implements pipeline.Sample
func (SampleByPkg) SampleTag() {}

// Add implements sample.Addable
func (s SampleByPkg) Add(a sample.Addable) sample.Addable {
	for k, v := range a.(SampleByPkg) {
		s[k] = append(s[k], v...)
	}
	return s
}
