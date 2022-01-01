package pythonmetrics

import (
	"fmt"
	"go/token"
	"io"
	"reflect"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

func derefType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		return derefType(t.Elem())
	default:
		return t
	}
}

// ASTNodeTypeOf the specified ast node
func ASTNodeTypeOf(n pythonast.Node) string {
	t := derefType(reflect.TypeOf(n))
	return t.Name()
}

// WordInfo encapsulates anonymized information about a lexed Word
type WordInfo struct {
	Token string
	Begin token.Pos
	End   token.Pos
}

// BadStmtInfo encapsulates the tokens and errors associated with a BadStmt
type BadStmtInfo struct {
	Words     []WordInfo
	Errors    []pythonscanner.PosError
	NumWords  int
	NumErrors int
}

func wordsAroundCursor(words []pythonscanner.Word, cursor token.Pos, limit int) []pythonscanner.Word {
	afterStart := len(words)
	for i, word := range words {
		if word.End > cursor {
			afterStart = i
			break
		}
	}

	start := afterStart - limit
	end := afterStart + limit
	if start < 0 {
		start = 0
	}
	if end > len(words) {
		end = len(words)
	}
	return words[start:end]
}

func newBadStmtInfo(bad *pythonast.BadStmt, words []pythonscanner.Word, errs errors.Errors, wordLimit, errorLimit int) *BadStmtInfo {
	var info BadStmtInfo

	for _, word := range words {
		if word.End <= bad.Begin() {
			continue
		}
		if word.Begin >= bad.End() { // words are in order
			break
		}
		info.Words = append(info.Words, WordInfo{word.Token.String(), word.Begin, word.End})
	}
	if errs != nil {
		for _, err := range errs.Slice() {
			posErr, ok := err.(pythonscanner.PosError)
			if !ok {
				rollbar.Error(errors.New("non-PosError returned by ParseWords"))
				continue
			}
			if posErr.Pos < bad.Begin() || posErr.Pos >= bad.End() { // errors may be out of order
				continue
			}
			info.Errors = append(info.Errors, posErr)
		}
	}

	info.NumWords = len(info.Words)
	info.NumErrors = len(info.Errors)
	if wordLimit != 0 && len(info.Words) > wordLimit {
		info.Words = info.Words[:wordLimit]
	}
	if errorLimit != 0 && len(info.Errors) > errorLimit {
		info.Errors = info.Errors[:errorLimit]
	}

	return &info
}

// ASTTraceNode describes an anonymized node in an AST trace
type ASTTraceNode struct {
	Begin, End  token.Pos
	Type        string
	BadStmtInfo *BadStmtInfo
}

func (n ASTTraceNode) string(includePositions bool) string {
	ts := n.Type
	if includePositions {
		return fmt.Sprintf("%d:%d::%s", n.Begin, n.End, ts)
	}
	return ts
}

// String representation of the node
func (n ASTTraceNode) String() string {
	return n.string(true)
}

func newASTTraceNode(n pythonast.Node, words []pythonscanner.Word, errs errors.Errors, wordLimit, errorLimit int) ASTTraceNode {
	tn := ASTTraceNode{
		Begin: n.Begin(),
		End:   n.End(),
		Type:  ASTNodeTypeOf(n),
	}
	if bad, ok := n.(*pythonast.BadStmt); ok {
		tn.BadStmtInfo = newBadStmtInfo(bad, words, errs, wordLimit, errorLimit)
	}
	return tn
}

// ASTTrace describes a DFS path in an AST from root to cursor position along with other relevant information
type ASTTrace struct {
	NumNodes int
	Nodes    []ASTTraceNode
	Edges    []string
	Cursor   token.Pos
}

// Print the ast trace to the specified writer
func (a ASTTrace) Print(w io.Writer, includePositions bool) {
	fmt.Fprintln(w, fmt.Sprintf("Trace to %d:", a.Cursor))
	for i := 0; i < len(a.Nodes); i++ {
		n := a.Nodes[i]
		if i == 0 {
			fmt.Fprintln(w, n.string(includePositions))
			continue
		}
		e := a.Edges[i-1]
		fmt.Fprintln(w, fmt.Sprintf("- %s ->", e))
		fmt.Fprintln(w, n.string(includePositions))
	}
}

// ASTTraceInputs for NewASTTraceEvent
type ASTTraceInputs struct {
	AST         pythonast.Node
	Cursor      int64
	Words       []pythonscanner.Word
	ParseErrors errors.Errors

	// NodeLimit limits the trace to the deepest n nodes under the cursor
	NodeLimit int
	// WordLimit limits BadStmt traces to the first n Words under the BadStmt
	WordLimit int
	// ErrorLimit limits BadStmt traces to the first n Errors for the BadStmt
	ErrorLimit int
}

// NewASTTrace constructs a new ast trace from the specified ast and cursor position
func newASTTrace(inps ASTTraceInputs) ASTTrace {
	var nodes []ASTTraceNode
	var edges []string
	pythonast.InspectEdges(inps.AST, func(parent, child pythonast.Node, edge string) bool {
		if !pythonhelpers.UnderCursor(child, inps.Cursor) {
			return false
		}

		nodes = append(nodes, newASTTraceNode(child, inps.Words, inps.ParseErrors, inps.WordLimit, inps.ErrorLimit))
		if !pythonast.IsNil(parent) { // happens only at the root
			edges = append(edges, edge)
		}
		return true
	})

	numNodes := len(nodes)
	if inps.NodeLimit != 0 && numNodes > inps.NodeLimit {
		nodes = nodes[numNodes-inps.NodeLimit:]
		// inps.NodeLimit > 0, and len(edges) == numNodes-1, so these bounds are in range:
		edges = edges[numNodes-inps.NodeLimit:]
	}

	return ASTTrace{
		NumNodes: numNodes,
		Nodes:    nodes,
		Edges:    edges,
		Cursor:   token.Pos(inps.Cursor),
	}
}

// ASTTraceEvent is the actual event ast trace event sent to segment
type ASTTraceEvent struct {
	Trace ASTTrace
}

// NewASTTraceEvent from the specified ast and cursor position
func NewASTTraceEvent(inps ASTTraceInputs) ASTTraceEvent {
	trace := newASTTrace(inps)
	return ASTTraceEvent{
		Trace: trace,
	}
}
