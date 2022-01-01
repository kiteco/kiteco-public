package pythonstatic

import (
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	numProposalsToKeep = 5
	numItersPropagate  = 1
)

// Capability represents a capability for a symbol.
type Capability struct {
	Attr string
}

// capabilityDelegate handles recording capabilities and edges in the symbol graph
type capabilityDelegate struct {
	capabilities     map[*pythontype.Symbol][]Capability
	forwardNeighbors map[*pythontype.Symbol][]*pythontype.Symbol
}

func newCapabilityDelegate() *capabilityDelegate {
	return &capabilityDelegate{
		capabilities:     make(map[*pythontype.Symbol][]Capability),
		forwardNeighbors: make(map[*pythontype.Symbol][]*pythontype.Symbol),
	}
}

// RecordCapability records the provided Capability `cap` on the Symbol `s`.
func (c *capabilityDelegate) RecordCapability(s *pythontype.Symbol, cap Capability) {
	c.capabilities[s] = append(c.capabilities[s], cap)
}

// RecordEdge records the directed edge from `start` to `end`.
func (c *capabilityDelegate) RecordEdge(start *pythontype.Symbol, end *pythontype.Symbol) {
	c.forwardNeighbors[start] = append(c.forwardNeighbors[start], end)
}

// ForwardGraph returns the forward symbol graph.
func (c *capabilityDelegate) ForwardGraph() map[*pythontype.Symbol][]*pythontype.Symbol {
	deduped := make(map[*pythontype.Symbol][]*pythontype.Symbol)
	for s, ns := range c.forwardNeighbors {
		seen := make(map[*pythontype.Symbol]bool)
		var nns []*pythontype.Symbol
		for _, n := range ns {
			if seen[n] {
				continue
			}
			seen[n] = true
			nns = append(nns, n)
		}
		deduped[s] = nns
	}
	return deduped
}

// BackwardGraph returns the backward symbol graph.
func (c *capabilityDelegate) BackwardGraph() map[*pythontype.Symbol][]*pythontype.Symbol {
	// build backward graph
	bg := make(map[*pythontype.Symbol][]*pythontype.Symbol)
	for s, ns := range c.forwardNeighbors {
		for _, n := range ns {
			bg[n] = append(bg[n], s)
		}
	}

	// dedupe
	deduped := make(map[*pythontype.Symbol][]*pythontype.Symbol)
	for s, ns := range bg {
		seen := make(map[*pythontype.Symbol]bool)
		var nns []*pythontype.Symbol
		for _, n := range ns {
			if seen[n] {
				continue
			}
			seen[n] = true
			nns = append(nns, n)
		}
		deduped[s] = nns
	}
	return deduped
}

// Capabilities returns the set of overserved capabilities.
func (c *capabilityDelegate) Capabilities() map[*pythontype.Symbol][]Capability {
	return c.capabilities
}

type weightedClass struct {
	Class  *pythontype.SourceClass
	Weight float64
}

type byWeight []weightedClass

func (b byWeight) Len() int           { return len(b) }
func (b byWeight) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byWeight) Less(i, j int) bool { return b[i].Weight < b[j].Weight }

type scores map[string]float64

type estimator struct {
	assembly *Assembly
	// inverse document frequencies for each attr
	idfs scores
	// attr frequency scores for each symbol
	tfs map[*pythontype.Symbol]scores
	// classes that support a given capability
	classByCap map[string]map[*pythontype.SourceClass]struct{}
	// trace writer for tracing output
	tw io.Writer

	// forward symbol graph
	forwardGraph map[*pythontype.Symbol][]*pythontype.Symbol
	// backward symbol graph
	backwardGraph map[*pythontype.Symbol][]*pythontype.Symbol
	// recorded capabilities
	capabilities map[*pythontype.Symbol][]Capability
}

