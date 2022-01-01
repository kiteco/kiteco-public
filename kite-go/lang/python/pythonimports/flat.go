package pythonimports

import (
	"log"

	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// A FlatMember is an attribute/node ID pair
type FlatMember struct {
	Attr   string
	NodeID int64
}

// A FlatNode represents information associated with an entry in the Python import graph, using
// IDs to reference other nodes so that the structure is acyclic for serialization.
type FlatNode struct {
	NodeInfo
	TypeID  int64        `json:"type_id"`
	Members []FlatMember `json:"members"`
	Bases   []int64
}

// LoadFlatGraph loads a graph from the provided path
func LoadFlatGraph(path string) ([]*FlatNode, error) {
	var nodes []*FlatNode
	err := serialization.Decode(path, func(node *FlatNode) {
		if node.Classification == Function {
			node.Members = node.Members[:0]
		}
		for i, part := range node.CanonicalName.Parts {
			node.CanonicalName.Parts[i] = intern(part)
		}
		for i := range node.Members {
			node.Members[i].Attr = intern(node.Members[i].Attr)
		}
		nodes = append(nodes, node)
	})
	log.Printf("interning reduced %d strings to %d", numInternHits+len(interns), len(interns))
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// FlattenNodes converts a list of nodes pointers to a flat graph. It complains
// if any of the input nodes reference a node that is not in the input list.
func FlattenNodes(nodes []Node) ([]*FlatNode, map[*Node]int64) {
	idmap := make(map[*Node]int64)
	for i := range nodes {
		idmap[&nodes[i]] = int64(i)
	}
	idmap[nil] = -1

	// construct the forwarding function
	forward := func(n *Node) int64 {
		if id, found := idmap[n]; found {
			return id
		}
		id := int64(len(idmap))
		idmap[n] = id
		return id
	}

	var out []*FlatNode
	for i := range nodes {
		flat := &FlatNode{
			NodeInfo: nodes[i].NodeInfo,
			TypeID:   forward(nodes[i].Type),
		}
		flat.ID = forward(&nodes[i])
		for attr, child := range nodes[i].Members {
			flat.Members = append(flat.Members, FlatMember{
				Attr:   attr,
				NodeID: forward(child),
			})
		}
		for _, base := range nodes[i].Bases {
			flat.Bases = append(flat.Bases, forward(base))
		}
		out = append(out, flat)
	}
	return out, idmap
}

// InflateNodes converts FlatNodes to Nodes
func InflateNodes(flatNodes []*FlatNode, nodesByID map[int64]*Node) ([]Node, map[int64]*Node) {
	if nodesByID == nil {
		nodesByID = make(map[int64]*Node, len(flatNodes))
	}
	nodes := make([]Node, len(flatNodes))
	for i, flatNode := range flatNodes {
		if other, dup := nodesByID[flatNode.ID]; dup {
			log.Printf("Warning: InflateNodes got duplicate node with ID=%d ('%s' and '%s')",
				flatNode.ID, flatNode.CanonicalName.String(), other.CanonicalName.String())
		}
		nodesByID[flatNode.ID] = &nodes[i]
	}

	// Convert IDs to pointers
	for i, flatNode := range flatNodes {
		nodes[i].Type = nodesByID[flatNode.TypeID]
		nodes[i].NodeInfo = flatNode.NodeInfo
		nodes[i].Members = make(map[string]*Node, len(flatNode.Members))
		for _, member := range flatNode.Members {
			nodes[i].Members[member.Attr] = nodesByID[member.NodeID]
		}
		for _, baseID := range flatNode.Bases {
			if base := nodesByID[baseID]; base != nil {
				nodes[i].Bases = append(nodes[i].Bases, base)
			}
		}
	}
	return nodes, nodesByID
}
