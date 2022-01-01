package util

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"go/token"
	"log"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kr/pretty"
)

var (
	scanOpts = pythonscanner.Options{
		ScanComments: false,
		ScanNewLines: false,
	}

	parseOpts = pythonparser.Options{
		ScanOptions: scanOpts,
		ErrorMode:   pythonparser.Recover,
	}

	resolveOpts = pythonanalyzer.Options{}

	emptyArgSpec = pythoncode.ArgSpec{}

	superAnyName = pythonimports.NewDottedPath("__builtin__.super")
)

// CallSpec encapsulates the data neccesary to generate
// a signature pattern from an example source code call expression.
type CallSpec struct {
	// AnyName for the node in the import graph associated with this call.
	AnyName pythonimports.DottedPath

	// Code is the code for the call, used for debugging.
	Code string

	// NodeArgSpec is the arg spec for the example call.
	NodeArgSpec *pythonimports.ArgSpec

	// Args are the positional arguments for the call.
	Args []*pythoncode.ArgSpec

	// Kwargs are the keyword arguments for the call.
	Kwargs []*pythoncode.ArgSpec
}

// String returns a string representation of the CallSpec.
func (cs *CallSpec) String() string {
	return pretty.Sprintf("%# v", cs)
}

// Valid  determines whether the arguments parsed in the CallSpec
// conform to the argspec in `NodeArgSpec`. # of positional arguments are enforced if there is no
// vararg, and keyword names are enforced if there is no **kwarg argument.
func (cs *CallSpec) Valid() bool {
	if cs.NodeArgSpec == nil {
		return true
	}

	// num (required) positional arguments
	var positional int
	// names for keyword arguments (not **kwargs!)
	kwargs := make(map[string]struct{})
	for _, arg := range cs.NodeArgSpec.Args {
		if arg.Name == "self" || arg.Name == "cls" {
			continue
		}
		if arg.DefaultType == "" {
			positional++
		} else {
			kwargs[arg.Name] = struct{}{}
		}
	}

	// Only enforce positional argument check if Vararg is empty
	if cs.NodeArgSpec.Vararg == "" {
		// the user could have passed keyword arguments as positional
		// arguments, thus we can only be sure the call is invalid
		// if we observed strictly fewer arguments than the required
		// number of positional arguments.
		if len(cs.Args) < positional {
			return false
		}
	}

	// Only enforce keyword argument check if Kwarg is empty
	// kwarg represents the **kwargs.
	if cs.NodeArgSpec.Kwarg == "" {
		for _, arg := range cs.Kwargs {
			if _, exists := kwargs[arg.Key]; !exists {
				return false
			}
		}
	}
	return true
}

// Snippet encapsulates the data neccesary to generate signature patterns
// from a block of code.
type Snippet struct {
	// Code is the source code for the snippet, this is used for deduping.
	Code string

	// Hash of the source code that generated the snippet, used for deduping
	Hash SnippetHash

	// Incantations is a slice of all of the calls in the snippet.
	Incantations []*CallSpec
	// Decorators is a slice of all of the decorator call expressions in the snippet
	Decorators []*CallSpec
}

// hash returns a hash of the input source.
func hash(src []byte) SnippetHash {
	var h SnippetHash
	spooky.Hash128(src, &h[0], &h[1])
	return h
}

// SnippetHash represents a 128-bit hash of the code in a snippet.
type SnippetHash [2]uint64

// String returns a base64-encoded string representation of the hash.
func (h SnippetHash) String() string {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, h[0])
	binary.Write(&buf, binary.LittleEndian, h[1])
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// Params encapsulates the parameters required to extract snippets.
type Params struct {
	Graph       *pythonimports.Graph
	ArgSpecs    pythonimports.ArgSpecs
	TypeInducer *typeinduction.Client
	AnyNames    map[*pythonimports.Node]pythonimports.DottedPath
}

// Extract extracts a Snippet to represent the block of code in src for signature pattern generation.
func Extract(src []byte, params Params) *Snippet {
	if params.Graph == nil || params.ArgSpecs == nil || params.TypeInducer == nil || params.AnyNames == nil {
		log.Fatal("invalid extraction parameters")
	}
	// parse
	mod, _ := pythonparser.Parse(kitectx.Background(), src, parseOpts)
	if pythonast.IsNil(mod) {
		return nil
	}

	// resolve
	resolver := pythonanalyzer.NewResolver(params.Graph, params.TypeInducer, resolveOpts)
	resolved, err := resolver.Resolve(mod)
	if err != nil {
		return nil
	}

	// call specs for (non decorator) call expressions
	calls := calls(mod)
	incs := callSpecs(calls, src, resolved, params)

	// call specs for decorator call expressions
	calls = decoratorCalls(resolved)
	decs := callSpecs(calls, src, resolved, params)

	// check if snippet has calls that were resolved
	if len(incs) == 0 && len(decs) == 0 {
		return nil
	}

	return &Snippet{
		Hash:         hash(src),
		Incantations: incs,
		Decorators:   decs,
	}
}