// This code is not accessed anymore
// The CapabilityDelegate is now mostly used to restrict union values
// To reactivate the estimator, one should add
// // newEstimator(b.assembly, b.helpers.CapabilityDelegate, b.helpers.TraceWriter).EstimateValues()
// After doing the propagation in Assembly.go
func newEstimator(assembly *Assembly, cd *capabilityDelegate, tw io.Writer) estimator {
	// treat each slice of capabilities as a document
	idfs := inverseDocFreqs(cd.capabilities)

	// for each symbol store the frequencies of the capabilities
	// that the symbol exhibits
	tfs := make(map[*pythontype.Symbol]scores)

	// for each capabilitiy store the classes that
	// support that capability
	classByCap := make(map[string]map[*pythontype.SourceClass]struct{})

	e := estimator{
		assembly:      assembly,
		idfs:          idfs,
		tfs:           tfs,
		classByCap:    classByCap,
		tw:            tw,
		forwardGraph:  cd.ForwardGraph(),
		backwardGraph: cd.BackwardGraph(),
		capabilities:  cd.Capabilities(),
	}

	// trace graph and capabilities, we check that
	// tw is not nil here to avoid doing extra
	// iteration over graph and capabilities
	if e.tw != nil {
		e.trace("### Symbol Graph ###\n")
		for s, ns := range e.forwardGraph {
			e.trace("  %s ->\n", s.Name.String())
			for _, n := range ns {
				e.trace("    %s\n", n.Name.String())
			}
		}

		e.trace("### Capabilities ###")
		for s, caps := range e.capabilities {
			e.trace("  %s:\n", s.Name.String())
			for _, c := range caps {
				e.trace("    %s\n", c.Attr)
			}
		}
	}

	return e
}

func (e estimator) trace(fmtstr string, args ...interface{}) {
	if e.tw != nil {
		fmt.Fprintf(e.tw, fmtstr, args...)
	}
}

// EstimateValues estimates the values for symbols based
// on the capabilities they exhibited and the symbol
// contstraint graph.
// (Not called anymore, see newEstimator comment)
func (e estimator) EstimateValues() {
	e.trace("### Estimating Values ###\n")
	for iter := 0; iter < numItersPropagate; iter++ {
		for start, neighbors := range e.forwardGraph {
			e.propagateAlongEdge(start, start)
			for _, neighbor := range neighbors {
				e.propagateAlongEdge(start, neighbor)
			}
		}

		for start, neighbors := range e.backwardGraph {
			e.propagateAlongEdge(start, start)
			for _, neighbor := range neighbors {
				e.propagateAlongEdge(start, neighbor)
			}
		}
	}

	// update nodes with no incoming our outgoing edges,
	// only need to do this once
	for s := range e.capabilities {
		if _, found := e.forwardGraph[s]; found {
			continue
		}

		if _, found := e.backwardGraph[s]; found {
			continue
		}
		e.propagateAlongEdge(s, s)
	}
}

