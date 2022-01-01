package pythonimports

import "fmt"

const (
	// None represents the zero value for Kind
	None Kind = iota
	// Function is the classification for nodes representing functions
	Function
	// Type is the classification for nodes representing types
	Type
	// Module is the classification for nodes representing modules
	Module
	// Descriptor is the classification for nodes representing descriptors
	Descriptor
	// Object is the classification for nodes that do not fall into any other category
	Object
	// Root is the classification for the virtual root node (graph.Root)
	Root
)

// Kind represents the classification for a node
type Kind int

// String converts a Kind to a string
func (c Kind) String() string {
	switch c {
	case Function:
		return "function"
	case Type:
		return "type"
	case Module:
		return "module"
	case Descriptor:
		return "descriptor"
	case Object:
		return "object"
	default:
		return fmt.Sprintf("Kind(%d)", c)
	}
}

// ParseKind converts a string to a Kind
func ParseKind(s string) Kind {
	switch s {
	case "function":
		return Function
	case "type":
		return Type
	case "module":
		return Module
	case "descriptor":
		return Descriptor
	case "object":
		return Object
	default:
		return None
	}
}

// NodeInfo represents information associated with an entry in the Python import graph.
type NodeInfo struct {
	ID             int64      `json:"id"`
	CanonicalName  DottedPath `json:"canonical_name"`
	Classification Kind       `json:"classification"`
	Origin         Origin
}

// A Node represents information associated with an entry in the Python import graph
type Node struct {
	NodeInfo
	Type    *Node
	Members map[string]*Node
	// Bases is nil if Classification != Type, and it corresponds to __bases__ in python.
	Bases []*Node
}

// NewNode creates a node with a name and a kind
func NewNode(name string, kind Kind) *Node {
	return &Node{
		Members: make(map[string]*Node),
		NodeInfo: NodeInfo{
			CanonicalName:  NewDottedPath(name),
			Classification: kind,
		},
	}
}

// HasMember checks whether a node has the given node as its member.
func (n *Node) HasMember(m *Node) bool {
	if m == nil {
		return false
	}
	for _, node := range n.Members {
		if m == node {
			return true
		}
	}
	return false
}

// HasUnresolvedBase checks whether a node has any unresolved parent.
// Only a node of `Type` kind can possible have this function return true.
func (n *Node) HasUnresolvedBase() bool {
	for _, base := range n.Bases {
		if base == nil {
			return true
		}
	}
	if n.Type != nil && n.Type != n {
		return n.Type.HasUnresolvedBase()
	}
	return false
}

const maxAttrDepth = 10

func (n *Node) attr(attr string, depth int) (*Node, bool) {
	if depth == maxAttrDepth {
		return nil, false
	}
	depth++

	if node, exists := n.Members[attr]; exists {
		return node, true
	}
	if n.Type != nil && n.Type != n {
		if node, found := n.Type.attr(attr, depth); found {
			return node, true
		}
	}
	for _, base := range n.Bases {
		if base == nil {
			continue
		}
		if base != n {
			if node, found := base.attr(attr, depth); found {
				return node, true
			}
		}
	}
	return nil, false
}

// Attr evaluates an attribute on this node by first looking in the members map for this
// node, and then looking within the type of this node, just like python does.
func (n *Node) Attr(attr string) (*Node, bool) {
	return n.attr(attr, 0)
}

// AttrOf finds attribute of the class that has the given node class type and the given name.
func (n *Node) AttrOf(attr string, kind Kind) (*Node, bool) {
	node, found := n.Attr(attr)
	if found || node.Classification == kind {
		return node, true
	}
	return nil, false
}

// String returns a short string representation of the node
func (n *Node) String() string {
	if n == nil {
		return "{Node=nil}"
	}
	if n.CanonicalName.Empty() {
		if n.Type == nil || n.Type.CanonicalName.Empty() {
			return fmt.Sprintf("{Node %d}", n.ID)
		}
		return fmt.Sprintf("{instance of %s}", n.Type.CanonicalName.String())
	}
	return n.CanonicalName.String()
}

func (n *Node) attrs(steps int) []string {
	if steps >= maxAttrDepth {
		return nil
	}
	steps++

	var attributes []string
	for attr := range n.Members {
		attributes = append(attributes, attr)
	}
	if n.Type != nil && n.Type != n {
		attributes = append(attributes, n.Type.attrs(steps)...)
	}
	for _, base := range n.Bases {
		if base != nil && base != n {
			attributes = append(attributes, base.attrs(steps)...)
		}
	}
	return attributes
}

// Attrs returns all the attributes of a node.
func (n *Node) Attrs() []string {
	return n.attrs(0)
}

func (n *Node) attrsByKind(steps int) map[Kind][]string {
	if steps >= maxAttrDepth {
		return nil
	}
	steps++

	byKind := make(map[Kind][]string)
	for attr, node := range n.Members {
		if node != nil {
			byKind[node.Classification] = append(byKind[node.Classification], attr)
		} else {
			byKind[None] = append(byKind[None], attr)
		}
	}
	if n.Type != nil && n.Type != n {
		for kind, attrs := range n.Type.attrsByKind(steps) {
			byKind[kind] = append(byKind[kind], attrs...)
		}
	}
	for _, base := range n.Bases {
		if base != nil && base != n {
			for kind, attrs := range base.attrsByKind(steps) {
				byKind[kind] = append(byKind[kind], attrs...)
			}
		}
	}
	return byKind
}

// AttrsByKind returns the node's attributes by kind.
func (n *Node) AttrsByKind() map[Kind][]string {
	return n.attrsByKind(0)
}

// ShallowCopy makes a shallow copy of this node. This includes a copy of the
// members and bases, but none of the referenced nodes are copied.
func (n *Node) ShallowCopy() *Node {
	clone := *n
	clone.Bases = make([]*Node, len(n.Bases))
	for i, base := range n.Bases {
		clone.Bases[i] = base
	}
	clone.Members = make(map[string]*Node)
	for attr, child := range n.Members {
		clone.Members[attr] = child
	}
	return &clone
}
