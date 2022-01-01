package pythoncode

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// PackageStats contains raw counts for how many incantations are part of the
// named Package. It includes a breakdown by method.
type PackageStats struct {
	Package string
	Count   int
	Methods []*MethodStats
}

// MethodStats holds a count for the number of times the method is referenced in
// an Snippet.
type MethodStats struct {
	Ident string
	Count int
}

// MethodsByCount implements the sort interface
type MethodsByCount []*MethodStats

func (mbc MethodsByCount) Len() int           { return len(mbc) }
func (mbc MethodsByCount) Swap(i, j int)      { mbc[i], mbc[j] = mbc[j], mbc[i] }
func (mbc MethodsByCount) Less(i, j int) bool { return mbc[i].Count < mbc[j].Count }

// --

// StringCount is a simple structure to store a string/count pair
type StringCount struct {
	Value string
	Count int
}

// StringCountFromMap converts a map[string]int to a sorted slice of []*StringCount
func StringCountFromMap(vals map[string]int) []*StringCount {
	var counts []*StringCount
	for value, count := range vals {
		counts = append(counts, &StringCount{value, count})
	}
	sort.Sort(sort.Reverse(stringByCount(counts)))
	return counts
}

type stringByCount []*StringCount

func (n stringByCount) Len() int           { return len(n) }
func (n stringByCount) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n stringByCount) Less(i, j int) bool { return n[i].Count < n[j].Count }

// ArgStats aggregates information about arguments passed into functions
type ArgStats struct {
	Name           string
	Type           string
	Keyword        bool
	Count          int
	ExprStrs       []*StringCount `json:"VarNames"` // old name (in data) was VarNames
	Types          []*StringCount
	LiteralsByType map[string][]*StringCount
}

// NewArgStats returns a new initialized ArgStats structure.
func NewArgStats() *ArgStats {
	return &ArgStats{
		LiteralsByType: make(map[string][]*StringCount),
	}
}

// MethodPatterns contains argument statistics and signature patterns
// for a method.
type MethodPatterns struct {
	Method      string
	MethodCount int
	Args        []*ArgStats
	Kwargs      map[string]*ArgStats
	Patterns    []*SignaturePattern

	// This is a flag to indicate if this MethodPattern has been preprocessed
	// so it can be used for signature completions.
	processed bool
}

// SignaturePattern represents a canonical invocation type.
type SignaturePattern struct {
	PatternCount int
	Signature    string
	Frequency    float64
	Args         int
	Kwargs       []string

	// These unexported fields are used to store precomputed structures that
	// are helpful during signature completions.
	all    []*ArgStats
	args   []*ArgStats
	kwargs map[string]*ArgStats
}

func (s *SignaturePattern) legacySignature(method string) string {
	var args []string
	for _, arg := range s.all {
		if arg.Keyword {
			args = append(args, fmt.Sprintf("%s=%s", arg.Name, arg.Type))
		} else {
			args = append(args, arg.Type)
		}
	}
	return fmt.Sprintf("%s(%s)", method, strings.Join(args, ", "))
}

// PrivateArgs exports the private `args` member for use by the signature pattern resource builder
func (s *SignaturePattern) PrivateArgs() []*ArgStats {
	return s.args
}

// PrivateKwargs exports the private `kwargs` member for use by the signature pattern resource builder
func (s *SignaturePattern) PrivateKwargs() map[string]*ArgStats {
	return s.kwargs
}

// CooccurrencePattern represents a group of methods that occur together
type CooccurrencePattern struct {
	Method       string
	MethodCount  int
	ClusterID    int
	ClusterCount int
	Frequency    float64
	Pattern      []string
	Hashes       []string
}

// GroupedStats represents an identifier that was used in a particular aggregation pool,
// which could be a file, directory, or repository.
type GroupedStats struct {
	Package    string
	Identifier string
	Counts     map[string]int
}

// ObjectAttribute represents an attribute found associated with an object.
type ObjectAttribute struct {
	Parent     string
	Identifier string
	Type       string
	Count      int
	Frequency  float64
}

// ObjectAttributeByCount lets us sort attributes by count.
type ObjectAttributeByCount []*ObjectAttribute

func (b ObjectAttributeByCount) Len() int           { return len(b) }
func (b ObjectAttributeByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ObjectAttributeByCount) Less(i, j int) bool { return b[i].Count < b[j].Count }

// ObjectUsage contains an object and attributes accessed in a single "usage". A usage
// is either an occurance in a function, or in a class.
type ObjectUsage struct {
	Identifier string
	Attributes []*ObjectAttribute
}

// ObjectSummary contains a summary of attributes observed for a given identifier
type ObjectSummary struct {
	Identifier string
	Count      int
	Attributes []*ObjectAttribute
}

// Kwargs represents possible **kwargs that can be passed to a function.
type Kwargs struct {
	// AnyName for the `pythonimports.Node` associated with the function.
	AnyName pythonimports.DottedPath
	// Kwargs maps from possible **kwarg name to information about the frequency of the name and the type information.
	Kwargs map[string]*Kwarg
	// Name is the name of the `**kwargs` as specified in the arg spec.
	Name string
}

// Kwarg represents a possible **kwarg that can be passed to a function.
type Kwarg struct {
	Count int64
	Types map[string]int64
}

// --

type kwargsByCount struct {
	kwargs   []string
	patterns *MethodPatterns
}

func (s kwargsByCount) Len() int      { return len(s.kwargs) }
func (s kwargsByCount) Swap(i, j int) { s.kwargs[i], s.kwargs[j] = s.kwargs[j], s.kwargs[i] }
func (s kwargsByCount) Less(i, j int) bool {
	return s.score(s.kwargs[i]) < s.score(s.kwargs[j])
}

func (s kwargsByCount) score(val string) int {
	if arg, ok := s.patterns.Kwargs[val]; ok {
		return arg.Count
	}
	return 0
}
