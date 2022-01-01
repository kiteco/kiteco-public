package utils

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	// scanOpts and parseOpts should match the options in the driver (or whatever is running inference with the model)
	scanOpts = pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}

	parseOpts = pythonparser.Options{
		ScanOptions: pythonscanner.Options{
			ScanComments: false,
			ScanNewLines: false,
		},
		ErrorMode: pythonparser.Recover,
	}
)

const (
	// ArgCategories corresponds to the number of classes considered for the filtering model
	// Each class correspond to a number of args in the completion
	// Each class will be balanced in term of number of positive and negative sample in the training dataset
	ArgCategories = 3
)

// Parse ...
func Parse(src []byte) (*pythonast.Module, []pythonscanner.Word, error) {
	var ast *pythonast.Module
	var words []pythonscanner.Word
	err := kitectx.Background().WithTimeout(2*time.Second, func(ctx kitectx.Context) error {
		var err error
		words, err = pythonscanner.Lex(src, scanOpts)
		if err != nil {
			return errors.Errorf("unable to re-lex file: %v", err)
		}

		ast, err = pythonparser.ParseWords(ctx, src, words, parseOpts)
		if err != nil {
			return errors.Errorf("unable to re-parse file")
		}
		return err
	})

	if ast == nil {
		return nil, nil, err
	}
	return ast, words, nil
}

// TryCall processes the source by chopping off the arguments and try to find the same CallExpr
func TryCall(call *pythonast.CallExpr, src []byte, res Resources) (Input, error) {
	// TODO: we should be removing the rest of the line as well
	src = bytes.Join([][]byte{
		src[:call.LeftParen.End],
		src[call.RightParen.Begin:],
	}, nil)

	ast, words, err := Parse(src)
	if err != nil {
		return Input{}, err
	}

	// find the CallExpr at the same position as the original one
	var call2 *pythonast.CallExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !pythonast.IsNil(call2) {
			return false
		}

		if n, ok := n.(*pythonast.CallExpr); ok {
			if n.LeftParen.Begin == call.LeftParen.Begin {
				call2 = n
				return false
			}
		}
		return true
	})
	if pythonast.IsNil(call2) {
		return Input{}, errors.Errorf("unable to find CallExpr again")
	}

	rast, err := Resolve(ast, res.RM)
	if err != nil {
		return Input{}, errors.Errorf("unable to resolve file: %v", err)
	}

	return Input{
		RM:     res.RM,
		RAST:   rast,
		Words:  words,
		Call:   call2,
		Src:    src,
		Cursor: int64(call2.LeftParen.Begin),
	}, nil
}

// Input is a general input that can be used for both the expr model and the callProb model.
type Input struct {
	RM     pythonresource.Manager
	Src    []byte
	Words  []pythonscanner.Word
	RAST   *pythonanalyzer.ResolvedAST
	Call   *pythonast.CallExpr
	Cursor int64
}

// SampleTag ...
func (Input) SampleTag() {}

// Sample represents the raw sample which has source and left parenthesis positions.
type Sample struct {
	Source  []byte
	PosList []int
}

// SampleTag ...
func (Sample) SampleTag() {}

// Resources contain resources for running the binary
type Resources struct {
	RM     pythonresource.Manager
	Models *pythonmodels.Models
}

// PredictCall ...
func (r Resources) PredictCall(src []byte, words []pythonscanner.Word, rast *pythonanalyzer.ResolvedAST, site pythonast.Expr) ([]pythongraph.PredictedCall, pythonresource.Symbol, int, error) {
	var ptn *pythongraph.PredictionTreeNode
	err := kitectx.Background().WithTimeout(3*time.Second, func(ctx kitectx.Context) error {
		result, err := r.Models.Expr.Predict(ctx, pythonexpr.Input{
			RM:    r.RM,
			RAST:  rast,
			Words: words,
			Src:   src,
			Expr:  site,
		})
		if err == nil {
			ptn = result.OldPredictorResult
		}
		return err
	})

	if err != nil {
		return nil, pythonresource.Symbol{}, -1, errors.Errorf("infer error: %v", err)
	}

	for _, child := range ptn.Children {
		if len(child.Call.Predicted) > 0 {
			return child.Call.Predicted, child.Call.Symbol, child.Call.ScopeSize, nil
		}
	}
	return nil, pythonresource.Symbol{}, -1, errors.Errorf("no call found after inference")
}

