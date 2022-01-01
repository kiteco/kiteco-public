package traindata

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/fileutil"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

const (
	// SubtokenIndexPath is the path to the current name subtoken index
	SubtokenIndexPath = "s3://kite-data/python-index-subtokens/2019-01-29_06-49-45-AM.index.json"
	// NewSubtokenIndexPath is the path to the name subtoken index from the new github dump
	NewSubtokenIndexPath = "s3://kite-data/python-index-subtokens/2019-08-20_04-19-07-PM.index.json"
)

// SubtokenIndex maps subtokens to their row index in the subtoken embedding matrix.
// Separate subtoken indices are maintained for both names (literals) and for type names.
type SubtokenIndex map[string]int

// NewSubtokenIndex using the serialized index at path as the base
// index and appending the special tokens afterwards.
func NewSubtokenIndex(base string) (SubtokenIndex, error) {
	r, err := fileutil.NewCachedReader(base)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var idx SubtokenIndex
	if err := json.NewDecoder(r).Decode(&idx); err != nil {
		return nil, err
	}
	idx.AddSpecialSubtokens()
	return idx, nil
}

// AddSpecialSubtokens at end
func (s SubtokenIndex) AddSpecialSubtokens() {
	for _, sp := range specialSubtokens {
		if _, ok := s[sp]; !ok {
			s[sp] = len(s)
		}
	}
}

// Index of the specified sub token
func (s SubtokenIndex) Index(t string) int {
	// this is a hack to avoid a model retrain, since in the currently trained model NAType and UnknownType were both
	// represented as UnknownTokenMarker
	// TODO: remove this before the next training run
	if t == UnknownTypeMarker || t == NATypeMarker {
		return s[UnknownTokenMarker]
	}
	if !IsSpecialToken(t) {
		t = strings.ToLower(t)
	}

	i, ok := s[t]
	if !ok {
		return s[UnknownTokenMarker]
	}
	return i
}

const (
	// MarkerNamespace is used to prefix all special marker tokens to ensure
	// that they do not collide with subtokens extracted from github
	MarkerNamespace = "KITE_"
	// EOFMarker is used to mark the node associated with the EOF
	EOFMarker = MarkerNamespace + "EOF"
	// SOFMarker is a special marker for the pseudo ast terminal node that marks the start of the
	// file, note that the pythonscanner package already includes an EOF marker.
	// TODO(juan): add this to pythonscanner package?
	SOFMarker = MarkerNamespace + "SOF"
	// InferNameMarker is used to mark the "context node" for infer name tasks
	InferNameMarker = MarkerNamespace + "CONTEXTNODE"
	// InferAttrMarker is used to mark the prediction node for infer attr tasks
	InferAttrMarker = MarkerNamespace + "ATTRPLACEHOLDER"
	// InferArgTypeMarker is used to mark the prediction node for infer arg type tasks
	InferArgTypeMarker = MarkerNamespace + "ARGNODE"
	// InferKwargNameMarker is used to mark the prediction node for infer kwarg name tasks
	InferKwargNameMarker = MarkerNamespace + "KWARGNAME"
	// KwargValuePlaceholder is the placeholder for dummy keyword argument value for creating kwarg name training samples
	KwargValuePlaceholder = MarkerNamespace + "KWARG_VALUE_PLACEHOLDER"
	// ScopeNodeMarker is used to make the scope node
	ScopeNodeMarker = MarkerNamespace + "SCOPE"
	// InferArgPlaceholderMarker is the placeholder for the site evaluated by the arg placeholder task
	InferArgPlaceholderMarker = MarkerNamespace + "ARGPLACEHOLDER"

	// the following markers are applicable to the type subtoken index
	//

	// UnknownTypeMarker is the subtoken that represents an unknown type
	UnknownTypeMarker = MarkerNamespace + "UNKNOWN_TYPE"
	// NATypeMarker ins the subtoken that represents a not-applicable type
	NATypeMarker = MarkerNamespace + "NA_TYPE"

	// AttrBaseNameDecoder is the marker for the decoder embedding used when choosing a name expression to use
	// as an attribute base
	AttrBaseNameDecoder = MarkerNamespace + "ATTR_BASE_NAME_DECODER"

	// UnknownTokenMarker ...
	UnknownTokenMarker = MarkerNamespace + "UNKNOWN34159871"

	// InferExprTypeMarker is used to mark the prediction node for infer expr type tasks
	InferExprTypeMarker = MarkerNamespace + "INFER_EXPR_TYPE"

	// ChooseTerminalTypeMarker ...
	ChooseTerminalTypeMarker = MarkerNamespace + "CHOOSE_TERMINAL_TYPE"
)