// propagateAlongEdge performs type checking and type inference
// on the value associated with the symbol `end`, based on the capabilities
// expressed by the symbol `start`. In particular, this method performs two steps:
//   1) Type inference -- propose types for `end`'s `Value` based on the
//   recorded capabilities for the symbol `start` using the set of user defined classes as "queries",
//   and treating the set of capabilities recorded for the node as the"document".
//   2) Type checking -- prune types from the `end`'s `Value`
//   that do not exhibit all of the recorded capabilities for the symbol `start`.
func (e estimator) propagateAlongEdge(start, end *pythontype.Symbol) {
	capabilities := e.capabilities[start]
	if len(capabilities) == 0 {
		return
	}

	e.trace("\n### Propgating along edge %s -> %s ###\n", start.Name.String(), end.Name.String())

	// Type Inference: propose new classes by treating the capabilities as a document
	// and each class as a query
	tfs := e.tfs[start]
	if tfs == nil {
		tfs = termFreqs(capabilities)
		e.tfs[start] = tfs
	}

	props := make(map[*pythontype.SourceClass]float64)
	for _, cap := range capabilities {
		if isCommonAttr(cap.Attr) {
			continue
		}

		classes, found := e.classByCap[cap.Attr]
		if !found {
			classes = make(map[*pythontype.SourceClass]struct{})
			e.classByCap[cap.Attr] = classes
			for _, class := range e.assembly.Classes {
				if _, err := pythontype.Attr(kitectx.TODO(), class, cap.Attr); err == nil {
					classes[class] = struct{}{}
				}
			}
		}

		for class := range classes {
			props[class] += e.idfs[cap.Attr] * tfs[cap.Attr]
		}
	}

	var wcs []weightedClass
	for class, weight := range props {
		wcs = append(wcs, weightedClass{
			Weight: weight,
			Class:  class,
		})
	}

	sort.Sort(sort.Reverse(byWeight(wcs)))
	if len(wcs) > numProposalsToKeep {
		wcs = wcs[:numProposalsToKeep]
	}

	proposals := pythontype.Disjuncts(kitectx.TODO(), end.Value)
	var emptyArgs pythontype.Args
	for _, wc := range wcs {
		e.trace("  proposing class %s (%f)\n", wc.Class.Address().String(), wc.Weight)
		proposals = append(proposals, wc.Class.Call(emptyArgs))
	}

	// Type Checking: prune values that do not exhibit the provided capabilities
	var accepted []pythontype.Value
	for _, v := range proposals {
		if supportsCapabilities(v, capabilities) {
			accepted = append(accepted, v)
			e.trace("  accepted value %v\n", v)
		} else {
			e.trace("  removing value %v\n", v)
		}
	}

	if len(accepted) > 0 {
		end.Value = pythontype.Unite(kitectx.TODO(), accepted...)
	}
}

func supportsCapabilities(v pythontype.Value, caps []Capability) bool {
	for _, c := range caps {
		if _, err := pythontype.Attr(kitectx.TODO(), v, c.Attr); err != nil {
			return false
		}
	}
	return true
}

func termFreqs(caps []Capability) map[string]float64 {
	tfs := make(map[string]float64)
	for _, cap := range caps {
		tfs[cap.Attr] += 1.
	}
	return tfs
}

func inverseDocFreqs(capabilities map[*pythontype.Symbol][]Capability) scores {
	dfs := make(map[string]int64)
	idfs := make(scores)
	logLenCaps := math.Log(float64(len(capabilities)))
	for _, caps := range capabilities {
		seen := make(map[string]bool)
		for _, cap := range caps {
			if seen[cap.Attr] {
				continue
			}
			seen[cap.Attr] = true
			dfs[cap.Attr]++
			idfs[cap.Attr] = logLenCaps - math.Log(float64(dfs[cap.Attr]))
		}
	}
	return idfs
}

func isCommonAttr(attr string) bool {
	switch attr {
	case "__module__":
		return true
	case "__base__":
		return true
	case "__itemsize__":
		return true
	case "__str__":
		return true
	case "__reduce__":
		return true
	case "__weakrefoffset__":
		return true
	case "__dict__":
		return true
	case "__sizeof__":
		return true
	case "__lt__":
		return true
	case "__init__":
		return true
	case "__setattr__":
		return true
	case "__reduce_ex__":
		return true
	case "__subclasses__":
		return true
	case "__new__":
		return true
	case "__format__":
		return true
	case "__class__":
		return true
	case "__mro__":
		return true
	case "__bases__":
		return true
	case "__dictoffset__":
		return true
	case "__call__":
		return true
	case "__doc__":
		return true
	case "__abstractmethods__":
		return true
	case "__ne__":
		return true
	case "__getattribute__":
		return true
	case "__instancecheck__":
		return true
	case "__subclasscheck__":
		return true
	case "__subclasshook__":
		return true
	case "__gt__":
		return true
	case "__name__":
		return true
	case "__eq__":
		return true
	case "mro":
		return true
	case "__basicsize__":
		return true
	case "__flags__":
		return true
	case "__delattr__":
		return true
	case "__le__":
		return true
	case "__repr__":
		return true
	case "__hash__":
		return true
	case "__ge__":
		return true
	default:
		return false
	}
}
