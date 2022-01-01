package pythonimports

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	defaultDataset = "s3://kite-data/type-inference-models/2016-11-04_17-54-37-PM"
	// DefaultImportGraph is the latest production import graph dataset.
	DefaultImportGraph = defaultDataset + "/import-graph.gob.gz"
	// SmallImportGraph is a subset of the full import graph that is intended for development.
	SmallImportGraph = defaultDataset + "/small-import-graph.gob.gz"
	// DefaultImportGraphStrings contains auxiliary strings (e.g. docstrings) for the import graph.
	DefaultImportGraphStrings = defaultDataset + "/import-graph-strings.json.gz"
	// DefaultImportGraphArgSpecs contains the arg specs for the import graph.
	DefaultImportGraphArgSpecs = defaultDataset + "/import-graph-arg-specs.json.gz"
	// DefaultTypeshedArgSpecs contains the arg specs from typeshed
	DefaultTypeshedArgSpecs = "s3://kite-data/typeshed-argspecs/2017-10-03_02-42-28-PM.json.gz"
	// DefaultImportGraphIndex contains the index for the graph
	DefaultImportGraphIndex = defaultDataset + "/graph-index.json.gz"
	// DefaultModuleAttributes contains the lookup table for attributes
	DefaultModuleAttributes = defaultDataset + "/module-attributes.json.gz"
)

var (
	// ErrNotFound is returned when graph.Navigate cannot find a node
	ErrNotFound = errors.New("node was not found in graph")
)

// Origin represents the origin of a node.
type Origin int

const (
	// GlobalGraph denotes that a node is in the global graph.
	GlobalGraph = iota
	// LocalGraph denotes that a node is in the local graph.
	LocalGraph
	// AnalyzerGraph denotes that a node is created during the analysis process.
	AnalyzerGraph
)

const rootName = "kiteroot065151"

// Graph is a collection of Nodes representing the Python import graph.
type Graph struct {
	Nodes     []Node
	PkgToNode map[string]*Node
	idToNode  map[int64]*Node

	// Root is the parent of every package level node and
	// has no parent itself.
	// This is an artificial node we add for convenience,
	// this node does not appear in the Nodes slice.
	Root *Node

	// Stores a possible path from the root for every node.
	AnyPaths map[*Node]DottedPath
}

// NewGraph constructs a new Graph from the provided path.
func NewGraph(path string) (*Graph, error) {
	nodes, err := LoadFlatGraph(path)
	if err != nil {
		return nil, err
	}
	return NewGraphFromNodes(nodes), nil
}

// NewEmptyGraph returns an empty graph
func NewEmptyGraph() *Graph {
	return &Graph{
		PkgToNode: make(map[string]*Node),
		idToNode:  make(map[int64]*Node),
		AnyPaths:  make(map[*Node]DottedPath),
		Root: &Node{
			NodeInfo: NodeInfo{
				Classification: Root,
				CanonicalName:  NewDottedPath(rootName),
			},
			Members: make(map[string]*Node),
		},
	}
}

// NewGraphFromNodes constructs a new graph from the provided nodes.
func NewGraphFromNodes(flatNodes []*FlatNode) *Graph {
	nodes, _ := InflateNodes(flatNodes, nil)

	// Compute pkgToNode
	pkgToNode := make(map[string]*Node)
	idToNode := make(map[int64]*Node)
	for i := range nodes {
		idToNode[nodes[i].ID] = &nodes[i]
		if len(nodes[i].CanonicalName.Parts) == 1 {
			pkgToNode[nodes[i].CanonicalName.Head()] = &nodes[i]
		}
	}

	graph := &Graph{
		Nodes:     nodes,
		PkgToNode: pkgToNode,
		idToNode:  idToNode,
		Root:      MakeRoot(pkgToNode),
	}
	graph.AnyPaths = ComputeAnyPaths(graph)
	return graph
}

// MakeRoot sets up a root node which is the parent of
// all package level nodes and has no parent. This is neccesary for
// offering autocomplete on package names without doing a Walk.
func MakeRoot(pkgToNode map[string]*Node) *Node {
	root := &Node{
		NodeInfo: NodeInfo{
			Classification: Root,
			CanonicalName:  NewDottedPath(rootName),
		},
		Members: make(map[string]*Node),
	}

	for pkg, node := range pkgToNode {
		root.Members[pkg] = node
	}

	return root
}

// CanonicalName returns the canonical name for the provided identifier.
func (i *Graph) CanonicalName(ident string) (string, error) {
	_, _, canon, err := i.find(ident)
	if err != nil {
		return "", err
	}
	return canon, err
}

// FindByID finds a node by ID.
func (i *Graph) FindByID(id int64) (*Node, bool) {
	n, found := i.idToNode[id]
	return n, found
}

// Find finds a node by identifier.
func (i *Graph) Find(ident string) (*Node, error) {
	var ok bool
	node := i.Root
	for ident != "" {
		var part string

		pos := strings.Index(ident, ".")
		if pos == -1 {
			part = ident
			ident = ""
		} else {
			part = ident[:pos]
			ident = ident[pos+1:]
		}

		if part == rootName {
			continue
		}

		node, ok = node.Attr(part)
		if !ok || node == nil {
			return nil, ErrNotFound
		}
	}
	return node, nil
}

