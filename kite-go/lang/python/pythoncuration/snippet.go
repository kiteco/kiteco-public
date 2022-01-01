package pythoncuration

import (
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
)

// Snippet groups a curation.Example with the associated
// pythoncode.Snippet objects.
type Snippet struct {
	Curated *curation.Example
	Snippet *pythoncode.Snippet
}

// Attribute contains information about an attribute accessed on a type.
type Attribute struct {
	Type      string
	Attribute string
}

// AnalyzedSnippet is a CuratedSnippet with data from dynamic analysis.
type AnalyzedSnippet struct {
	Snippet    *Snippet
	Attributes []Attribute // all attributes accessed on all types in this snippet
}
