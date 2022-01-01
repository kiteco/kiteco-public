package utils

import (
	"bytes"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
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

// Completion adds an Identifier to the mtacconf.Completion
type Completion struct {
	mtacconf.Completion
	Identifier string
}

// Resources contains the resources necessary for the pipeline.
type Resources struct {
	RM     pythonresource.Manager
	Models *pythonmodels.Models
}

// getNameExprCutoffPos gets inputs for preprocess the buffer to be fed into the expr model
func getNameExprCutoffPos(node pythonast.Node, rast *pythonanalyzer.ResolvedAST) (int, error) {
	switch n := rast.Parent[node].(type) {
	case *pythonast.Argument:
		parent, _ := rast.Parent[n]
		call, ok := parent.(*pythonast.CallExpr)
		if !ok {
			return 0, errors.Errorf("can't convert to CallExpr")
		}
		return int(call.RightParen.Begin), nil
	case *pythonast.WhileStmt:
		return int(n.Condition.End()), nil
	case *pythonast.IfExpr:
		return int(n.Condition.End()), nil
	case *pythonast.Branch:
		return int(n.Condition.End()), nil
	case *pythonast.ForStmt:
		return int(n.Iterable.End()), nil
	}
	return 0, errors.Errorf("not a scenario the model considers")
}

// FindNameExprScenarios finds NameExpr under while, if, for, and call scenarios
func FindNameExprScenarios(rast *pythonanalyzer.ResolvedAST) []*pythonast.NameExpr {
	var nameList []*pythonast.NameExpr
	pythonast.Inspect(rast.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}

		switch n := n.(type) {
		case *pythonast.NameExpr:
			pn := rast.Parent[n]
			switch pn := pn.(type) {
			case *pythonast.WhileStmt:
				nameList = append(nameList, n)
			case *pythonast.Branch:
				nameList = append(nameList, n)
			case *pythonast.Argument:
				if pythonast.IsNil(pn.Name) {
					nameList = append(nameList, n)
				}
			case *pythonast.ForStmt:
				if pn.Iterable == n {
					nameList = append(nameList, n)
				}
			case *pythonast.IfExpr:
				if pn.Condition == n {
					nameList = append(nameList, n)
				}
			default:
				return false
			}
		case *pythonast.AttributeExpr:
			name, ok := n.Value.(*pythonast.NameExpr)
			if !ok {
				break
			}
			pn := rast.Parent[n]
			switch pn := pn.(type) {
			case *pythonast.WhileStmt:
				nameList = append(nameList, name)
			case *pythonast.Branch:
				nameList = append(nameList, name)
			case *pythonast.Argument:
				if pythonast.IsNil(pn.Name) {
					nameList = append(nameList, name)
				}
			case *pythonast.ForStmt:
				if pn.Iterable == n {
					nameList = append(nameList, name)
				}
			case *pythonast.IfExpr:
				if pn.Condition == n {
					nameList = append(nameList, name)
				}
			default:
				return false
			}

		}
		return true
	})
	return nameList
}

// NameInput is a general input that can be used for both the expr model and the MTACConf model.
type NameInput struct {
	pythonproviders.Inputs
	Name      *pythonast.NameExpr
	UserTyped []byte
}

// TryName tries getting general inputs for the given name
func TryName(src []byte, name *pythonast.NameExpr, rast *pythonanalyzer.ResolvedAST, global pythonproviders.Global) (NameInput, error) {
	userTyped := src[name.Begin():]

	var cutoff int
	var err error
	if _, ok := rast.Parent[name].(*pythonast.AttributeExpr); !ok {
		cutoff, err = getNameExprCutoffPos(name, rast)
	} else {
		cutoff, err = getNameExprCutoffPos(rast.Parent[name], rast)
	}
	if err != nil {
		return NameInput{}, errors.Errorf("can't get nameExprInputs: %v", err)
	}

	// since name with a single character will get thrown out because we won't be able to re-find it again
	cursor := int(name.End())
	if name.End()-name.Begin() > 1 {
		cursor--
	}

	src = bytes.Join([][]byte{
		src[:cursor],
		src[cutoff:],
	}, nil)

	inputs, err := pythonproviders.NewInputs(kitectx.Background(), global, data.NewBuffer(string(src)).Select(data.Cursor(cursor)), false, false)
	if err != nil {
		return NameInput{}, errors.Errorf("unable to compute inputs: %v", err)
	}

	var found bool
	pythonast.Inspect(inputs.ResolvedAST().Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		switch n := n.(type) {
		case *pythonast.NameExpr:
			if n.Begin() == name.Begin() {
				name = n
				found = true
				return false
			}
		}
		return true
	})
	if !found {
		return NameInput{}, errors.Errorf("unable to find NameExpr again")
	}

	return NameInput{inputs, name, userTyped}, nil
}

// Resolve resolves the ast
func Resolve(ast *pythonast.Module, rm pythonresource.Manager) (*pythonanalyzer.ResolvedAST, error) {
	var rast *pythonanalyzer.ResolvedAST
	err := kitectx.Background().WithTimeout(5*time.Second, func(ctx kitectx.Context) error {
		var err error
		rast, err = pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{Path: "/src.py"}).ResolveContext(ctx, ast, false)
		return err
	})
	return rast, err
}

// TypeValueForName returns the type value for the given name in the scope of the NameInput
func (i NameInput) TypeValueForName(ctx kitectx.Context, rm pythonresource.Manager, name string) (pythontype.Value, error) {
	table, _ := i.ResolvedAST().TableAndScope(i.Name)
	if table == nil {
		return nil, errors.Errorf("can't get table from RAST")
	}

	sym := table.Find(name)
	if sym == nil {
		return nil, errors.Errorf("can't find %v in symbol table", name)
	}

	val := sym.Value
	if val == nil {
		return nil, errors.Errorf("can't find value for a given symbol")
	}

	valType := sym.Value.Type()
	return pythontype.Translate(ctx, valType, rm), nil
}

// GetLabel returns the index of the mix input corresponding to what the user typed, or -1 if there is no match.
func GetLabel(userTyped []byte, comps []string) (int, error) {
	labels, err := getLabels(userTyped, comps)
	if err != nil {
		return -1, err
	}
	maxLength := len(comps[labels[0]])
	var longestMatch int
	for i := 1; i < len(labels); i++ {
		comp := comps[labels[i]]
		if len(comp) > maxLength {
			longestMatch = i
			maxLength = len(comp)
		}
	}
	return labels[longestMatch], nil
}

// getLabels returns the index of all the completion that are a prefix of what the user typed
func getLabels(userTyped []byte, comps []string) ([]int, error) {
	scannedWords, err := pythonscanner.Scan(userTyped)
	if err != nil {
		return nil, err
	}

	var labels []int
	for i, comp := range comps {
		scannedComp, err := pythonscanner.Scan([]byte(comp))
		if err != nil {
			return nil, err
		}

		matched := isPrefix(scannedWords, scannedComp)
		if matched {
			labels = append(labels, i)
		}
	}
	if len(labels) == 0 {
		return nil, errors.Errorf("No valid label for input")
	}
	return labels, nil
}

// isPrefix checks if prefix is a prefix of tokens
func isPrefix(tokens []pythonscanner.Word, prefix []pythonscanner.Word) bool {
	if len(tokens) < len(prefix) {
		return false
	}

	// skip the last prefix token before it's EOF
	// TODO: ignore whitespace while lexing?
	for i := 0; i < len(prefix)-1; i++ {
		p := prefix[i]
		t := tokens[i]

		if t.Literal != p.Literal || t.Token != p.Token {
			return false
		}
	}
	return true
}
