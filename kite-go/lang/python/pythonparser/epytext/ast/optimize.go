package ast

import "bytes"

// Optimize walks the AST rooted at n and runs optimization passes
// on it. The optimizations applied are:
// - removal of empty nodes
// - merging of subsequent Text nodes
func Optimize(n Node) {
	Walk(&optimizer{}, n)
}

type optimizer struct {
	node NestingNode
}

func (o *optimizer) Visit(n Node) Visitor {
	// visit each NestingNode with its own optimizer
	if nn, ok := n.(NestingNode); ok {
		return &optimizer{node: nn}
	}

	if n != nil {
		// do not visit leaves
		return nil
	}

	// at this point, n is nil so this is the exit from that optimizing
	// visitor - apply the optimizations on o.node.
	o.removeEmptyNodes()
	o.collapseTextNodes()
	return nil
}

func (o *optimizer) collapseTextNodes() {
	var collapsedNodes []Node
	originalChildren := o.node.children()

	// helper function to collapse and add the Text nodes from [start, end).
	appendCollapsed := func(start, end int) {
		if start < 0 || end <= start {
			return
		}
		if end-start == 1 {
			// single node, no collapsing needed
			collapsedNodes = append(collapsedNodes, originalChildren[start])
		} else {
			var buf bytes.Buffer
			for i := start; i < end; i++ {
				buf.WriteString(string(originalChildren[i].(Text)))
			}
			collapsedNodes = append(collapsedNodes, Text(buf.String()))
		}
	}

	firstText := -1
	for i, child := range originalChildren {
		if _, ok := child.(Text); ok {
			// this is a text node
			if firstText < 0 {
				// start of (possibly) as list of subsequent Text nodes
				firstText = i
			}
			continue
		}

		// at this point, child is not a Text node, collapse the existing list of text
		// nodes if any.
		if firstText >= 0 {
			appendCollapsed(firstText, i)
			firstText = -1
		}

		// and add the current node to the list of nodes (we don't remove any non-text node here)
		collapsedNodes = append(collapsedNodes, child)
	}

	// if the original children ended with a Text node, make sure it is added
	// and collapsed if needed.
	appendCollapsed(firstText, len(originalChildren))

	o.node.setChildren(collapsedNodes)
}

func (o *optimizer) removeEmptyNodes() {
	var nonEmptyNodes []Node
	originalChildren := o.node.children()

	for _, child := range originalChildren {
		switch child := child.(type) {
		case NestingNode:
			if len(child.children()) > 0 {
				nonEmptyNodes = append(nonEmptyNodes, child)
			}
		case LeafNode:
			if child.Text() != "" {
				nonEmptyNodes = append(nonEmptyNodes, child)
			}
		}
	}
	o.node.setChildren(nonEmptyNodes)
}
