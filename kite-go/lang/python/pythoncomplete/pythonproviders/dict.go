package pythonproviders

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// DictKeys provides completions for the known keys of a dictionary
type DictKeys struct{}

// Name implements Provider
func (DictKeys) Name() data.ProviderName {
	return data.PythonDictKeysProvider
}

type keyCompletion struct {
	literal string
	isStr   bool
	value   pythontype.Value
	score   float64
}

func (kc keyCompletion) quoted(quote string) string {
	if !kc.isStr {
		return kc.literal
	}
	return pythonscanner.QuoteString(quote, kc.literal)
}

// Provide implements Provider
func (dp DictKeys) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	applicableCase, err := dp.provideAttributeCompletion(ctx, g, in, out)
	if applicableCase {
		return err
	}

	applicableCase, err = dp.provideBracketCompletion(ctx, g, in, out)
	if applicableCase {
		return err
	}

	applicableCase, err = dp.provideGetPopCompletions(ctx, g, in, out)
	if applicableCase {
		return err
	}

	return data.ProviderNotApplicableError{}
}

func (dp DictKeys) provideBracketCompletion(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) (bool, error) {
	ctx.CheckAbort()
	var indexExpr *pythonast.IndexExpr
	underPos := in.UnderSelection()
	for i := len(underPos) - 1; i >= 0; i-- {
		n := underPos[i]
		if idx, _ := n.(*pythonast.IndexExpr); idx != nil {
			if in.Selection.Begin >= int(idx.LeftBrack.End) &&
				(idx.RightBrack == nil || in.Selection.End <= int(idx.RightBrack.Begin)) {
				indexExpr = idx
				// We don't break as we want the deepest indexExpr as that's the one will autocomplete
			}
		}
	}

	if indexExpr == nil {
		return false, data.ProviderNotApplicableError{}
	}

	val := in.ResolvedAST().References[indexExpr.Value]
	dict, ok := val.(pythontype.DictLike)
	if !ok {
		return true, data.ProviderNotApplicableError{}
	}

	if len(indexExpr.Subscripts) > 1 {
		return true, data.ProviderNotApplicableError{}
	}

	var prefix, quoteChar string
	if len(indexExpr.Subscripts) == 1 {
		ss, ok := indexExpr.Subscripts[0].(*pythonast.IndexSubscript)
		if !ok {
			// We can't do any autocomplete on a subscript that is not an index (ie the value accessed is not a dict)
			return true, data.ProviderNotApplicableError{}
		}
		switch sc := ss.Value.(type) {
		case *pythonast.NameExpr:
			prefix = sc.Ident.Literal
		case *pythonast.NumberExpr:
			prefix = sc.Number.Literal
		case *pythonast.StringExpr:
			quoteChar = in.TextAt(data.NewSelection(sc.Begin(), sc.Begin()+1))
			prefix = sc.Literal()
		}
	}
	// quoteChar must be empty or one of `'`, `"`
	switch quoteChar {
	case `'`, `"`, ``:
	default:
		quoteChar = ""
	}

	end := in.Selection.End
	if indexExpr.RightBrack != nil {
		end = int(indexExpr.RightBrack.End)
	}

	replace := data.Selection{Begin: int(indexExpr.LeftBrack.Begin), End: end}
	for _, kc := range dp.getValidKeys(dict, prefix, 1) {
		compl := data.Completion{
			Replace: replace,
			Snippet: data.NewSnippet("[" + kc.quoted(quoteChar) + "]"),
		}
		dp.emit(ctx, out, in.SelectedBuffer, kc, false, compl)
	}
	return true, nil
}

// MarshalJSON implements Provider
func (dp DictKeys) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: dp.Name(),
	})
}

func (dp DictKeys) provideGetPopCompletions(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) (bool, error) {
	var callExpr *pythonast.CallExpr
	for _, n := range in.UnderSelection() {
		if n, _ := n.(*pythonast.CallExpr); n != nil {
			// if in.Selection.Begin == callExpr.LeftParen.Begin then the selection
			// includes the left paren which we do not want
			if in.Selection.Begin >= int(n.LeftParen.End) {
				// the end of selection is "exclusive" so in.Selection.End == callExpr.Righparen.Begin is ok
				// since the right paren is not included in the selection
				if n.RightParen != nil && in.Selection.End <= int(n.RightParen.Begin) {
					callExpr = n
					break
				}
			}
		}
	}
	if callExpr == nil {
		return false, data.ProviderNotApplicableError{}
	}

	attrExpr, ok := callExpr.Func.(*pythonast.AttributeExpr)
	if !ok {
		return true, data.ProviderNotApplicableError{}
	}

	attr := attrExpr.Attribute.Literal
	if attr != "get" && attr != "pop" && attr != "setdefault" {
		return true, data.ProviderNotApplicableError{}
	}

	val := in.ResolvedAST().References[attrExpr.Value]
	dict, ok := val.(pythontype.DictLike)
	if !ok {
		return true, data.ProviderNotApplicableError{}
	}

	if len(callExpr.Args) > 1 {
		// We only support completion for the first argument
		return true, data.ProviderNotApplicableError{}
	}

	var prefix, quoteChar string
	if len(callExpr.Args) > 0 {
		firstArg := callExpr.Args[0]
		switch sc := firstArg.Value.(type) {
		case *pythonast.NameExpr:
			prefix = sc.Ident.Literal
		case *pythonast.StringExpr:
			prefix = sc.Literal()
			quoteChar = in.TextAt(data.NewSelection(sc.Begin(), sc.Begin()+1))
		}
		if name, ok := callExpr.Args[0].Value.(*pythonast.NameExpr); ok {
			prefix = name.Ident.Literal
		}
	}
	// quoteChar must be empty or one of `'`, `"`
	switch quoteChar {
	case `'`, `"`, ``:
	default:
		quoteChar = ""
	}

	replace := data.NewSelection(callExpr.LeftParen.End, callExpr.RightParen.Begin)
	for _, kc := range dp.getValidKeys(dict, prefix, 1) {
		compl := data.Completion{
			Replace: replace,
			Snippet: data.NewSnippet(kc.quoted(quoteChar)),
		}
		dp.emit(ctx, out, in.SelectedBuffer, kc, false, compl)
	}
	return true, nil
}

