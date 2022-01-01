package pythondocs

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"

// ParentMap is a map from import graph nodes to their parents
type ParentMap map[*pythonimports.Node][]*pythonimports.Node

// BuildParentMap constructs a map from nodes to their parents
func BuildParentMap(nodes []pythonimports.Node) ParentMap {
	// Map each node to its immediate parents
	parents := make(ParentMap)
	for i := range nodes {
		for _, child := range nodes[i].Members {
			if child == nil {
				continue
			}
			parents[child] = append(parents[child], &nodes[i])
		}
	}
	return parents
}
