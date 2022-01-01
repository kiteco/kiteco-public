package python

import (
	"go/token"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// CalleeResult contains an editorapi.CalleeResponse object (if GetCallee was successful) along with related
// information. A CalleeResult can contain both a valid Response a Failure; these are not mutually exclusive.
type CalleeResult struct {
	Response      *editorapi.CalleeResponse
	Failure       pythontracking.CalleeFailure
	OutsideParens bool
	CallExpr      *pythonast.CallExpr
	CalleeValue   pythontype.Value
}

// CalleeInputs are used to compute CalleeResult
type CalleeInputs struct {
	Buffer []byte
	Cursor int64

	Words      []pythonscanner.Word
	Resolved   *pythonanalyzer.ResolvedAST
	LocalIndex *pythonlocal.SymbolIndex

	Services *Services

	BufferIndex *bufferIndex
}

// NewCalleeInputs is used to create a new set of inputs, this should only be called
// when the lock for pyctx has been acquired, we need to be careful what
// fields are accessed here since these inputs may be used in a separate go routine
// to calculate the GGNN call completions.
// NOTE:
//  - it is safe to grab a reference to pyctx.Buffer since this is a copy of the underlying file buffer
//  - it is safe to call pyctx.IncrLexer.Words() since this returns a read only slice and the lock
//    for pyctx has been acquired.
//  - it is safe to grab a reference to pyctx.Resolved since this is a read only map and the lock for
//    pyctx has been acquired. NOTE: we temorarily modify pyctx.Resolved.Root during GGNN call completions.
//    TODO: fix this
func NewCalleeInputs(pyctx *Context, cursor int64, services *Services) CalleeInputs {
	return CalleeInputs{
		Buffer:      pyctx.Buffer,
		Cursor:      cursor,
		Words:       pyctx.IncrLexer.Words(),
		Resolved:    pyctx.Resolved,
		LocalIndex:  pyctx.LocalIndex,
		BufferIndex: pyctx.BufferIndex,
		Services:    services,
	}
}

// GetCallee extracts the callee relevant to the callee request and builds an editorapi.CalleeResponse object
// containing the callee and relevant information.
func GetCallee(ctx kitectx.Context, in CalleeInputs) CalleeResult {
	ctx.CheckAbort()

	var result CalleeResult

	callExpr, outsideParens, _ := FindCallExpr(ctx, in.Resolved.Root, in.Buffer, in.Cursor)
	if callExpr == nil {
		result.Failure = pythontracking.NoCallExprFailure
		return result
	} else if outsideParens {
		result.Failure = pythontracking.OutsideParensFailure
		return result
	}

	result.CallExpr = callExpr
	funcExpr := callExpr.Func

	ref := in.Resolved.References[funcExpr]
	if ref == nil {
		result.Failure = pythontracking.UnresolvedValueFailure
		return result
	}

	var val pythontype.Value
	for _, v := range pythontype.Disjuncts(ctx, ref) {
		if v.Kind() == pythontype.FunctionKind || v.Kind() == pythontype.TypeKind {
			val = v
			break
		}
	}
	if val == nil {
		result.Failure = pythontracking.InvalidKindFailure
		return result
	}
	result.CalleeValue = val

	editor := newEditorServices(in.Services)
	vb := newValueBundle(ctx, val, indexBundle{
		idx:   in.LocalIndex,
		graph: in.Services.ResourceManager,
		bi:    in.BufferIndex,
	})

	if vb.val == nil {
		result.Failure = pythontracking.ValTranslateFailure
		return result
	}

	sigs := editor.renderSignatures(ctx, vb)

	result.Response = &editorapi.CalleeResponse{
		Language:   lang.Python.Name(),
		FuncName:   val.Address().Path.Last(),
		Callee:     editor.renderValueExt(ctx, vb),
		Report:     editor.renderValueReport(ctx, vb),
		Signatures: sigs,
	}
	if len(result.Response.Signatures) == 0 {
		result.Failure = pythontracking.NoSignaturesFailure
	}
	return result
}

// FindCallExpr at the specified cursor position and return true
// if the cursor is within the parentheses for the returned call expression.
func FindCallExpr(ctx kitectx.Context, ast pythonast.Node, buf []byte, cursor int64) (expr *pythonast.CallExpr, outsideParens bool, inBadNode bool) {
	ctx.CheckAbort()

	// move to the nearest non-space character before the cursor so that we handle incomplete calls correctly
	cursor = pythonhelpers.NearestNonWhitespace(buf, cursor, unicode.IsSpace)

	visitor := calleeVisitor{cursor: cursor, expr: &expr, inBadNode: &inBadNode, ctx: ctx}
	pythonast.Walk(visitor, ast)
	if expr != nil {
		outsideParens = !pythonhelpers.CursorBetweenCallParens(expr, token.Pos(cursor))
	}
	return
}

type calleeVisitor struct {
	cursor   int64
	curInBad bool
	// write the output to these variables
	expr      **pythonast.CallExpr
	inBadNode *bool
	ctx       kitectx.Context // store here because no other way
}

// Visit implements pythonast.NodeVisitor
func (f calleeVisitor) Visit(n pythonast.Node) pythonast.Visitor {
	f.ctx.CheckAbort()

	if n == nil || !pythonhelpers.UnderCursor(n, f.cursor) {
		return nil
	}
	switch n := n.(type) {
	case *pythonast.CallExpr:
		if *f.expr == nil {
			*f.expr = n
			*f.inBadNode = f.curInBad
			return f
		}
		// Nesting
		// NOTE: in order to properly handle chained calls we MUST use >= here
		if n.Begin() >= (*f.expr).Begin() {
			insideParens := pythonhelpers.CursorBetweenCallParens(n, token.Pos(f.cursor))
			if insideParens {
				*f.expr = n
				*f.inBadNode = f.curInBad
			}
		}
	case *pythonast.BadExpr:
		return calleeVisitor{cursor: f.cursor, curInBad: true, expr: f.expr, inBadNode: f.inBadNode, ctx: f.ctx}
	case *pythonast.BadStmt:
		return calleeVisitor{cursor: f.cursor, curInBad: true, expr: f.expr, inBadNode: f.inBadNode, ctx: f.ctx}
	}
	return f
}
