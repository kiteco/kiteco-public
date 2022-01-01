package main

import (
	"log"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/rawgraph/types"
)

// navigate navigates a DottedPath, returning the corresponding NodeData for that path
func navigate(g types.Graph, rootID types.NodeID, path pythonimports.DottedPath) *types.NodeData {
	// assume path.HasPrefix(root.CanonicalName)
	root := g[rootID]
	cur := root
	for _, part := range path.Parts[strings.Count(root.CanonicalName, ".")+1:] {
		if cur == nil {
			return nil
		}

		if childID, ok := cur.Children[part]; ok {
			cur = g[childID] // nil if not found, which gets caught in the next iteration
		} else {
			// non-existent or external child
			cur = nil
		}
	}
	return cur
}

// validate explores & validates the graph, logging and fixing inconsistencies.
// The following properties will hold of the graph as a post-condition:
//
// 1) All nodes are reachable (via a path of attribute/child lookups)
// 2) All children/attributes point to valid nodes
// 3) All canonical names (paths) are valid, in the following sense:
//    a) the canonical name of a node is navigable and resolves to the same node
//    b) the predecessor of a node's canonical name is also a canonical name if it is navigable (otherwise, it must be the root node)
// and logging all such discovered inconsistencies.
func validateGraph(g types.Graph, name types.TopLevelName, rootID types.NodeID) {
	// All canonical paths to have the following properties
	//  1) navigate(graph, anypath[node]) returns the node.
	//  2) S has no prefix P that resolves to a node that has a canonical name other than P

	// trackedNode tracks a node along with a valid path satisfying properties 3a & 3b
	// which can be used if the pre-existing canonical path fails to satisfy 3a
	type trackedNode struct {
		path string
		node *types.NodeData
	}

	missingNodeStats := make(map[string]uint32)

	// getChildrenSorted collects all the internal children of a given trackedNode in sorted order
	getChildrenSorted := func(x trackedNode) []trackedNode {
		var keys []string
		for k := range x.node.Children {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var out []trackedNode
		for _, k := range keys {
			childID := x.node.Children[k]
			childNode := g[childID]
			if childNode == nil {
				missingNodeStats[x.node.Classification]++
				delete(x.node.Children, k)
				// these are typically due to members that are added and subsequently skipped during exploration
				log.Printf("[WARN] cannot locate node %d, child %s of %s %s (%d)\n", childID, k, x.node.Classification, x.node.CanonicalName, x.node.ID)
				continue
			}

			out = append(out, trackedNode{
				// assume x.node.CanonicalName is already valid; then this path satisfies 3b by construction
				path: x.node.CanonicalName + "." + k,
				node: childNode,
			})
		}

		return out
	}

	root := g[rootID] // note that this node may have an empy canonical name! we explicitly use string(name) below

	// Breadth-first search so that when we fix canonical paths, we take the lexicographically smallest
	visited := make(map[*types.NodeData]struct{})
	q := []trackedNode{trackedNode{path: string(name), node: root}}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]

		if _, ok := visited[cur.node]; ok {
			continue
		}
		visited[cur.node] = struct{}{}

		// skip external nodes
		canonicalPath := pythonimports.NewDottedPath(cur.node.CanonicalName)
		if !canonicalPath.Empty() && !canonicalPath.HasPrefix(string(name)) {
			if cur.node.Reference == "" {
				log.Printf("[SEVERE] node %d with external canonical path %s is not marked as reference", cur.node.ID, cur.node.CanonicalName)
				cur.node.Reference = cur.node.CanonicalName // overwrite the reference
			}
			continue
		}

		// try to resolve the canonical path
		var resolved *types.NodeData
		if !canonicalPath.Empty() {
			resolved = navigate(g, rootID, canonicalPath)
		}
		if resolved == nil {
			// Fix the canonical path (condition 3a), since the one from exploration is unresolvable
			// This is expected due to deficiencies in runtime exploration
			log.Printf("[WARN] replacing unresolvable canonical path %s with %s\n", cur.node.CanonicalName, cur.path)
			cur.node.CanonicalName = cur.path // cur.path satisfies condition 3b by construction
		} else if resolved != cur.node {
			// The canonical path resolved to the wrong node, so fix it (condition 3a).
			// This is usually a hard-to-handle edge case.
			log.Printf("[SEVERE] canonical path %s resolves to incorrect node (replacing with %s)\n", cur.node.CanonicalName, cur.path)
			cur.node.CanonicalName = cur.path // cur.path satisfies condition 3b by construction
		} else { // otherwise, condition 3a already holds, in which case check condition 3b
			if !canonicalPath.Equals(string(name)) {
				predPath := canonicalPath.Predecessor()
				predNode := navigate(g, rootID, predPath)
				if !predPath.Equals(predNode.CanonicalName) {
					fixed := predNode.CanonicalName + "." + canonicalPath.Last()
					log.Printf("[WARN] parent of node with path %s has differing canonical name (replacing with %s)\n", cur.node.CanonicalName, fixed)
					cur.node.CanonicalName = fixed
				}
			}
			// otherwise, we're at the root node, so nothing to check
		}

		// add all children to queue in sorted order
		q = append(q, getChildrenSorted(cur)...)
	}

	for cl, cnt := range missingNodeStats {
		log.Printf("[STATS] %d member node IDs missing for nodes with classification %s\n", cnt, cl)
	}

	orphanedCount := 0
	for i, node := range g {
		if node.Reference != "" {
			// external/reference nodes should be kept around (i.e. never considered orphaned),
			// since they might be used as types or base classes
			continue
		}
		if _, ok := visited[node]; !ok {
			// these are likely types or base classes that aren't reachable via attribute lookups
			// i.e. types/classes generated inside of a function body
			log.Printf("[WARN] deleting orphaned node %s\n", node)
			delete(g, i)
			orphanedCount++
		}
	}
	if orphanedCount > 0 {
		log.Printf("[WARN] %d nodes are orphaned", orphanedCount)
	}
}

func validateData(d *types.ExplorationData) {
	for name, g := range d.TopLevels {
		if strings.Contains(string(name), ".") {
			log.Fatalf("[FATAL] invalid top-level name (contains dot) %s\n", string(name))
		}

		log.Printf("[INFO] validating and fixing top-level %s\n", name)
		validateGraph(g, name, d.RootIDs[name])
	}
}
