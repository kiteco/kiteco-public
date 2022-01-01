package symgraph

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/stringutil"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// TopLevelNotFound is an error
type TopLevelNotFound string

// Error implements error
func (e TopLevelNotFound) Error() string {
	return fmt.Sprintf("top-level package not found for path %s", string(e))
}

// AttributeNotFound is an error
type AttributeNotFound string

// Error implements error
func (e AttributeNotFound) Error() string {
	return fmt.Sprintf("attribute %s not found", string(e))
}

// ExternalEncountered is an Error returned when an external reference is encountered during canonicalization
type ExternalEncountered struct {
	// External is the external reference encountered
	External pythonimports.DottedPath
	// Rest contains the remaining path components unprocessed by canonicalization when the reference was encountered
	Rest []string
}

// WithRest returns the extension of path e.External with the path components in e.Rest
func (e ExternalEncountered) WithRest() pythonimports.DottedPath {
	return e.External.WithTail(e.Rest...)
}

// Error implements error
func (e ExternalEncountered) Error() string {
	return fmt.Sprintf("encountered external symbol reference %s with remaining path %s", e.External.String(), strings.Join(e.Rest, "."))
}

// Ref is a efficient reference to a graph node
type Ref struct {
	TopLevel string
	Internal int
}

func (g Graph) refNode(ref Ref) *Node {
	shard := g[ref.TopLevel]
	if int(ref.Internal) >= len(shard) {
		rollbar.Error(errors.New("called symgraph.Graph.refNode with invalid Ref"), fmt.Sprintf("%s:%d", ref.TopLevel, int(ref.Internal)))
		return nil
	}
	return &shard[int(ref.Internal)]
}

// TopLevel looks up and returns a Ref to the node for a given toplevel,
// potentially returning TopLevelNotFound
func (g Graph) TopLevel(toplevel string) (Ref, error) {
	if _, ok := g[toplevel]; !ok {
		return Ref{}, TopLevelNotFound(toplevel)
	}
	return Ref{TopLevel: toplevel, Internal: 0}, nil
}

// Lookup looks up the Ref for the node corresponding to a given DottedPath,
// potentially returning TopLevelNotFound, AttributeNotFound, or ExternalEncountered
func (g Graph) Lookup(p pythonimports.DottedPath) (Ref, error) {
	cur, err := g.TopLevel(p.Head())
	if err != nil {
		return Ref{}, err
	}

	for i, part := range p.Parts[1:] {
		cur, err = g.Child(cur, part)
		if err == nil {
			continue
		}
		if extErr, ok := err.(ExternalEncountered); ok {
			// p.Parts[i+1] == part, which is included in the extErr.External
			extErr.Rest = p.Parts[i+2:]
			return Ref{}, extErr
		}
		return Ref{}, err
	}

	return cur, nil
}

// Canonical returns the canonical path for the given Ref
func (g Graph) Canonical(ref Ref) pythonimports.DottedPath {
	n := g.refNode(ref)
	if n == nil {
		return pythonimports.DottedPath{}
	}
	return n.Canonical.Cast()
}

// Children returns a list of child attributes for the given Ref
func (g Graph) Children(ref Ref) []string {
	n := g.refNode(ref)
	if n == nil {
		return nil
	}

	var out []string
	for c := range n.Children {
		out = append(out, stringutil.FromUint64(c))
	}
	return out
}

func refOfNodeRef(tl string, n NodeRef) (Ref, error) {
	switch {
	case !n.External.Cast().Empty():
		return Ref{}, ExternalEncountered{External: n.External.Cast()}
	case n.Internal >= 0:
		return Ref{TopLevel: tl, Internal: n.Internal}, nil
	default:
		return Ref{}, errors.New("nil symbol reference")
	}
}

// Child looks up a Ref to the child for the given parent Ref,
// potentially returning AttributeNotFound or ExternalEncountered
func (g Graph) Child(parent Ref, attr string) (Ref, error) {
	n := g.refNode(parent)
	if n == nil {
		return Ref{}, errors.New("called symgraph.Graph.Child with invalid Ref; this should never happen")
	}

	child, ok := n.Children[stringutil.Hash64(attr)]
	if !ok {
		return Ref{}, AttributeNotFound(attr)
	}

	return refOfNodeRef(parent.TopLevel, child)
}

// Kind looks up the Kind from a Ref
func (g Graph) Kind(ref Ref) keytypes.Kind {
	n := g.refNode(ref)
	if n == nil {
		return keytypes.NoneKind
	}
	return keytypes.Kind(n.Kind)
}

// Type looks up a Ref to the type of the given node Ref
func (g Graph) Type(ref Ref) (Ref, error) {
	n := g.refNode(ref)
	if n == nil {
		return Ref{}, errors.New("called symgraph.Graph.Child with invalid Ref; this should never happen")
	}
	if n.Type == nil {
		return Ref{}, errors.New("nil symbol reference")
	}
	return refOfNodeRef(ref.TopLevel, *n.Type)
}

// NumBases returns the number of base classes for the given Ref
func (g Graph) NumBases(ref Ref) int {
	n := g.refNode(ref)
	if n == nil {
		return 0
	}
	return len(n.Bases)
}

// GetBase gets a Ref to the ith base class for the given Ref
func (g Graph) GetBase(ref Ref, i int) (Ref, error) {
	n := g.refNode(ref)
	if n == nil {
		return Ref{}, errors.New("called symgraph.Graph.Child with invalid Ref; this should never happen")
	}
	if i >= len(n.Bases) {
		return Ref{}, errors.New("GetBase index out of bounds")
	}
	return refOfNodeRef(ref.TopLevel, n.Bases[i])
}
