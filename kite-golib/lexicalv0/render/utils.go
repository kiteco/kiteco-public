package render

import (
	sitter "github.com/kiteco/go-tree-sitter"
)

const (
	// JsTempPlaceholder is temporarily used to make `FormatCompletion` work properly for JS
	// Later to be replaced by the real Placeholder
	JsTempPlaceholder = "K§"
	// GoTempPlaceholder is temporarily used to make `FormatCompletion` work properly for Golang
	// Later to be replaced by the real Placeholder, it's safe because the model does noe predict string literals
	GoTempPlaceholder = "\"K\""
	// BlankPlaceholder is the true Placeholder string
	BlankPlaceholder = "..."
)

// CursorInsideNode returns true if cursor is in node
// specifically for nodes like `(a, b, c)` or `[a, b, c]`
func CursorInsideNode(n *sitter.Node, cursor int) bool {
	first, last := SafeChild(n, 0), SafeChild(n, int(n.ChildCount())-1)
	if first == nil || last == nil {
		return false
	}
	return cursor >= int(first.EndByte()) && cursor <= int(last.StartByte())
}

// SafeSymbol ...
func SafeSymbol(n *sitter.Node) int {
	if n != nil {
		return int(n.Symbol())
	}
	return -1
}

// SafeEqual ...
func SafeEqual(n1, n2 *sitter.Node) bool {
	switch {
	case n1 == nil && n2 == nil:
		return true
	case n1 != nil && n2 != nil:
		return n1.Equal(n2)
	default:
		return false
	}
}

// SafeChild ...
func SafeChild(n *sitter.Node, c int) *sitter.Node {
	if n == nil {
		return nil
	}
	if c >= 0 && int(n.ChildCount()) > c {
		return n.Child(c)
	}
	return nil
}

// SafeParent ...
func SafeParent(n *sitter.Node) *sitter.Node {
	if n == nil {
		return nil
	}
	return n.Parent()
}
