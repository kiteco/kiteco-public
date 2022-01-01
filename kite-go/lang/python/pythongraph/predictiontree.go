package pythongraph

import (
	"fmt"
	"io"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

const predictionTreeRootMarker = ExprSubTaskType("root")

// PredictionTreeNode is a node in a prediction tree
type PredictionTreeNode struct {
	Task     ExprSubTaskType
	Prob     float32
	Children []*PredictionTreeNode

	AttrBase string
	Attr     pythonresource.Symbol
	Call     PredictedCallSummary
}

func (ptn *PredictionTreeNode) add(child *PredictionTreeNode) {
	ptn.Children = append(ptn.Children, child)
}

// InspectFn inspects a prediction tree node, see Inspect below
type InspectFn func(*PredictionTreeNode) bool

// Inspect traverses a prediction tree in depth-first order: It starts by calling
// f(node); node must not be nil. If f returns true, Inspect invokes f
// recursively for each of the non-nil children of node, followed by a
// call of f(nil).
func Inspect(ptn *PredictionTreeNode, f InspectFn) {
	if f(ptn) {
		for _, child := range ptn.Children {
			Inspect(child, f)
		}
	}
	f(nil)
}

// Print the node to the writer, ptn must be non nil
func Print(ptn *PredictionTreeNode, w io.Writer) {
	var depth int
	Inspect(ptn, func(n *PredictionTreeNode) bool {
		if n == nil {
			depth--
			return false
		}

		space := strings.Repeat("  ", depth)
		var s string
		switch n.Task {
		case InferAttrBaseTask:
			s = fmt.Sprintf("%s (%f)", n.AttrBase, n.Prob)
		case InferAttrTask:
			s = fmt.Sprintf("%s (%f)", n.Attr.Path().Last(), n.Prob)
		case InferCallTask:
			var ps []string
			for _, p := range n.Call.Predicted {
				ps = append(ps, space+"  "+p.String())
			}
			s = fmt.Sprintf("%s:\n%s\n", n.Call.Symbol.PathString(), strings.Join(ps, "\n"))
		}

		if s != "" {
			fmt.Fprintln(w, space+s)
		}

		depth++
		return true
	})
}

func noArgCallNode(sym pythonresource.Symbol, scopeSize int) *PredictionTreeNode {
	return &PredictionTreeNode{
		Task: InferCallTask,
		Prob: 1.,
		Call: PredictedCallSummary{
			Symbol: sym,
			Predicted: []PredictedCall{
				{
					Prob: 1,
				},
			},
			ScopeSize: scopeSize,
		},
	}
}

// ScoredAttribute represents an attribute completion and a score
type ScoredAttribute struct {
	Symbol pythonresource.Symbol
	Score  float32
}

// ScoredArgType represents an argument type prediction and a score
type ScoredArgType struct {
	Type  traindata.ArgType
	Score float32
}

// ScoredKwargName represents a keyword argument name suggestion and a score
type ScoredKwargName struct {
	Name  string
	Score float32
}