func (dp DictKeys) provideAttributeCompletion(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) (bool, error) {
	ctx.CheckAbort()
	var attrExpr *pythonast.AttributeExpr
	underPos := in.UnderSelection()
	for i := len(underPos) - 1; i >= 0; i-- {
		n := underPos[i]
		if attr, _ := n.(*pythonast.AttributeExpr); attr != nil {
			if in.Selection.Begin >= int(attr.Attribute.Begin) && in.Selection.End <= int(attr.Attribute.End) {
				attrExpr = attr
				break
			}
		}
	}

	if attrExpr == nil {
		return false, data.ProviderNotApplicableError{}
	}

	val := in.ResolvedAST().References[attrExpr.Value]
	dict, ok := val.(pythontype.DictLike)
	if !ok {
		return true, data.ProviderNotApplicableError{}
	}

	_, isADataFrame := dict.(pythontype.DataFrameInstance)

	var prefix string
	if attrExpr.Attribute != nil {
		prefix = attrExpr.Attribute.Literal
	}

	if prefix == "" && !isADataFrame {
		// We don't want attribute completion for empty prefix
		return true, data.ProviderNotApplicableError{}
	}

	// Setting a score of 0 allows for these completions to be put at the end of the list (after the normal attributes)
	// That also means the speculation after these completion will have the lowest priority
	// It's ok for now as we don't speculate after them but might need to be changed if this behavior change in the future
	validKeys := dp.getValidKeys(dict, prefix, 0)
	if len(validKeys) == 0 && len(prefix) > 0 {
		var isStr bool
		if _, err := strconv.Atoi(prefix); err != nil {
			isStr = true
		}
		validKeys = append(validKeys, keyCompletion{
			literal: prefix,
			isStr:   isStr,
			value:   nil,
			score:   0,
		})
	}

	replace := data.Selection{Begin: int(attrExpr.Dot.Begin), End: in.Selection.End}
	for _, kc := range validKeys {
		compl := data.Completion{Replace: replace}
		attrToSubs := !isADataFrame || !pythonscanner.IsValidIdent(kc.literal)
		if attrToSubs {
			compl.Snippet = data.NewSnippet("[" + kc.quoted("") + "]")
		} else {
			compl.Snippet = data.NewSnippet("." + kc.literal)
		}
		dp.emit(ctx, out, in.SelectedBuffer, kc, attrToSubs, compl)
	}
	return true, nil
}

func (dp DictKeys) emit(ctx kitectx.Context, out OutputFunc, sb data.SelectedBuffer, kc keyCompletion, attrToSubs bool, compl data.Completion) {
	out(ctx, sb, MetaCompletion{
		Completion: compl,
		Provider:   dp.Name(),
		Source:     response.DictCompletionSource,
		Score:      kc.score,
		RenderMeta: RenderMeta{Referent: kc.value},
		DictMeta:   &DictMeta{AttributeToSubscript: attrToSubs},
	})
}

func (dp DictKeys) getValidKeys(dict pythontype.DictLike, prefix string, score float64) []keyCompletion {
	var validKeys []keyCompletion
	for key, val := range dict.GetTrackedKeys() {
		switch key := key.(type) {
		case pythontype.StrConstant:
			literal := string(key)
			if !strings.HasPrefix(literal, prefix) {
				continue
			}

			validKeys = append(validKeys, keyCompletion{
				literal: literal,
				isStr:   true,
				value:   val,
				score:   score,
			})
		case pythontype.IntConstant:
			keyStr := fmt.Sprint(key)
			if !strings.HasPrefix(keyStr, prefix) {
				continue
			}

			validKeys = append(validKeys, keyCompletion{
				literal: keyStr,
				isStr:   false,
				value:   val,
				score:   score,
			})
		}
	}
	sort.Slice(validKeys, func(i, j int) bool {
		switch {
		case validKeys[i].isStr && !validKeys[j].isStr:
			return true
		case !validKeys[i].isStr && validKeys[j].isStr:
			return false
		case !validKeys[i].isStr && !validKeys[j].isStr:
			// TODO(naman) it doesn't really make sense to sort ints like this
			fallthrough
		case validKeys[i].isStr && validKeys[j].isStr:
			return validKeys[i].literal < validKeys[j].literal
		default:
			panic("unreachable")
		}
	})
	return validKeys
}
