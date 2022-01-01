package main

import (
	"fmt"
	"go/token"
	"reflect"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/internal/inspectorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

func getExprListings(ctx *python.Context) []inspectorapi.ExprListing {
	var listings []inspectorapi.ExprListing
	for _, expr := range getAllNameExprs(ctx) {
		listings = append(listings, getNameExprListing(expr, ctx))
	}
	sort.Slice(listings, func(i, j int) bool {
		return listings[i].Begin < listings[j].Begin
	})
	return listings
}

func getExprDetail(exprCursor int64, ctx *python.Context) inspectorapi.ExprDetail {
	var expr *pythonast.NameExpr
	for _, e := range getAllNameExprs(ctx) {
		if e.Begin() == token.Pos(exprCursor) {
			expr = e
			break
		}
	}
	if expr == nil {
		return inspectorapi.ExprDetail{}
	}

	detail := inspectorapi.ExprDetail{
		Name:     expr.Ident.Literal,
		Cursor:   exprCursor,
		ExprType: reflect.TypeOf(expr).String(),
		Begin:    int64(expr.Begin()),
		End:      int64(expr.End()),
	}

	resolvedVal := ctx.Resolved.References[expr]
	if resolvedVal == nil {
		return detail
	}
	detail.ResolvedValue = getValueDetail(resolvedVal, ctx)

	return detail
}

func getAllNameExprs(ctx *python.Context) []*pythonast.NameExpr {
	var exprs []*pythonast.NameExpr

	pythonast.Inspect(ctx.AST, func(n pythonast.Node) bool {
		if n == nil {
			return false
		}
		switch v := n.(type) {
		case *pythonast.NameExpr:
			exprs = append(exprs, v)
			return false
		}
		return true
	})
	return exprs
}

func getNameExprListing(expr pythonast.Expr, ctx *python.Context) inspectorapi.ExprListing {
	var resolvesTo string
	ref := ctx.Resolved.References[expr]
	if ref != nil {
		resolvesTo = fmt.Sprintf("%v", ref)
	}
	return inspectorapi.ExprListing{
		ExprType:   reflect.TypeOf(expr).String(),
		Begin:      int64(expr.Begin()),
		End:        int64(expr.End()),
		ResolvesTo: resolvesTo,
	}
}

func getValueDetail(val pythontype.Value, ctx *python.Context) inspectorapi.ValueDetail {
	if union, ok := val.(pythontype.Union); ok {
		var constituents []inspectorapi.ValueDetail
		for _, v := range union.Constituents {
			constituents = append(constituents, getValueDetail(v, ctx))
		}

		return inspectorapi.ValueDetail{
			Repr:         fmt.Sprintf("%v", val),
			Kind:         val.Kind().String(),
			Type:         reflect.TypeOf(val).String(),
			Constituents: constituents,
		}
	}

	address := val.Address().String()

	var globalType string
	var canonicalName string

	val = pythontype.TranslateNoCtx(val, ctx.Importer.Global)
	if global, ok := val.(pythontype.GlobalValue); ok {
		globalType = reflect.TypeOf(global).String()
		switch global := global.(type) {
		case pythontype.ExternalInstance:
			canonicalName = global.TypeExternal.Symbol().Canonical().String()
		case pythontype.External:
			canonicalName = global.Symbol().Canonical().String()
		}
	}

	kind := pythontype.UnknownKind
	if val != nil {
		kind = val.Kind()
	}

	return inspectorapi.ValueDetail{
		Repr:          fmt.Sprintf("%v", val),
		Kind:          kind.String(),
		Type:          reflect.TypeOf(val).String(),
		Address:       address,
		GlobalType:    globalType,
		CanonicalName: canonicalName,
	}
}
