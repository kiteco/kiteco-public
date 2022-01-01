package pythonimports

import (
	"container/heap"
)

// An item is something we manage in a priority queue.
type item struct {
	node      *Node      // a node in the import graph
	shortname DottedPath // the shortest name found so far for this node
	index     int        // used for updating nodes that are already in the heap
	locked    bool       // shortname should not be changed because it is the canonical name for this node
}

// A nodeQueue implements heap.Interface and holds Items.
type nodeQueue []*item

func (q nodeQueue) Len() int { return len(q) }

func (q nodeQueue) Less(i, j int) bool {
	return q[i].shortname.Less(q[j].shortname)
}

func (q nodeQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *nodeQueue) Push(x interface{}) {
	n := len(*q)
	item := x.(*item)
	item.index = n
	*q = append(*q, item)
}

func (q *nodeQueue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

// ComputeAnyPaths computes a path for each node in the graph such that:
//  1) graph.Navigate(anypath[node]) returns the node.
//  2) S has no prefix P that resolves to a node that has a canonical name other than P
//
// WARNING: if you run this multiple times on the same graph, you will get different
// paths for the same node. In other words, this is a one-way map: each anypath
// corresponds to a single import graph node, but an import graph node corresponds to
// many anypaths (usually infinitely many). So if you have two anypaths and you want to
// know whether they represent the same node, it is _not_ sufficient to simply compare
// the strings -- you need to call Graph.Find with the two paths and then check
// for equality between the pointers.
//
// Orphan nodes not connected to the graph will not have entries in the returned map.
func ComputeAnyPaths(g *Graph) map[*Node]DottedPath {
	items := make(map[*Node]*item)
	queue := make(nodeQueue, 0, len(g.Nodes))

	// First lock in all the canonical names
	for i := range g.Nodes {
		node := &g.Nodes[i]
		if verifyCanonicalName(node, g) {
			it := &item{
				node:      node,
				shortname: node.CanonicalName,
				locked:    true,
			}
			items[node] = it
			queue = append(queue, it)
		}
	}

	// Add all the top-level nodes
	for pkg, node := range g.PkgToNode {
		if _, found := items[node]; !found {
			it := &item{
				node:      node,
				shortname: NewDottedPath(pkg),
			}
			items[node] = it
			queue = append(queue, it)
		}
	}
	heap.Init(&queue)

	explored := make(map[*Node]bool)
	return updateAndExtract(items, queue, explored)
}

func verifyCanonicalName(node *Node, graph *Graph) bool {
	if node.CanonicalName.Empty() {
		return false
	}
	resolved, err := graph.Navigate(node.CanonicalName)
	if err != nil {
		return false
	}
	return resolved == node
}

func updateAndExtract(items map[*Node]*item, queue nodeQueue, explored map[*Node]bool) map[*Node]DottedPath {
	// If the new shortname is shorter than the existing name for this node, or there is no
	// existing name for that node, then update the item and adjust the heap. Otherwise, ignore.
	offer := func(n *Node, newbase DottedPath, newattr string) {
		if explored[n] {
			return
		}

		nextname := newbase.WithTail(newattr)

		it, found := items[n]
		if !found {
			it = &item{
				node:      n,
				shortname: nextname,
			}
			items[n] = it
			heap.Push(&queue, it)
			return
		}

		if !it.locked && nextname.Less(it.shortname) {
			it.shortname = nextname
			heap.Fix(&queue, it.index)
		}
	}

	// Now update the graph one step at a time
	for len(queue) > 0 {
		top := heap.Pop(&queue).(*item)
		for attr, node := range top.node.Members {
			if node != nil {
				offer(node, top.shortname, attr)
			}
		}
	}

	// Extract the final list of short names
	anypaths := make(map[*Node]DottedPath)
	for node, item := range items {
		if item.index != -1 {
			// Should not be possible for any graph -- indicates an implementation error
			panic("there was an unconfirmed item left over")
		}
		anypaths[node] = item.shortname
	}
	return anypaths
}