// NumCallArgs counts the number of argument in the call
func NumCallArgs(args []pythongraph.PredictedCallArg) int {
	var numArgs int
	for _, arg := range args {
		if arg.Stop {
			break
		}
		numArgs++
	}
	return numArgs
}

// Resolve ...
func Resolve(ast *pythonast.Module, rm pythonresource.Manager) (*pythonanalyzer.ResolvedAST, error) {
	var rast *pythonanalyzer.ResolvedAST
	err := kitectx.Background().WithTimeout(1*time.Second, func(ctx kitectx.Context) error {
		var err error
		rast, err = pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{Path: "/src.py"}).ResolveContext(ctx, ast, false)
		return err
	})
	if err != nil {
		return nil, err
	}
	return rast, nil
}

// IsPredicted compares between the call expression in user's source code and available call expression predicted by the expr model
// It returns a position and a nil error if there is a match.
func IsPredicted(preds []pythongraph.PredictedCall, call *pythonast.CallExpr, partialCalls bool) ([]int, error) {
	var result []int

outer:
	for pos, p := range preds {
		if !partialCalls && NumCallArgs(p.Args) != len(call.Args) {
			continue
		}
		for i, predArg := range p.Args {
			if predArg.Stop {
				break
			}
			if i >= len(call.Args) {
				continue outer
			}

			var actualName string
			if !pythonast.IsNil(call.Args[i].Name) {
				actualName = call.Args[i].Name.(*pythonast.NameExpr).Ident.Literal
			}
			actualValue := pythongraph.PlaceholderPlaceholder
			if val, ok := call.Args[i].Value.(*pythonast.NameExpr); ok {
				actualValue = val.Ident.Literal
			}
			if predArg.Name != actualName || predArg.Value != actualValue {
				continue outer
			}
		}
		result = append(result, pos)
		if !partialCalls {
			// There can be only one match in full call mode
			break outer
		}
	}
	if result == nil {
		return nil, errors.Errorf("no match found")
	}
	return result, nil

}

// FindCalls takes sample which are the source and positions of parenthesis and find all the corresponding calls
func FindCalls(s Sample) ([]*pythonast.CallExpr, error) {
	words, err := pythonscanner.Lex(s.Source, scanOpts)
	if err != nil {
		return nil, errors.Errorf("unable to lex file: %v", err)
	}

	ast, err := pythonparser.ParseWords(kitectx.Background(), s.Source, words, parseOpts)
	if err != nil {
		return nil, errors.Errorf("unable to parse file: %v", err)
	}

	var calls []*pythonast.CallExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		if n, ok := n.(*pythonast.CallExpr); ok {
			for _, p := range s.PosList {
				if int(n.LeftParen.Begin) == p {
					calls = append(calls, n)
					break
				}
			}
		}
		return true
	})

	if len(calls) == 0 {
		return nil, errors.Errorf("no call expressions found")
	}

	return calls, nil
}

// ValueSupported ...
func ValueSupported(res Resources, val pythontype.Value) bool {
	for _, sym := range python.GetExternalSymbols(kitectx.Background(), res.RM, val) {
		if res.Models.Expr.CallSupported(res.RM, sym) == nil {
			return true
		}
	}
	return false
}

// KeepOnlyCompleteCalls filter predicted calls to only keep the one representing a complete call (with closing parenthesis)
func KeepOnlyCompleteCalls(calls []pythongraph.PredictedCall) []pythongraph.PredictedCall {
	var result []pythongraph.PredictedCall
	for _, c := range calls {
		if len(c.Args) == 0 {
			fmt.Println("Call with no args at all")
		}
		if c.Args[len(c.Args)-1].Stop {
			result = append(result, c)
		}
	}
	return result
}

// TruncateCalls truncates prediction to produce all possible partial call from this set of predictions
func TruncateCalls(calls []pythongraph.PredictedCall) []pythongraph.PredictedCall {
	var result []pythongraph.PredictedCall
	for _, c := range calls {
		result = append(result, truncateCall(c)...)
	}
	return dedupeCalls(result)
}

