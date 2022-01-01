package pythonimports

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// NodeStrings represents auxiliary strings not contained in the Node struct due to their large size
type NodeStrings struct {
	NodeID    int64  `json:"node_id"`
	Docstring string `json:"docstring"`
	Str       string `json:"str"`
	Repr      string `json:"repr"`
}

// GraphStrings is a map from node ID to the auxiliary strings for that node
type GraphStrings map[int64]*NodeStrings

// LoadGraphStrings loads axiliary strings for an import graph from a file
func LoadGraphStrings(path string) (GraphStrings, error) {
	out := make(GraphStrings)
	err := serialization.Decode(path, func(v *NodeStrings) {
		out[v.NodeID] = v
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load graph strings: %v", err)
	}
	return out, nil
}
