//go:generate msgp -marshal=false

package symgraph

import (
	"unsafe"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// Kind is a structural analog of keytypes.Kind
type Kind int

//msgp:tuple DottedPath

// DottedPath is a structural analog of pythonimports.DottedPath
type DottedPath struct {
	Hash  uint64   `msg:"hash"`
	Parts []String `msg:"parts"`
}

// CastDottedPath uses unsafe to cast a pythonimport.DottedPath to a DottedPath
func CastDottedPath(p pythonimports.DottedPath) DottedPath {
	return *(*DottedPath)(unsafe.Pointer(&p))
}

// Cast uses unsafe to cast this to a pythonimport.DottedPath
func (p DottedPath) Cast() pythonimports.DottedPath {
	return *(*pythonimports.DottedPath)(unsafe.Pointer(&p))
}

//msgp:tuple NodeRef

// NodeRef is an indirect reference to a Node, internal or external
type NodeRef struct {
	Internal int        `msg:"internal"`
	External DottedPath `msg:"external"`
}

//msgp:tuple Node

// Node represents a node internal to a toplevel package
// It is a structural analog of symgraph.Node
type Node struct {
	Canonical DottedPath         `msg:"canonical"`
	junk      map[string]NodeRef // TODO(naman) remove once old data is gone, and we generate this struct directly
	Children  ChildMap           `msg:"children"`
	Type      *NodeRef           `msg:"type"`
	Bases     []NodeRef          `msg:"bases"`
	Kind      Kind               `msg:"kind"`
}

// Graph is a structural analog of symgraph.Graph
type Graph map[string][]Node