// IsSpecialToken test is the type corresponds to a marker or another special token
func IsSpecialToken(s string) bool {
	_, isMarker := specialSubtokensMap[s]
	return isMarker
}

// WordLiteral returns the literal that should be placed in the
// graph node for the specified word.
func WordLiteral(w pythonscanner.Word) string {
	if w.Token == pythonscanner.EOF {
		return EOFMarker
	}

	if w.Token == pythonscanner.Ident {
		// TODO: hacky, this can happen for nodes
		// that come from the approx parser.
		if w.Literal == "" {
			return UnknownTokenMarker
		}
		return w.Literal
	}

	return MarkerNamespace + w.Token.String()
}

// ASTNodeLiteral returns the literal string that should be placed in the graph
// for a specified ast node
func ASTNodeLiteral(node pythonast.Node) string {
	switch node := node.(type) {
	case *pythonast.NameExpr:
		return node.Ident.Literal
	case *pythonast.EllipsisExpr:
		return MarkerNamespace + "ELLIPSIS"
	case *pythonast.PassStmt:
		return MarkerNamespace + "PASS"
	case *pythonast.ContinueStmt:
		return MarkerNamespace + "CONTINUE"
	case *pythonast.BreakStmt:
		return MarkerNamespace + "BREAK"
	case *pythonast.StringExpr:
		return MarkerNamespace + "STRING"
	case *pythonast.NumberExpr:
		return MarkerNamespace + "NUMBER"
	default:
		return ""
	}
}

// ASTNodeType returns a string for the specified ast node
// NOTE: these should never be split with SplitNameLiteral
// but just for safety we capitalize the entire string anyways
func ASTNodeType(n pythonast.Node) string {
	if n == nil {
		return MarkerNamespace + "NIL"
	}
	tn := typename(n)
	if _, ok := n.(pythonast.Expr); ok && !strings.HasSuffix(tn, "Expr") {
		tn = tn + "Expr"
	}
	return MarkerNamespace + strings.ToUpper(tn)
}

// specialSubtokens that can appear in graph nodes
var specialSubtokens []string
var specialSubtokensMap map[string]bool

func init() {
	specialSubtokens = append(specialSubtokens,
		SOFMarker,
		InferNameMarker,
		InferAttrMarker,
		InferArgTypeMarker,
		InferKwargNameMarker,
		InferArgPlaceholderMarker,
		KwargValuePlaceholder,
		ScopeNodeMarker,
		AttrBaseNameDecoder,
		UnknownTokenMarker,
		InferExprTypeMarker,
		// these are used for the type subtoken index
		UnknownTypeMarker,
		NATypeMarker,
	)
	for _, tok := range pythonscanner.Tokens {
		if tok == pythonscanner.EOF {
			specialSubtokens = append(specialSubtokens, EOFMarker)
			continue
		}
		specialSubtokens = append(specialSubtokens, MarkerNamespace+tok.String())
	}
	for _, node := range pythonast.NodeList {
		specialSubtokens = append(specialSubtokens, ASTNodeType(node))
		if _, ok := node.(*pythonast.NameExpr); ok {
			// the literal for name nodes is the actual identifier
			continue
		}

		if s := ASTNodeLiteral(node); s != "" {
			specialSubtokens = append(specialSubtokens, s)
		}
	}
	specialSubtokensMap = make(map[string]bool, len(specialSubtokens))
	for _, st := range specialSubtokens {
		specialSubtokensMap[st] = true
	}
}

func derefType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		return derefType(t.Elem())
	default:
		return t
	}
}

func typename(obj interface{}) string {
	return derefType(reflect.TypeOf(obj)).Name()
}