func callSpecs(
	calls []*pythonast.CallExpr,
	src []byte,
	resolved *pythonanalyzer.ResolvedAST,
	params Params) []*CallSpec {

	var specs []*CallSpec
	for _, call := range calls {
		ref, isResolved := resolved.References[call.Func]
		if !isResolved || ref.Node == nil || ref.Node.Origin != pythonimports.GlobalGraph {
			continue
		}

		anyname := params.AnyNames[ref.Node]
		if anyname.Empty() {
			continue
		}

		// check if this is a type constructor, and if the type has an __init__ method
		// then group this usage under the __init__ usage.
		// SEE: kite-go.lang.python.pythoncode.NewSignaturePatterns
		// we ignore super since it is a type in python and it also has an init method.
		if ref.Node.Classification == pythonimports.Type && anyname.Hash != superAnyName.Hash {
			if initNode := ref.Node.Members["__init__"]; initNode != nil {
				if initAnyName := params.AnyNames[initNode]; !initAnyName.Empty() {
					anyname = initAnyName
				}
			}
		}

		specs = append(specs, &CallSpec{
			AnyName:     anyname,
			Code:        typed(src, call),
			NodeArgSpec: nodeArgSpec(ref, params.ArgSpecs),
			Args:        argSpecs(resolved, src, call.Args, false, params.AnyNames),
			Kwargs:      argSpecs(resolved, src, call.Args, true, params.AnyNames),
		})
	}
	return specs
}

func decoratorCalls(resolved *pythonanalyzer.ResolvedAST) []*pythonast.CallExpr {
	var decorators []*pythonast.CallExpr
	for fn := range resolved.Functions {
		for _, expr := range fn.Decorators {
			switch expr := expr.(type) {
			case *pythonast.CallExpr:
				decorators = append(decorators, expr)
			}
		}
	}
	return decorators
}

func calls(mod *pythonast.Module) []*pythonast.CallExpr {
	var calls []*pythonast.CallExpr
	pythonast.InspectEdges(mod, func(parent, child pythonast.Node, field string) bool {
		if pythonast.IsNil(child) {
			return false
		}

		if pythonast.IsNil(parent) {
			// must be at module, recur into children
			return true
		}

		if call, isCall := child.(*pythonast.CallExpr); isCall {
			// check to make sure not a decorator
			if _, isFnDef := parent.(*pythonast.FunctionDefStmt); isFnDef && field == "Decorators" {
				return true
			}
			calls = append(calls, call)
		}

		return true
	})
	return calls
}

func argSpecs(
	resolved *pythonanalyzer.ResolvedAST,
	src []byte,
	args []*pythonast.Argument,
	keywords bool,
	anynames map[*pythonimports.Node]pythonimports.DottedPath) []*pythoncode.ArgSpec {

	var specs []*pythoncode.ArgSpec
	for _, arg := range args {
		if (keywords && pythonast.IsNil(arg.Name)) || (!keywords && !pythonast.IsNil(arg.Name)) {
			continue
		}

		var typename string
		switch arg.Value.(type) {
		case *pythonast.LambdaExpr:
			typename = "__builtin__.function"
		default:
			ref := resolved.References[arg.Value]
			switch {
			case ref == nil || ref.Node == nil:
			case ref.Node.Type != nil && ref.Node.Type.Origin == pythonimports.GlobalGraph:
				typename = anynames[ref.Node.Type].String()
			}
		}

		spec := pythoncode.ArgSpec{
			Key:  typed(src, arg.Name),
			Type: typename,
		}

		if pythonast.IsLiteral(arg.Value) {
			spec.Literal = typed(src, arg.Value)
		} else {
			spec.ExprStr = typed(src, arg.Value)
		}

		if spec == emptyArgSpec {
			continue
		}

		specs = append(specs, &spec)
	}
	return specs
}

func nodeArgSpec(ref *pythonanalyzer.Reference, argSpecs pythonimports.ArgSpecs) *pythonimports.ArgSpec {
	if ref == nil || ref.Node == nil || argSpecs == nil {
		return nil
	}

	if as, found := argSpecs[ref.Node.ID]; found {
		return as
	}

	if ref.Node.Type == nil {
		return nil
	}
	return argSpecs[ref.Node.Type.ID]
}

func isVarName(expr pythonast.Expr) bool {
	switch expr := expr.(type) {
	case *pythonast.NameExpr:
		return true
	case *pythonast.AttributeExpr:
		return isVarName(expr.Value)
	default:
		return false
	}
}

func typed(src []byte, node pythonast.Node) string {
	if pythonast.IsNil(node) || node.Begin() >= token.Pos(len(src)) || node.End() > token.Pos(len(src)) {
		return ""
	}
	return string(src[node.Begin():node.End()])
}
