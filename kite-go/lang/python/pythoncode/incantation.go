package pythoncode

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"go/token"
	"sort"
	"strings"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// ArgSpec collects information about each argument in an Incantation
// TODO(juan): add fields to track more general expressions for arguments
// such as foo(1 + 2), foo(bar()) ?
type ArgSpec struct {
	Key     string
	ExprStr string
	Type    string
	Literal string
}

// String returns a string representation of an ArgSpec.
func (as *ArgSpec) String() string {
	return fmt.Sprintf("Key: %s, ExprStr: %s, Type: %s, Literal: %s", as.Key, as.ExprStr, as.Type, as.Literal)
}

// Empty returns whether or not this ArgSpec holds data.
func (as *ArgSpec) Empty() bool {
	return as.Key == "" && as.ExprStr == "" && as.Type == "" && as.Literal == ""
}

type argSpecByKey []*ArgSpec

func (b argSpecByKey) Len() int           { return len(b) }
func (b argSpecByKey) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b argSpecByKey) Less(i, j int) bool { return b[i].Key < b[j].Key }

// --

// CallSpec represents a call of a function. A CallSpec stores the value of
// the function being called as well as the positional and keyword arguments
// used in the call.
//
// A CallSpec can be hashed to compare itself to other CallSpec objects. Two
// calls are considered equal if they have the same number of positional
// arguments and the same set of keyword arguments. This hash can thus be used
// to compute frequency of a set equivalent call specs.
type CallSpec struct {
	Args   []*ArgSpec
	Kwargs []*ArgSpec
}

// NewCallSpec creates a new CallSpec object from a call expression, source code
// and a map to resolve expressions to values.
func NewCallSpec(
	ctx kitectx.Context,
	call *pythonast.CallExpr,
	src []byte, refs map[pythonast.Expr]pythontype.Value) *CallSpec {
	ctx.CheckAbort()

	if _, exists := refs[call]; !exists {
		return nil
	}
	return &CallSpec{
		Args:   argSpecs(ctx, call.Args, false, src, refs),
		Kwargs: argSpecs(ctx, call.Args, true, src, refs),
	}
}

func argSpecs(
	ctx kitectx.Context,
	args []*pythonast.Argument, kw bool,
	src []byte, refs map[pythonast.Expr]pythontype.Value) []*ArgSpec {
	ctx.CheckAbort()

	var specs []*ArgSpec
	for _, arg := range args {
		if spec := newArgSpec(ctx, arg, kw, src, refs); spec != nil && !spec.Empty() {
			specs = append(specs, spec)
		}
	}
	return specs
}

func newArgSpec(
	ctx kitectx.Context,
	arg *pythonast.Argument, kw bool,
	src []byte, refs map[pythonast.Expr]pythontype.Value) *ArgSpec {
	ctx.CheckAbort()

	// Check argument constraints
	if (kw && pythonast.IsNil(arg.Name)) || (!kw && !pythonast.IsNil(arg.Name)) {
		return nil
	}

	// Set the type
	var typename string
	switch arg.Value.(type) {
	case *pythonast.LambdaExpr:
		typename = "types.FunctionType"
	default:
		if val := refs[arg.Value]; val != nil && val.Type() != nil {
			typename = pythonenv.Locator(val.Type())
		}
	}

	// Create arg spec and set literal or variable name field
	spec := &ArgSpec{
		Key:  typed(src, arg.Name),
		Type: typename,
	}
	if pythonast.IsLiteral(arg.Value) {
		spec.Literal = typed(src, arg.Value)
	} else {
		spec.ExprStr = typed(src, arg.Value)
	}

	return spec
}

// String returns a string representation of a CallSpec.
func (s *CallSpec) String() string {
	var args []string
	for _, arg := range s.Args {
		args = append(args, "("+arg.String()+")")
	}
	var kwargs []string
	for _, arg := range s.Kwargs {
		kwargs = append(kwargs, "("+arg.String()+")")
	}
	return fmt.Sprintf("Args: [%s], Kwargs: [%s]", strings.Join(args, ", "), strings.Join(kwargs, ", "))
}

// hash returns the hash of a CallSpec to be used to check for equivalence with
// other CallSpec objects. A CallSpec's hash is a combination of the number of
// positional arguments and the set of keyword arguments used in the call.
func (s *CallSpec) hash() string {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, int32(len(s.Args)))
	kwargs := make([]*ArgSpec, len(s.Kwargs))
	copy(kwargs, s.Kwargs)
	sort.Sort(argSpecByKey(kwargs))
	for _, arg := range kwargs {
		buf.Write([]byte(arg.Key))
	}

	var fp [2]uint64
	spooky.Hash128(buf.Bytes(), &fp[0], &fp[1])
	buf.Reset()

	binary.Write(buf, binary.LittleEndian, fp[0])
	binary.Write(buf, binary.LittleEndian, fp[1])
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// --

// Incantation represents an example of a particular function call
type Incantation struct {
	ExampleOf string   // What the code is an example of.
	Snippet   *Snippet `json:"-"` // Omitted from json to avoid cycle

	Code           string // Code of function call
	LineNumber     int    // Line number within snippet containing function call
	Nested         bool   // Whether the call was a nested call
	NumArgs        int    // Number of args in this function call
	NumLiteralArgs int    // Number of literals amont arguments in function call

	// Python-specific Features
	NumKeywordArgs        int      // Number of keyword arguments
	NumLiteralKeywordArgs int      // Number of literal keyword arguments
	HasStarArgs           bool     // Whether arguments contains *arg
	HasStarKwargs         bool     // Whether arguments contains **kwargs
	Keywords              []string // Keywords used in keyword arguments

	Args   []*ArgSpec
	Kwargs []*ArgSpec

	Key string
}

// Score returns rough hueristic-based score describing how good this incantaiton is.
// Lower is better. The idea is to favor code snippets that are small, and to penalize
// any non-literal arguments (positional or keyword)
func (inc *Incantation) Score() int64 {
	return int64(inc.Snippet.Area + len(inc.Code) + inc.NumArgs + inc.NumKeywordArgs - inc.NumLiteralArgs - inc.NumLiteralKeywordArgs)
}

// ByScore is a wrapper for a slice of Incantation that implements the sort.Sort interface.
type ByScore []*Incantation

func (b ByScore) Len() int           { return len(b) }
func (b ByScore) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByScore) Less(i, j int) bool { return b[i].Score() < b[j].Score() }

// --

// typed returns the string literal of an AST node in source code.
func typed(src []byte, node pythonast.Node) string {
	if pythonast.IsNil(node) || node.Begin() >= token.Pos(len(src)) || node.End() > token.Pos(len(src)) {
		return ""
	}
	return string(src[node.Begin():node.End()])
}
