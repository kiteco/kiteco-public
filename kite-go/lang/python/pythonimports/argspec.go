package pythonimports

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// Arg represents an argument in a python function definition. DefaultType is the fully
// qualified name of the type of the argument's default value, or the empty string if
// the argument has no default value, Types are the types of the argument populated by the pythonskeletons.UpdateArgSpecs method.
type Arg struct {
	Name        string `json:"name"`
	DefaultType string `json:"default_type"`
	// HasDefaultValue is deprecated; remove after removing from specialized argspec parser TODO(naman)
	HasDefaultValue bool     `json:"has_default_value"`
	KeywordOnly     bool     `json:"keyword_only"`
	DefaultValue    string   `json:"default_value"`
	Types           []string `json:"types"` // TODO(juan): make this a slice of *pythonimports.Node and omit from json?
}

// Required argument
func (a Arg) Required() bool {
	return a.DefaultValue == ""
}

// ArgSpec represents the argument list for a python function. Vararg is the name of
// the "*args" argument, or empty string if there was no such argument. Kwarg is the
// name of the "**kwargs" argument, or empty string if there was no such argument.
type ArgSpec struct {
	NodeID int64  `json:"node_id"`
	Args   []Arg  `json:"args"`
	Vararg string `json:"vararg"`
	Kwarg  string `json:"kwarg"`
}

// IsFirstArgReceiver tests if the first arg is named self or cls
// Return false if there is no arg in the spec
func (as ArgSpec) IsFirstArgReceiver() bool {
	if len(as.Args) < 1 {
		return false
	}
	if as.Args[0].Name == "self" || as.Args[0].Name == "cls" {
		return true
	}
	return false
}

// NonReceiverArgs return a slice of args with the receiver removed (if present)
// Is considered as receiver the first arg if its name is cls or self
func (as ArgSpec) NonReceiverArgs() []Arg {
	if len(as.Args) > 0 && as.IsFirstArgReceiver() {
		return as.Args[1:]
	}
	return as.Args
}

// PositionalCount returns the number of positional args possible for this function
// Warning, this function doesn't account for vararg, that need to be checked independently
func (as ArgSpec) PositionalCount() int {
	var count int
	for _, a := range as.NonReceiverArgs() {
		if a.KeywordOnly {
			break
		}
		count++
	}
	return count
}

// NamedArgSpec represents an ArgSpec along with the canonical name of the function/type,
// used for (un)marshalling TypeshedArgSpecs
type NamedArgSpec struct {
	ArgSpec       ArgSpec `json:"argspec"`
	CanonicalName string  `json:"canonical_name"`
}

// ArgSpecs is a map from node id to the ArgSpec for that node.
type ArgSpecs struct {
	ImportGraphArgSpecs map[int64]*ArgSpec
	TypeshedArgSpecs    map[string]*ArgSpec
}

// Find coalesces the import graph and typeshed-sourced data to find the ArgSpec for a given node
func (s *ArgSpecs) Find(n *Node) *ArgSpec {
	if n == nil {
		return nil
	}

	if argspec, exists := s.ImportGraphArgSpecs[n.ID]; exists {
		return argspec
	}
	if argspec, exists := s.TypeshedArgSpecs[n.CanonicalName.String()]; exists {
		argspec.NodeID = n.ID // fill in the NodeID, because why not?
		return argspec
	}
	return nil
}

// LoadArgSpecs loads argument specs for an import graph from file.
func LoadArgSpecs(graph *Graph, importGraphPath, typeshedPath string) (*ArgSpecs, error) {
	out := ArgSpecs{
		ImportGraphArgSpecs: make(map[int64]*ArgSpec),
		TypeshedArgSpecs:    make(map[string]*ArgSpec),
	}

	err := serialization.Decode(importGraphPath, func(v *ArgSpec) {
		out.ImportGraphArgSpecs[v.NodeID] = v
	})
	if err != nil {
		return &out, fmt.Errorf("failed to load import graph arg specs: %v", err)
	}

	err = serialization.Decode(typeshedPath, func(v *NamedArgSpec) {
		out.TypeshedArgSpecs[v.CanonicalName] = &v.ArgSpec
	})
	if err != nil {
		return &out, fmt.Errorf("failed to load typeshed arg specs: %v", err)
	}

	// associate type nodes with arg spec from __init__ method if it exists
	// for typeshed argspecs, we assume this is handled at generation time
	for _, node := range graph.Nodes {
		if node.Classification != Type {
			continue
		}

		if member := node.Members["__init__"]; member != nil {
			if as := out.ImportGraphArgSpecs[member.ID]; as != nil {
				out.ImportGraphArgSpecs[node.ID] = as
			}
		}
	}
	return &out, nil
}
