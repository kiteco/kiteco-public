package resources

import (
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/popularsignatures"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/argspec"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/kwargs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/returntypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/sigstats"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symgraph"
)

// Resource supports ser/des methods Encode & Decode
// We could use encoding.Marshaler/Unmarshaler here, but this is more general, as it supports streaming data
type Resource interface {
	Encode(io.Writer) error
	Decode(io.Reader) error
}

// Group bundles all resources for a given versioned package
type Group struct {
	// all resources should be indirect/reference types so that Load works correctly
	SymbolGraph       *symgraph.Graph
	ArgSpec           argspec.Entities
	PopularSignatures popularsignatures.Entities
	SignatureStats    sigstats.Entities
	Documentation     docs.Entities
	SymbolCounts      symbolcounts.Entities
	Kwargs            kwargs.Entities
	ReturnTypes       returntypes.Entities
}

// EmptyGroup creates an empty Group
func EmptyGroup() *Group {
	return &Group{
		SymbolGraph:       new(symgraph.Graph),
		ArgSpec:           make(argspec.Entities),
		PopularSignatures: make(popularsignatures.Entities),
		SignatureStats:    make(sigstats.Entities),
		Documentation:     make(docs.Entities),
		SymbolCounts:      make(symbolcounts.Entities),
		Kwargs:            make(kwargs.Entities),
		ReturnTypes:       make(returntypes.Entities),
	}
}

// NewGroup loads a Group for a given versioned package via the global resource manifest
func NewGroup(lg LocatorGroup) (*Group, error) {
	rg := EmptyGroup()

	err := lg.Load(*rg)
	if err != nil {
		return nil, err
	}

	return rg, nil
}
