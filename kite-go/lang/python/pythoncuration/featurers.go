package pythoncuration

import (
	"encoding/gob"
	"io"
	"strings"

	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-golib/text"
)

func init() {
	gob.Register(&codeWidthFeaturer{})
	gob.Register(&codeLengthFeaturer{})
	gob.Register(&absFreqFeaturer{})
	gob.Register(&relativeFreqFeaturer{})
	gob.Register(&argNumFeaturer{})
	gob.Register(&titleLengthFeaturer{})
	gob.Register(&matchedPackageFeaturer{})
}

// Featurer defines what a featurizer for passive search must satisfy.
type Featurer interface {
	Features(string, *Snippet, *dynamicanalysis.ResolvedSnippet) float64
	Label() string
}

// ExampleFeaturer generates features for ranking code
// examples for passive search.
type ExampleFeaturer struct {
	Featurers []Featurer
}

// NewExampleFeaturer returns a pointer to a new ExampleFeaturer instance.
func NewExampleFeaturer() *ExampleFeaturer {
	var featurers []Featurer

	featurers = append(featurers, newCodeWidthFeaturer("code_width"))
	featurers = append(featurers, newCodeLengthFeaturer("code_length"))
	featurers = append(featurers, newAbsFreqFeaturer("abs_freq"))
	featurers = append(featurers, newRelativeFreqFeaturer("relative_freq"))
	featurers = append(featurers, newMatchedPackageFeaturer("matched_package"))

	return &ExampleFeaturer{
		Featurers: featurers,
	}
}

// Features converts an identifier and the query to a feature vector.
func (f *ExampleFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) []float64 {
	var feats []float64
	for _, featurer := range f.Featurers {
		feats = append(feats, featurer.Features(ident, cs, ref))
	}
	return feats
}

// Labels returns the names of the features.
func (f *ExampleFeaturer) Labels() []string {
	var labels []string
	for _, featurer := range f.Featurers {
		labels = append(labels, featurer.Label())
	}
	return labels
}

// --

// codeWidthFeaturer measures the width of the code snippet
// and uses it as a feature.
type codeWidthFeaturer struct {
	Name string
}

// newCodeWidthFeaturer returns a pointer to a new codeWidthFeaturer
// object.
func newCodeWidthFeaturer(name string) *codeWidthFeaturer {
	return &codeWidthFeaturer{
		Name: name,
	}
}

// Features loops through lines in the code of the given snippet
// and finds and returns the length of the widest line.
func (c *codeWidthFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) float64 {
	return float64(cs.Snippet.Width)
}

// Label returns the name of the featurer.
func (c *codeWidthFeaturer) Label() string {
	return c.Name
}

// --

// matchedPackageFeaturer checks whether the code example is in the same package
// as the identifier.
type matchedPackageFeaturer struct {
	Name string
}

// newMatchedPackageFeaturer returns a pointer to a new codeWidthFeaturer
// object.
func newMatchedPackageFeaturer(name string) *matchedPackageFeaturer {
	return &matchedPackageFeaturer{
		Name: name,
	}
}

// Features checks whether the code example is in the same package
// as the identifier.
// and finds and returns the length of the widest line.
func (c *matchedPackageFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) float64 {
	tokens := strings.Split(ident, ".")
	var exp string
	if len(tokens) > 0 {
		exp = strings.ToLower(tokens[0])
	}
	for _, pkg := range strings.Split(cs.Curated.Snippet.RelevantPackages, ",") {
		if strings.ToLower(pkg) == exp {
			return 1.0
		}
	}
	return 0.0
}

// Label returns the name of the featurer.
func (c *matchedPackageFeaturer) Label() string {
	return c.Name
}

// --

// codeLengthFeaturer measures the line number of the code snippet
// and uses it as a feature.
type codeLengthFeaturer struct {
	Name string
}

// newCodeLengthFeaturer returns a pointer to a new codeLengthFeaturer object.
func newCodeLengthFeaturer(name string) *codeLengthFeaturer {
	return &codeLengthFeaturer{
		Name: name,
	}
}

// Features returns the number of lines of code in the snippet
func (c *codeLengthFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) float64 {
	return float64(cs.Snippet.NumLines)
}