func truncateCall(call pythongraph.PredictedCall) []pythongraph.PredictedCall {
	var result []pythongraph.PredictedCall
	for i := range call.Args {
		if call.Args[i].Stop {
			break
		}
		tCall := call
		tCall.Args = tCall.Args[:i+1]
		updateScore(&tCall)
		tCall.PartialCall = true
		result = append(result, tCall)
		if len(tCall.Args) > 0 && tCall.Args[len(tCall.Args)-1].Name != "" {
			tCall2 := tCall
			tCall2.Args = append([]pythongraph.PredictedCallArg{}, tCall.Args...)
			tCall2.Args[len(tCall2.Args)-1].Value = ""
			result = append(result, tCall2)
		}
	}

	return result
}

func updateScore(call *pythongraph.PredictedCall) {
	score := float32(1.0)
	for _, c := range call.Args {
		score *= c.Prob
	}
	call.Prob = score
}

func dedupeCalls(calls []pythongraph.PredictedCall) []pythongraph.PredictedCall {
	dedupMap := make(map[string]struct{})
	var result []pythongraph.PredictedCall
	for _, c := range calls {
		repr := getStringForCall(c)
		if _, ok := dedupMap[repr]; !ok {
			dedupMap[repr] = struct{}{}
			result = append(result, c)
		}
	}
	return result
}

func getStringForCall(c pythongraph.PredictedCall) string {
	var args []string
	var hasStop bool
	for _, a := range c.Args {
		if a.Stop {
			hasStop = true
		} else {
			var arg string
			if a.Name != "" {
				arg = fmt.Sprintf("%s=%s", a.Name, a.Value)
			} else {
				arg = a.Value
			}
			args = append(args, arg)
		}
	}
	result := fmt.Sprintf("(%s", strings.Join(args, ", "))
	if hasStop {
		result += ")"
	}
	return result
}

// GetLabelsOnStruct returns the index of all the completion that are a prefix of what the user typed
func GetLabelsOnStruct(userCall *pythonast.CallExpr, comps []pythongraph.PredictedCall, keepOnlyMaxLength bool, partialCall bool) ([]int, error) {
	type labelsAndLength struct {
		label  int
		length int
	}
	var labels []labelsAndLength

	for i, comp := range comps {
		prefixLength := isPrefixOnStruct(comp, userCall, partialCall)
		if prefixLength != -1 {
			labels = append(labels, labelsAndLength{
				label:  i,
				length: prefixLength,
			})
		}
	}
	if len(labels) == 0 {
		return nil, errors.Errorf("No valid label for input")
	}
	sort.Slice(labels, func(i, j int) bool {
		if labels[i].length == labels[j].length {
			return labels[i].label < labels[j].label
		}
		return labels[i].length > labels[j].length
	})
	var result []int
	maxLength := labels[0].length
	for _, l := range labels {
		if !keepOnlyMaxLength || l.length == maxLength {
			result = append(result, l.label)
		}
	}
	return result, nil
}

// isPrefix checks if prefix is a prefix of tokens
func isPrefixOnStruct(comp pythongraph.PredictedCall, userCall *pythonast.CallExpr, partialCall bool) int {
	compArgs := comp.Args
	if len(compArgs) > 0 && compArgs[len(compArgs)-1].Stop {
		compArgs = compArgs[:len(compArgs)-1]
	}

	if len(compArgs) > len(userCall.Args) {
		return -1
	}
	for i := 0; i < len(compArgs); i++ {
		// If it's not a name, it's a placeholder
		nameExpr, isNameExpr := userCall.Args[i].Value.(*pythonast.NameExpr)
		argGroundTruth := pythongraph.PlaceholderPlaceholder
		if isNameExpr {
			argGroundTruth = nameExpr.Ident.Literal
		}
		predArg := compArgs[i]
		predValue := predArg.Value
		if predValue == "" {
			predValue = pythongraph.PlaceholderPlaceholder
		}

		if predValue != argGroundTruth {
			return -1 // The completion match up to the previous arg
		}
		userKeywordExpr, hasKeyword := userCall.Args[i].Name.(*pythonast.NameExpr)
		if hasKeyword != (predArg.Name != "") {
			return -1 // The completion match up to the previous arg
		} else if hasKeyword && predArg.Name != userKeywordExpr.Ident.Literal {
			return -1 // The completion match up to the previous arg
		}
	}
	return len(compArgs)
}
