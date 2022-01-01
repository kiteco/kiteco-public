// TODO(naman) here, wherever you see pythonresource.Symbol, the distribution is irrelevant, since the model gets keyed on the path hash;
// eventually, we'll want to include the distribution in the key, but for now this is good, since an arbitrary matching symbol for a path
// may be used when computing probabilities

package typeinduction

import (
	"log"
	"math"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

const (
	outlierTypeLogp = -10.
	outlierAttrLogp = -10.
	builtinPkg      = "builtins"
	alpha           = 2.
	anneal          = 0.85
)

var replacements = map[string]string{
	"builtins.bytearray": "builtins.list",
}

// Element is a named entry within a discrete probability distribution
type Element struct {
	Name           string  `json:"name"`
	LogProbability float64 `json:"log_probability"`
}

// Type represents a python class together with a probability distribution over its attributes
type Type struct {
	Name       string    `json:"type"`
	Attributes []Element `json:"attributes"`
}

// ResolvedType represents a python class together with a probability distribution over its attributes.
type ResolvedType struct {
	Symbol     pythonresource.Symbol
	Attributes []Element
}

// Function represents a fully qualified python function together with a probability distribution
// over its return type
type Function struct {
	Name       string    `json:"function"`
	ReturnType []Element `json:"return_type"`
}

// ResolvedCandidate is a possible return type with a log probability
type ResolvedCandidate struct {
	Symbol         pythonresource.Symbol
	LogProbability float64
}

// ResolvedFunction represents a python function together with a probability distribution
// over its return type
type ResolvedFunction struct {
	Symbol     pythonresource.Symbol
	ReturnType []ResolvedCandidate
}

// Options encapsulates parameters used to construct a client
type Options struct {
	Types           string
	Functions       string
	DependencyGraph map[string]*pythonimports.Package
}

// DefaultRoot is the default root directory for type inference models
var DefaultRoot = "s3://kite-data/type-inference-models/2016-11-04_17-54-37-PM/"

// DefaultClientOptions is a set of reasonable defaults for constructing a client
var DefaultClientOptions = Options{
	Types:     fileutil.Join(DefaultRoot, "/types.json.gz"),
	Functions: fileutil.Join(DefaultRoot, "/functions.json.gz"),
}

// OptionsFromPath generates a client Option object using the base path provided for the models.
func OptionsFromPath(path string) Options {
	return Options{
		Types:     fileutil.Join(path, "/types.json.gz"),
		Functions: fileutil.Join(path, "/functions.json.gz"),
	}
}

// Client performs type induction using a pretrained type and function models
type Client struct {
	Types           map[pythonimports.Hash]*ResolvedType
	Functions       map[pythonimports.Hash]*ResolvedFunction
	TypePriors      map[pythonimports.Hash]float64
	DependencyGraph map[string]*pythonimports.Package
	SymbolGraph     pythonresource.Manager
}

// LoadModel constructs a client from data files in options
func LoadModel(manager pythonresource.Manager, opts Options) (*Client, error) {
	var types []*Type
	err := serialization.Decode(opts.Types, func(t *Type) {
		types = append(types, t)
	})
	if err != nil {
		return nil, err
	}

	var functions []*Function
	err = serialization.Decode(opts.Functions, func(f *Function) {
		functions = append(functions, f)
	})
	if err != nil {
		return nil, err
	}

	return ModelFromData(types, functions, manager, opts.DependencyGraph), nil
}

// ModelFromData constructs a client from data
func ModelFromData(ts []*Type, fs []*Function, rm pythonresource.Manager, deps map[string]*pythonimports.Package) *Client {
	numMissingByPkg := make(map[string]int)
	types := make(map[pythonimports.Hash]*ResolvedType)
	for _, t := range ts {
		sym, err := rm.PathSymbol(pythonimports.NewDottedPath(t.Name))
		if err != nil {
			top := t.Name
			if period := strings.Index(top, "."); period != -1 {
				top = t.Name[:period]
			}
			numMissingByPkg[top]++
			continue
		}

		types[sym.PathHash()] = &ResolvedType{
			Symbol:     sym,
			Attributes: t.Attributes,
		}
	}

	for pkg, count := range numMissingByPkg {
		if count > 2 {
			log.Printf("%d classes from type induction not found in %s", count, pkg)
		}
	}

	var numNodes, numMissing int
	functions := make(map[pythonimports.Hash]*ResolvedFunction)
	for _, f := range fs {
		sym, err := rm.PathSymbol(pythonimports.NewDottedPath(f.Name))
		numNodes++
		if err != nil {
			numMissing++
			continue
		}

		function := ResolvedFunction{
			Symbol: sym,
		}

		for _, c := range f.ReturnType {
			name := c.Name
			if replacement, ok := replacements[name]; ok {
				name = replacement
			}

			sym, err := rm.PathSymbol(pythonimports.NewDottedPath(name))
			numNodes++
			if err != nil {
				numMissing++
				continue
			}
			function.ReturnType = append(function.ReturnType, ResolvedCandidate{
				Symbol:         sym,
				LogProbability: c.LogProbability,
			})
		}
		if len(function.ReturnType) == 0 {
			continue
		}

		functions[function.Symbol.PathHash()] = &function
	}

	if numMissing > 0 {
		log.Printf("loaded type induction tables containing %d nodes, of which %d were missing from graph",
			numNodes, numMissing)
	}

	client := &Client{
		Types:           types,
		Functions:       functions,
		TypePriors:      make(map[pythonimports.Hash]float64),
		DependencyGraph: deps,
		SymbolGraph:     rm,
	}
	return client
}

// SymbolProbability represents a pythontype.Value and an associated probability
type SymbolProbability struct {
	Symbol      pythonresource.Symbol
	Probability float64
}

// Distribution represents a Distribution over pythontype.Values
type Distribution map[pythontype.FlatID]*SymbolProbability

// Estimate represents a probability distribution over the type of a variable
type Estimate struct {
	MostProbableType pythonresource.Symbol
	// Distribution is slice of values and probabilities, it is guaranteed to
	// contain at least one element and the probabilities sum to 1.
	Distribution []SymbolProbability
}

// TopByPercent returns a slice containing the `limit` (as a cumulative percentage) most probable symbols
func (e Estimate) TopByPercent(limit float64) []SymbolProbability {
	acc := float64(0)
	for i, sp := range e.Distribution {
		acc += sp.Probability
		if acc > limit {
			return e.Distribution[:i+1]
		}
	}
	return e.Distribution
}

// Observation represents the observables from which a type estimate is computed
type Observation struct {
	// ReturnedFrom is the name of the function that the object was returned from
	ReturnedFrom pythonresource.Symbol
	// Attributes is the attributes that were accessed on the object
	Attributes []string
	// Imports are the imports used in the current file, these can only be top level imports, e.g os or sys
	// sub modules are not yet suppoted, e.g os.path is not supported.
	// TODO(juan): support sub modules
	Imports []string
}

// EstimateType computes the distribution over types for a variable given the function it
// was returned from and the attributes accessed on it. If no type information could be
// induced then this function returns nil.
func (c *Client) EstimateType(observation Observation) Estimate {
	types := c.getTypes(observation)
	if len(types) == 0 {
		return Estimate{}
	}

	bestLogp := math.Inf(-1)
	var bestType pythonresource.Symbol

	distr := make([]SymbolProbability, 0, len(types))
	for typeHash, vp := range types {
		logp := vp.Probability

		resolvedType := c.Types[typeHash]
		if resolvedType != nil {
			for _, attr := range observation.Attributes {
				var found bool
				for _, a := range resolvedType.Attributes {
					if a.Name == attr {
						logp += a.LogProbability
						found = true
						break
					}
				}
				if !found {
					logp += outlierAttrLogp
				}
			}
		} else {
			logp += float64(len(observation.Attributes)) * outlierAttrLogp
		}

		distr = append(distr, *vp)
		if logp > bestLogp {
			bestLogp = logp
			bestType = vp.Symbol
		}
	}

	// Convert distr to a proper pdf.
	normalize(distr, bestLogp)

	return Estimate{
		MostProbableType: bestType,
		Distribution:     distr,
	}
}

// returns map from type to a prior probability
func (c *Client) getTypes(observation Observation) map[pythonimports.Hash]*SymbolProbability {
	types := make(map[pythonimports.Hash]*SymbolProbability)
	if !observation.ReturnedFrom.Nil() {
		if f, found := c.Functions[observation.ReturnedFrom.PathHash()]; found {
			for _, entry := range f.ReturnType {
				types[entry.Symbol.PathHash()] = &SymbolProbability{
					Symbol:      entry.Symbol,
					Probability: entry.LogProbability,
				}
			}
			return types
		}
		return nil
	}

	if c.SymbolGraph == nil || c.DependencyGraph == nil {
		return types
	}

	// always include builtins
	imports := append(observation.Imports, "builtins")
	for _, imp := range imports {
		// for now the dependency graph only supports top level modules.
		if pos := strings.Index(imp, "."); pos > -1 {
			imp = imp[:pos]
		}

		pkg, found := c.DependencyGraph[imp]
		if !found {
			continue
		}

		// types from dependencies
		// note that if C depends on B and
		// B depends on A then A is included in the
		// dependencies for C, thus this does not need to
		// be recursive.
		names := append(pkg.Dependencies, pkg.Name)
		for _, name := range names {
			sym, err := c.SymbolGraph.PathSymbol(pythonimports.NewDottedPath(name))
			if err != nil {
				continue
			}
			children, err := c.SymbolGraph.Children(sym)
			if err != nil {
				continue
			}
			for _, child := range children {
				childSym, err := c.SymbolGraph.ChildSymbol(sym, child)
				if err != nil {
					continue
				}
				lp, found := c.TypePriors[childSym.PathHash()]
				if !found {
					continue
				}

				types[childSym.PathHash()] = &SymbolProbability{
					Symbol:      childSym,
					Probability: lp,
				}
			}
		}
	}

	return types
}

// convert distr to a proper pdf
func normalize(distr []SymbolProbability, bestLogp float64) {
	var total float64
	for i := range distr {
		exp := math.Exp(distr[i].Probability - bestLogp)
		total += exp
		distr[i].Probability = exp
	}

	for i := range distr {
		distr[i].Probability /= total
	}

	// sort from most to least probable
	sort.Slice(distr, func(i, j int) bool {
		return distr[i].Probability > distr[j].Probability
	})
}
