package pythonhelpers

import (
	"go/token"
	"unicode"
	"unicode/utf8"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// UnderCursor checks if the given ast node is under the cursor;
// if the cursor is immediately before or immediately after the node, we return true
func UnderCursor(node pythonast.Node, cursor int64) bool {
	if pythonast.IsNil(node) {
		return false
	}
	return cursor >= int64(node.Begin()) && cursor <= int64(node.End())
}

// ContainsSelection checks if the given node contains the selection, inclusive of the node boundaries.
func ContainsSelection(node pythonast.Node, start, end int64) bool {
	return UnderCursor(node, start) && (start == end || UnderCursor(node, end))
}

// NodesUnderCursor returns a slice of nodes that are under the current cursor, in order of increasing depth.
func NodesUnderCursor(ctx kitectx.Context, ast pythonast.Node, cur int64) []pythonast.Node {
	ctx.CheckAbort()
	var nodes []pythonast.Node
	InspectContainingSelection(ctx, ast, cur, cur, func(node pythonast.Node) bool {
		nodes = append(nodes, node)
		return true
	})
	return nodes
}

// DeepestContainingSelection computes the deepest/last node containing (inclusive of node boundaries) the given selection.
func DeepestContainingSelection(ctx kitectx.Context, ast pythonast.Node, start, end int64) pythonast.Node {
	ctx.CheckAbort()
	var res pythonast.Node
	InspectContainingSelection(ctx, ast, start, end, func(node pythonast.Node) bool {
		res = node
		return true
	})
	return res
}

// InspectContainingSelection behaves like pythonast.Inspect:
// cb will be called on Nodes containing the given selection in depth-first order, and never with a nil node.
func InspectContainingSelection(ctx kitectx.Context, ast pythonast.Node, start, end int64, cb func(pythonast.Node) bool) {
	ctx.CheckAbort()
	pythonast.Inspect(ast, func(node pythonast.Node) bool {
		ctx.CheckAbort()
		if pythonast.IsNil(node) || !ContainsSelection(node, start, end) {
			return false
		}
		return cb(node)
	})
}

// -

// IsHSpace determines whether r is horizontal whitespace (tab or space, not newline)
func IsHSpace(r rune) bool {
	return r != '\n' && unicode.IsSpace(r)
}

// NearestNonWhitespace finds the nearest non-whitespace character next to the cursor. isSpace is used to determine
// whether the character under the cursor is a whitespace character.
func NearestNonWhitespace(buf []byte, cursor int64, isSpace func(r rune) bool) int64 {
	// if there is a non-whitespace to right of cursor then do not move
	// (using IsSpace not IsHSpace is intentional here)
	if cursor >= 0 && cursor < int64(len(buf)) {
		if r, _ := utf8.DecodeRune(buf[cursor:]); r == utf8.RuneError || !unicode.IsSpace(r) {
			return cursor
		}
	}

	// otherwise keep moving the cursor to the left until we hit a non-hspace char
	for cursor > 0 && cursor <= int64(len(buf)) {
		r, n := utf8.DecodeLastRune(buf[:cursor])
		if r == utf8.RuneError || !isSpace(r) {
			break
		}
		cursor -= int64(n)
	}
	return cursor
}

// -

func incompleteCall(c *pythonast.CallExpr) bool {
	return c.RightParen == nil || c.RightParen.Token == pythonscanner.BadToken
}

// CursorBetweenCallParens checks if the cursor is strictly between the parens for the given CallExpr
func CursorBetweenCallParens(c *pythonast.CallExpr, cursor token.Pos) bool {
	return cursor > c.LeftParen.Begin && (incompleteCall(c) || cursor < c.RightParen.End)
}
