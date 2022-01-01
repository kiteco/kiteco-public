package main

import (
	"fmt"
	"reflect"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

// Value is a serializable representation of a resolved pythontype.Value
type Value struct {
	String  string
	Kind    string
	Type    string
	Address string
}

func newValue(expr pythonast.Expr, resolved *pythonanalyzer.ResolvedAST) *Value {
	if resolved == nil {
		return nil
	}

	if ref := resolved.References[expr]; ref != nil {
		val := &Value{
			String: fmt.Sprintf("%v", ref),
			Kind:   ref.Kind().String(),
		}
		val.Address = ref.Address().String()
		if typ := ref.Type(); typ != nil {
			val.Type = fmt.Sprintf("%v", typ)
		}
		return val
	}

	return nil
}

// Loc is a line/column tuple
type Loc struct {
	Line   int
	Column int
}

func newLoc(lineMap *linenumber.Map, offset int) Loc {
	line, col := lineMap.LineCol(offset)
	return Loc{Line: line, Column: col}
}

// Node is a serializable representation of an AST node
type Node struct {
	Begin  Loc
	End    Loc
	Type   string
	Value0 *Value
	Value1 *Value
}

// SourceMap pairs a list of flat AST expressions with the corresponding source lines
type SourceMap struct {
	Nodes       []Node
	SourceLines []string
}

// NewSourceMap constructs a SourceMap from up to two ResolvedASTs; resolved0 should be non-nil
func NewSourceMap(source *pythonbatch.SourceUnit, resolved0, resolved1 *pythonanalyzer.ResolvedAST) SourceMap {
	var exprs []Node
	pythonast.Inspect(resolved0.Root, func(n pythonast.Node) bool {
		if node, ok := n.(pythonast.Expr); ok {
			begin, end := int(node.Begin()), int(node.End())

			exprs = append(exprs, Node{
				Begin:  newLoc(source.Lines, begin),
				End:    newLoc(source.Lines, end),
				Type:   reflect.TypeOf(node).Elem().Name(),
				Value0: newValue(node, resolved0),
				Value1: newValue(node, resolved1),
			})
		}
		return true
	})

	var lines []string
	for i, off := range source.Lines.LineOffsets {
		next := len(source.Contents)
		if i < len(source.Lines.LineOffsets)-1 {
			next = source.Lines.LineOffsets[i+1]
		}
		lines = append(lines, string(source.Contents[off:next]))
	}

	return SourceMap{
		Nodes:       exprs,
		SourceLines: lines,
	}
}
