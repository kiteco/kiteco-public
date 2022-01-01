package keytypes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Distribution represents a versioned Python distribution
type Distribution struct {
	Name    string
	Version string
}

var distNameNormRe = regexp.MustCompile(`[-_.]+`)

// NormalizeDistName normalizes a distribution name allowing for comparison
// see https://www.python.org/dev/peps/pep-0503/#normalized-names
func NormalizeDistName(name string) string {
	return strings.ToLower(distNameNormRe.ReplaceAllString(name, "-"))
}

// String implements fmt.Stringer
func (dist Distribution) String() string {
	if dist.Version == "" {
		return dist.Name
	}
	return fmt.Sprintf("%s==%s", dist.Name, dist.Version)
}

// Normalize normalizes the Distribution for valid comparison
func (dist Distribution) Normalize() Distribution {
	dist.Name = NormalizeDistName(dist.Name)
	return dist
}

// ParseDistribution is the inverse of Distribution.String
func ParseDistribution(str string) (Distribution, error) {
	parts := strings.Split(str, "==")
	switch len(parts) {
	case 1:
		return Distribution{Name: parts[0]}, nil
	case 2:
		return Distribution{Name: parts[0], Version: parts[1]}, nil
	default: // len(parts) > 2, since strings.Split cannot return an empty slice
		return Distribution{}, errors.Errorf("too many == found in distribution string %s", str)
	}
}

func (dist Distribution) compare(other Distribution) int {
	if dist.Name == other.Name {
		if dist.Version < other.Version {
			return -1
		} else if dist.Version == other.Version {
			return 0
		} else {
			return 1
		}
	}

	// if either is builtin, that one is smaller
	if dist.Name == BuiltinDistributionName {
		return -1
	}
	if other.Name == BuiltinDistributionName {
		return 1
	}

	// otherwise compare the strings
	if dist.Name < other.Name {
		return -1
	}
	return 1
}

// Less checks imposes an (arbitrary but fixed) ordering on Distributions
func (dist Distribution) Less(other Distribution) bool {
	return dist.compare(other) < 0
}

// DistributionList is a sortable collection of distributions
type DistributionList []Distribution

// Len implements sort.Interface
func (l DistributionList) Len() int {
	return len(l)
}

// Swap implements sort.Interface
func (l DistributionList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Less implements sort.Interface
func (l DistributionList) Less(i, j int) bool {
	return l[i].Less(l[j])
}

// Symbol represents a visible symbol (i.e. attribute path) inside a given versioned Python distribution
type Symbol struct {
	Dist Distribution
	Path pythonimports.DottedPath
}

// String implements fmt.Stringer
func (sym Symbol) String() string {
	return fmt.Sprintf("%s:%s", sym.Dist, sym.Path)
}

// Less imposes an (arbitrary but fixed) ordering on Symbols
func (sym Symbol) Less(other Symbol) bool {
	switch sym.Dist.compare(other.Dist) {
	case -1:
		return true
	case 1:
		return false
	case 0:
		return sym.Path.Less(other.Path)
	default:
		panic("invalid result of Distribution comparison")
	}
}

// Truthiness encapsulates information about the reliability of some data as a bit vector
type Truthiness uint8

const (
	// StubTruthiness ...
	StubTruthiness Truthiness = 1 << iota
	// NumpydocTruthiness ...
	NumpydocTruthiness
	// EpytextTruthiness ...
	EpytextTruthiness
	// DynamicAnalysisTruthiness ...
	DynamicAnalysisTruthiness
	// EMModelTruthiness ...
	EMModelTruthiness
)

// FromStub ...
func (t Truthiness) FromStub() bool {
	return t&StubTruthiness > 0
}

// FromNumpydoc ...
func (t Truthiness) FromNumpydoc() bool {
	return t&NumpydocTruthiness > 0
}

// FromEpytext ...
func (t Truthiness) FromEpytext() bool {
	return t&EpytextTruthiness > 0
}

// FromEMModel ...
func (t Truthiness) FromEMModel() bool {
	return t&EMModelTruthiness > 0
}

func (t Truthiness) String() string {
	var parts []string
	if t.FromStub() {
		parts = append(parts, "stub")
	}
	if t.FromNumpydoc() {
		parts = append(parts, "numpydoc")
	}
	if t.FromEpytext() {
		parts = append(parts, "epytext")
	}
	if t.FromEMModel() {
		parts = append(parts, "emmodel")
	}
	return strings.Join(parts, "|")
}