// Navigate finds the node corresponding to a dotted path. It is unlike Find because:
//  - it never traverses .Type links
//  - it does not allocate anything internally (it does not compute the traversal path)
func (i *Graph) Navigate(p DottedPath) (*Node, error) {
	var ok bool
	node := i.Root
	for _, part := range p.Parts {
		if part == rootName {
			continue
		}

		node, ok = node.Members[part]
		if !ok || node == nil {
			return nil, ErrNotFound
		}
	}
	return node, nil
}

// WalkFn is called by Walk on each node encountered. If WalkFn returns true,
// Walk will recurse through the node's members.
type WalkFn func(name string, node *Node) bool

// Walk takes an identifier and a WalkFn, and performs a depth-first walk on all decendants
// of the identifier (including the indentifier).
func (i *Graph) Walk(ident string, walker WalkFn) error {
	node, path, _, err := i.find(ident)
	if err != nil {
		return err
	}
	if !walker(ident, node) {
		return nil
	}
	seen := make(map[*Node]struct{})
	for _, ptr := range path {
		seen[ptr] = struct{}{}
	}
	return i.recurseMembers(ident, node, walker, seen)
}

// WalkPrefix takes a prefix and walks the graph matching all nodes that have a path
// that starts with the provided prefix.
func (i *Graph) WalkPrefix(prefix string, walker WalkFn) error {
	seen := make(map[*Node]struct{})
	parts := strings.Split(prefix, ".")

	// If there is only one part, we want to complete package names
	if len(parts) == 1 {
		for name, node := range i.PkgToNode {
			if strings.HasPrefix(name, prefix) {
				if !walker(name, node) {
					continue
				}
				seen[node] = struct{}{}
				err := i.recurseMembers(name, node, walker, seen)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	var node *Node
	for idx, part := range parts {
		switch idx {
		case 0:
			// Find the package node
			curNode, exists := i.PkgToNode[part]
			if !exists {
				return fmt.Errorf("failed to find %s component in %s", part, prefix)
			}
			seen[curNode] = struct{}{}
			node = curNode
		case len(parts) - 1:
			// We are at the last node, start checking for matching prefixes and recurse
			members := i.sortedMembers(node)
			for _, memberName := range members {
				if strings.HasPrefix(memberName, part) {
					memberNode := node.Members[memberName]
					if memberNode == nil {
						continue
					}
					name := strings.Join(parts[:len(parts)-1], ".") + "." + memberName
					if !walker(name, memberNode) {
						continue
					}
					seen[memberNode] = struct{}{}
					err := i.recurseMembers(name, memberNode, walker, seen)
					if err != nil {
						return err
					}
				}
			}
		default:
			// We are walking down intermediate nodes of the provided prefix
			n, exists := node.Members[part]
			if !exists {
				return fmt.Errorf("failed to find %s component in %s", part, prefix)
			}
			if n == nil {
				return fmt.Errorf("node for %s in %s was nil", part, prefix)
			}
			seen[n] = struct{}{}
			node = n
		}
	}

	return nil
}

// --

// find walks the graph to find the node associated with the provided identifier. It also
// returns the path it took to get to the node, a preferred canonical name for the node,
// and an error if any part of the node traversal fails.
func (i *Graph) find(ident string) (*Node, []*Node, string, error) {
	var node *Node

	var path []*Node
	var canon string
	var pos int
	for {
		part := ident[pos:]
		offs := strings.Index(part, ".")
		if offs != -1 {
			part = part[:offs]
		}
		var exists bool
		var curNode *Node
		if pos == 0 {
			curNode, exists = i.PkgToNode[part]
			if !exists {
				return node, path, canon, fmt.Errorf("failed to find %s component in %s", part, ident)
			}
			if curNode == nil {
				return node, path, canon, fmt.Errorf("node for %s in %s was nil", part, ident)
			}
		} else {
			n, exists := node.Members[part]
			if !exists {
				return node, path, canon, fmt.Errorf("failed to find %s component in %s", part, ident)
			}
			if n == nil {
				return node, path, canon, fmt.Errorf("node for %s in %s was nil", part, ident)
			}
			curNode = n
		}

		node = curNode
		path = append(path, node)
		if !node.CanonicalName.Empty() {
			canon = node.CanonicalName.String()
		} else if node.Type != nil && !node.Type.CanonicalName.Empty() && offs != -1 {
			canon = node.Type.CanonicalName.String()
		} else if canon == "" {
			canon = part
		} else {
			canon = canon + "." + part
		}

		if offs == -1 {
			break
		}
		pos += offs + 1
	}

	return node, path, canon, nil
}

func (i *Graph) recurseMembers(prefix string, node *Node, walker WalkFn, seen map[*Node]struct{}) error {
	members := i.sortedMembers(node)
	for _, memberName := range members {
		n := node.Members[memberName]
		if n == nil {
			continue
		}
		if _, exists := seen[n]; exists {
			continue
		}
		name := prefix + "." + memberName
		seen[n] = struct{}{}
		if !walker(name, n) {
			continue
		}
		err := i.recurseMembers(name, n, walker, seen)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Graph) sortedMembers(node *Node) []string {
	var names []string
	for name := range node.Members {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
