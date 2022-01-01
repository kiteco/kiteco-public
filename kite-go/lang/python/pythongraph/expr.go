package pythongraph

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// ExprSubTaskType denotes the various infer expr sub tasks, this is used
// when sending errors back to the train sample builder to indicate
// which sub task generated a particular bad hash
type ExprSubTaskType string

const (
	// InferAttrTask ...
	InferAttrTask = ExprSubTaskType("infer_attr_task")
	// InferAttrBaseTask ...
	InferAttrBaseTask = ExprSubTaskType("infer_attr_base_task")
	// InferCallTask ...
	InferCallTask = ExprSubTaskType("infer_call_task")
	// InferPlaceholderValueTask ...
	InferPlaceholderValueTask = ExprSubTaskType("infer_placeholder_value_task")

	// InferArgTypeTask (only used in training)
	InferArgTypeTask = ExprSubTaskType("infer_arg_type_task")
	// InferKwargNameTask (only used in training)
	InferKwargNameTask = ExprSubTaskType("infer_kwarg_name_task")
	// InferKwargValueTask (only used in training)
	InferKwargValueTask = ExprSubTaskType("infer_kwarg_value_task")
	// InferArgPlaceholderTask (only used in training)
	InferArgPlaceholderTask = ExprSubTaskType("infer_arg_placeholder_task")
)

// AttrCallbacks bundles the callbacks needed for infer attr tasks
type AttrCallbacks struct {
	Supported    func(pythonresource.Manager, pythonresource.Symbol) error
	Candidates   func(pythonresource.Manager, pythonresource.Symbol) ([]int32, []pythonresource.Symbol, error)
	ByPopularity func(pythonresource.Manager, pythonresource.Symbol) ([]ScoredAttribute, error)
}

// NameAndIdx ...
type NameAndIdx struct {
	Name string
	Idx  int32
}

// FuncInfo for predicting arguments for a call to a function
type FuncInfo struct {
	Symbol             pythonresource.Symbol
	Patterns           *traindata.CallPatterns
	ArgTypeIdxs        map[traindata.ArgType]int32
	KwargNameIdxs      []NameAndIdx
	ArgPlaceholderIdxs map[string]map[traindata.ArgPlaceholder]int32
}

// CallCallbacks bundles the callbacks needed for infer call tasks
type CallCallbacks struct {
	Supported func(pythonresource.Manager, pythonresource.Symbol) error
	Info      func(pythonresource.Manager, pythonresource.Symbol) (*FuncInfo, error)
}

// ExprCallbacks contains the callbacks needed when performing
// expr prediction
type ExprCallbacks struct {
	Attr AttrCallbacks
	Call CallCallbacks
}

// PredictExprInput are the inputs for expr prediction
type PredictExprInput struct {
	In        Inputs
	Model     *tensorflow.Model
	Expr      pythonast.Expr
	Arg       *pythonast.Argument
	Callbacks ExprCallbacks
	Meta      ModelMeta
	Tracer    io.Writer
	Saver     Saver
}

// PredictExprConfig ...
type PredictExprConfig struct {
	MaxHops              int
	Graph                GraphFeedConfig
	BeamSize             int
	MaxDepth             int
	UseUncompressedModel bool
}

func filterArgs(args []*pythonast.Argument) []*pythonast.Argument {
	var result []*pythonast.Argument
	for _, a := range args {
		if a.Begin() != a.End() {
			result = append(result, a)
		}
	}
	return result
}

// PredictExpr from the inputs
func PredictExpr(ctx kitectx.Context, config PredictExprConfig, in PredictExprInput) (*PredictionTreeNode, error) {
	ctx.CheckAbort()

	predictor, err := NewPredictor(ctx, ContextGraphConfig{
		Graph:     config.Graph,
		MaxHops:   config.MaxHops,
		Propagate: true,
	}, PredictorInputs{
		ModelMeta:            in.Meta,
		Model:                in.Model,
		In:                   in.In,
		Site:                 in.Expr,
		Tracer:               in.Tracer,
		Callbacks:            in.Callbacks,
		Saver:                in.Saver,
		UseUncompressedModel: config.UseUncompressedModel,
	})

	if err != nil {
		return nil, err
	}

	return predictor.PredictExpr(ctx)
}

//
// -- logging stuff
//

type logWriter struct{}

func (logWriter) Write(p []byte) (int, error) {
	log.Print(string(p))
	return len(p), nil
}

type tracer struct {
	indent int
	w      io.Writer
}

// Usage pattern: defer un(trace("..."))
func un(t *tracer) {
	if t == nil {
		return
	}
	t.indent--
}

func trace(t *tracer, format string, args ...interface{}) *tracer {
	if t == nil {
		return t
	}
	t.indent++

	print(t, format, args...)

	return t
}

func print(t *tracer, format string, args ...interface{}) {
	if t == nil {
		return
	}

	indent := strings.Repeat("  ", t.indent)
	fmt.Fprintln(t.w, indent+fmt.Sprintf(format, args...))
}