// Label returns the name of the featurer.
func (c *codeLengthFeaturer) Label() string {
	return c.Name
}

// --

// absFreqFeaturer counts the absolute number of times the identifier
// appears in the code block.
type absFreqFeaturer struct {
	Name string
}

// newAbsFreqFeaturer returns a pointer to a new absFreqFeaturer object.
func newAbsFreqFeaturer(name string) *absFreqFeaturer {
	return &absFreqFeaturer{
		Name: name,
	}
}

// Features counts the number of times that this identifier appears
// in the code snippet.
func (f *absFreqFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) float64 {
	var count int
	if ref == nil {
		for _, inc := range cs.Snippet.Incantations {
			if inc.ExampleOf == ident {
				count++
			}
		}
	} else {
		for _, r := range ref.References {
			if r.NodeType == "call" && r.FullyQualifiedName == ident {
				count++
			}
		}
	}
	return float64(count)
}

// Label returns the name of the featurer.
func (f *absFreqFeaturer) Label() string {
	return f.Name
}

// --

// relativeFreqFeaturer counts the relative number of times the
// identifier appears in the code block.
type relativeFreqFeaturer struct {
	Name string
}

// newRelativeFreqFeaturer returns a pointer to a new
// relativeFreqFeaturer object.
func newRelativeFreqFeaturer(name string) *relativeFreqFeaturer {
	return &relativeFreqFeaturer{
		Name: name,
	}
}

// Features counts the number of times that this identifier appears
// in the code snippet.
func (f *relativeFreqFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) float64 {
	var count int
	if ref == nil {
		for _, inc := range cs.Snippet.Incantations {
			if inc.ExampleOf == ident {
				count++
			}
		}
		if len(cs.Snippet.Incantations) == 0 {
			return 0.0
		}
		return float64(count) / float64(len(cs.Snippet.Incantations))
	}
	var totalCalls int
	for _, r := range ref.References {
		if r.NodeType == "call" && r.FullyQualifiedName == ident {
			count++
		}
		if r.NodeType == "call" {
			totalCalls++
		}
	}
	if totalCalls == 0 {
		return 0.0
	}
	return float64(count) / float64(totalCalls)
}

// Label returns the name of the featurer.
func (f *relativeFreqFeaturer) Label() string {
	return f.Name
}

// --

// argNumFeaturer counts the number of arguments used in the function call.
type argNumFeaturer struct {
	Name string
}

// newArgNumFeaturer returns a pointer to a new argNumFeaturer object.
func newArgNumFeaturer(name string) *argNumFeaturer {
	return &argNumFeaturer{
		Name: name,
	}
}

// Features returns the number of arguments used in the code snippet.
func (a *argNumFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) float64 {
	var argCount int
	var numInstance int

	for _, inc := range cs.Snippet.Incantations {
		if inc.ExampleOf == ident {
			argCount += inc.NumArgs
			numInstance++
		}
	}

	if numInstance == 0 {
		return 0.0
	}
	return float64(argCount) / float64(numInstance)
}

// Label returns the name of the featurer
func (a *argNumFeaturer) Label() string {
	return a.Name
}

// --

// titleLengthFeaturer counts the length of the title.
type titleLengthFeaturer struct {
	Name string
}

// newTitleLengthFeaturer returns a pointer to a new titleLengthFeaturer.
func newTitleLength(name string) *titleLengthFeaturer {
	return &titleLengthFeaturer{
		Name: name,
	}
}

// Features returns the number of arguments used in the code snippet.
func (f *titleLengthFeaturer) Features(ident string, cs *Snippet, ref *dynamicanalysis.ResolvedSnippet) float64 {
	return float64(len(text.TokenizeNoCamel(cs.Curated.Snippet.Title)))
}

// Label returns the name of the featurer
func (f *titleLengthFeaturer) Label() string {
	return f.Name
}

// --

// NewExampleFeaturerFromGOB loads the passive featurer from a gob file.
func NewExampleFeaturerFromGOB(r io.Reader) (*ExampleFeaturer, error) {
	var featurer ExampleFeaturer
	decoder := gob.NewDecoder(r)
	err := decoder.Decode(&featurer)
	if err != nil {
		return nil, err
	}
	return &featurer, nil
}
