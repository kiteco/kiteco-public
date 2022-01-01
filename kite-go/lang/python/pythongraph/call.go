package pythongraph

import (
	"fmt"
	"io"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// PredictedCallSummary contains the predicted calls for a given symbol
type PredictedCallSummary struct {
	Symbol    pythonresource.Symbol
	Predicted []PredictedCall
	// num variables in scope when prediction was made
	// TODO: clean this up
	ScopeSize int
}

// PredictedCallArg is a predicted argument for a call,
// along with the probability of the argument.
type PredictedCallArg struct {
	Name   string
	Value  string
	Stop   bool
	Prob   float32
	Symbol pythonresource.Symbol
}

// Placeholder return true if the PredictedCallArg contains a placeholder
func (p PredictedCallArg) Placeholder() bool {
	return p.Value == PlaceholderPlaceholder
}

// PredictedCallMetaData ...
type PredictedCallMetaData struct {
	FilteringFeatures []NameAndWeight
	ModelWeight       []NameAndWeight
}

// PredictedCall signature
type PredictedCall struct {
	NumOrigArgs int
	Args        []PredictedCallArg
	Prob        float32
	Symbol      pythonresource.Symbol
	PartialCall bool
	ScopeSize   int
	CallProb    float64
	SkipCall    bool
	MetaData    PredictedCallMetaData
}

// NewPredictedCallArg ...
func NewPredictedCallArg() PredictedCallArg {
	return PredictedCallArg{
		Prob: 1,
	}
}

func (p PredictedCall) id() string {
	var args []string
	for _, a := range p.Args {
		if !a.Stop {
			if a.Name == "" {
				args = append(args, fmt.Sprintf("%s", a.Value))
			} else {
				args = append(args, fmt.Sprintf("%s=%s", a.Name, a.Value))
			}
		} else {
			args = append(args, "STOP")
		}
	}

	return strings.Join(args, ", ")
}

// String implements stringer
func (p PredictedCall) String() string {
	var args []string
	for _, a := range p.Args {
		if !a.Stop {
			if a.Name == "" {
				args = append(args, fmt.Sprintf("%s", a.Value))
			} else {
				args = append(args, fmt.Sprintf("%s=%s", a.Name, a.Value))
			}
		} else {
			args = append(args, "STOP")
		}
	}

	return fmt.Sprintf("NumOrigArgs: %d, Predicted: %s (%f)", p.NumOrigArgs, strings.Join(args, ", "), p.Prob)
}

// beamSearchNode stores information for a step in a beam search.
type beamSearchNode struct {
	Chosen   PredictedCallArg
	Children []*beamSearchNode
}

// Print the node and its children.
func (bsn *beamSearchNode) Print(depth int, w io.Writer) {
	rep := strings.Repeat("\t", depth)
	s := fmt.Sprintf("%s %s %s %f", rep, bsn.Chosen.Name, bsn.Chosen.Value, bsn.Chosen.Prob)
	fmt.Fprintln(w, s)

	depth++
	for _, child := range bsn.Children {
		child.Print(depth, w)
	}
}

// walkArgs returns a slice where each entry in the slice is a
// full predicted call, e.g each entry in the slice
// is a path from the root of the beam search tree to a leaf
func (bsn *beamSearchNode) walkArgs(ctx kitectx.Context) [][]PredictedCallArg {
	ctx.CheckAbort()

	if len(bsn.Children) == 0 {
		return [][]PredictedCallArg{
			{bsn.Chosen},
		}
	}

	var prefix []PredictedCallArg
	if bsn.Chosen.Stop != false || bsn.Chosen.Value != "" || bsn.Chosen.Name != "" || bsn.Chosen.Prob != 0 {
		prefix = append(prefix, bsn.Chosen)
	}

	var res [][]PredictedCallArg
	for _, child := range bsn.Children {
		for _, suffix := range child.walkArgs(ctx) {
			newest := append(prefix, suffix...)
			res = append(res, newest)
		}
	}
	return res
}

// NameAndWeight ...
type NameAndWeight struct {
	Name   string
	Weight float32
}

// String ...
func (n NameAndWeight) String() string {
	return fmt.Sprintf("%v: %v", n.Name, n.Weight)
}
