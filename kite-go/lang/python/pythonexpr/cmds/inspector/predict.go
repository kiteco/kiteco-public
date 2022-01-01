package main

import (
	"bytes"
	"fmt"
	"go/token"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncall"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

var (
	scanOpts = pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}

	parseOpts = pythonparser.Options{
		Approximate: true,
		ErrorMode:   pythonparser.Recover,
	}
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func analyze(rm pythonresource.Manager, src string) ([]pythonscanner.Word, *pythonanalyzer.ResolvedAST, error) {
	bSrc := []byte(src)
	words, err := pythonscanner.Lex(bSrc, scanOpts)

	if err != nil {
		return nil, nil, errors.Errorf("unable to lex file: %v", err)
	}

	ast, err := pythonparser.ParseWords(kitectx.Background(), bSrc, words, parseOpts)
	if ast == nil {
		return nil, nil, errors.Errorf("unable to parse ast: %v", err)
	}

	rast, err := pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{
		Path: "/src.py",
	}).Resolve(ast)
	if err != nil {
		return nil, nil, errors.Errorf("analyze error: %v", err)
	}
	return words, rast, nil
}

type predicted struct {
	Trace          string
	PredictionTree string
	Meta           string
	Graph          *searchGraph
}

func predict(rm pythonresource.Manager, m pythonexpr.Model, hash string, sc srcCursor, metaonly bool) (predicted, error) {
	words, rast, err := analyze(rm, sc.Src)
	if err != nil {
		return predicted{}, err
	}

	// Select the shard before doing anything else
	if sharded, ok := m.(*pythonexpr.ShardedModel); ok {
		sharded.SelectShard(rast.Root)
	}

	var expr pythonast.Expr
	pythonast.Inspect(rast.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !pythonast.IsNil(expr) {
			return false
		}

		switch n := n.(type) {
		case *pythonast.CallExpr:
			begin := n.LeftParen.Begin
			if len(n.Args) > 0 && len(n.Commas) > 0 {
				begin = n.Commas[len(n.Commas)-1].Begin
			}
			end := n.End()
			if n.RightParen != nil {
				end = n.RightParen.Begin
			}

			if sc.Cursor > begin && sc.Cursor <= end {
				expr = n
			}
		case *pythonast.AttributeExpr:
			if sc.Cursor >= n.Attribute.Begin && sc.Cursor <= n.Attribute.End {
				expr = n
			}
		case *pythonast.NameExpr:
			if sc.Cursor >= n.Begin() && sc.Cursor <= n.End() {
				expr = n
			}

		}
		return true
	})

	var meta string
	switch e := expr.(type) {
	case *pythonast.CallExpr:
		meta = metaForCall(rm, m, rast, e)
	case *pythonast.NameExpr:
		meta = fmt.Sprintf("old lit: %s before wiping", e.Ident.Literal)
		munged, err := prepareForNamePrediction(rm, rast, sc.Src, e)
		if err != nil {
			return predicted{}, errors.Errorf("error preparing for name prediction: %v", err)
		}
		sc.Src = munged.Src
		words = munged.Words
		rast = munged.RAST
		expr = munged.Expr
	case *pythonast.AttributeExpr:
		meta = metaForAttr(rm, m, rast, e)
		munged, err := prepareForAttrPrediction(rm, rast, sc.Src, e)
		if err != nil {
			return predicted{}, errors.Errorf("error preparing for name prediction: %v", err)
		}
		sc.Src = munged.Src
		words = munged.Words
		rast = munged.RAST
		expr = munged.Expr
	default:
		return predicted{}, errors.Errorf("unable to find expr under cursor")
	}

	if pythonast.IsNil(expr) {
		return predicted{}, errors.Errorf("unable to find expr under cursor")
	}

	if metaonly {
		return predicted{
			Meta: meta,
		}, nil
	}

	var trace bytes.Buffer
	var saver saver

	in := pythonexpr.Input{
		Src:                 []byte(sc.Src),
		RM:                  rm,
		RAST:                rast,
		Words:               words,
		Expr:                expr,
		Tracer:              &trace,
		Saver:               &saver,
		MungeBufferForAttrs: true,
		MaxPatterns:         3,
	}

	res, err := m.Predict(kitectx.Background(), in)

	var predictions string
	if err != nil {
		predictions = err.Error()
	} else {
		var b bytes.Buffer
		pythongraph.Print(res.OldPredictorResult, &b)
		predictions = b.String()
	}

	var sg *searchGraph
	if len(saver.Saved) == 1 {
		sg, err = newSearchGraph(hash, saver.Saved[0])
		if err != nil {
			return predicted{}, errors.Errorf("error rendering search graph: %v", err)
		}
	} else {
		log.Printf("got %d saved samples\n", len(saver.Saved))
	}

	return predicted{
		PredictionTree: predictions,
		Trace:          trace.String(),
		Meta:           meta,
		Graph:          sg,
	}, nil
}

type srcCursor struct {
	Src    string
	Cursor token.Pos
}

func newSrcCursor(s string) (srcCursor, error) {
	parts := strings.Split(s, "$")
	switch len(parts) {
	case 1, 2:
		return srcCursor{
			Src:    strings.Join(parts, ""),
			Cursor: token.Pos(len(parts[0])),
		}, nil
	default:
		return srcCursor{}, errors.Errorf("input src may contain 0 or 1 cursor ($), got: %d", len(parts)-1)
	}
}

type munged struct {
	Src   string
	Words []pythonscanner.Word
	RAST  *pythonanalyzer.ResolvedAST
	Expr  pythonast.Expr
}

const guessMe = "GUESS_ME_1235987"

func prepareForNamePrediction(rm pythonresource.Manager, rast *pythonanalyzer.ResolvedAST, src string, name *pythonast.NameExpr) (munged, error) {
	trimEnd := trimEndLineOrStmt(name.End(), name, src, rast)
	newSrc := strings.Join([]string{
		src[:name.Begin()],
		guessMe,
		src[trimEnd:],
	}, "")

	words, rast, err := analyze(rm, newSrc)
	if err != nil {
		return munged{}, errors.Errorf("error re analyzing: %v", err)
	}

	var found *pythonast.NameExpr
	pythonast.Inspect(rast.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !pythonast.IsNil(found) {
			return false
		}

		if name1, ok := n.(*pythonast.NameExpr); ok {
			if name1.Ident.Literal == guessMe {
				found = name1
			}
		}

		return true
	})

	if found == nil {
		return munged{}, errors.Errorf("unable to refind name")
	}

	for i, w := range words {
		if w.Literal == guessMe {
			w.Literal = ""
			words[i] = w
			break
		}
	}

	found.Ident.Literal = ""

	return munged{
		Src:   newSrc,
		Words: words,
		RAST:  rast,
		Expr:  found,
	}, nil
}

func prepareForAttrPrediction(rm pythonresource.Manager, rast *pythonanalyzer.ResolvedAST, src string, attr *pythonast.AttributeExpr) (munged, error) {
	trimEnd := trimEndLineOrStmt(attr.Dot.End, attr, src, rast)
	newSrc := strings.Join([]string{
		src[:attr.Dot.End],
		guessMe,
		src[trimEnd:],
	}, "")

	log.Println(newSrc)

	words, rast, err := analyze(rm, newSrc)
	if err != nil {
		return munged{}, errors.Errorf("error re analyzing: %v", err)
	}

	var found *pythonast.AttributeExpr
	pythonast.Inspect(rast.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !pythonast.IsNil(found) {
			return false
		}

		if attr1, ok := n.(*pythonast.AttributeExpr); ok {
			if attr1.Attribute.Literal == guessMe {
				found = attr1
			}
		}

		return true
	})

	if found == nil {
		return munged{}, errors.Errorf("unable to refind attr")
	}

	for i, w := range words {
		if w.Literal == guessMe {
			w.Literal = ""
			words[i] = w
			break
		}
	}

	found.Attribute.Literal = ""

	return munged{
		Src:   newSrc,
		Words: words,
		RAST:  rast,
		Expr:  found,
	}, nil
}

func trimEndLineOrStmt(pos token.Pos, n pythonast.Expr, src string, rast *pythonanalyzer.ResolvedAST) int {
	lm := linenumber.NewMap([]byte(src))
	_, lePos := lm.LineBounds(lm.Line(int(pos)))
	stmt := rast.ParentStmts[n]
	if lePos < int(stmt.End()) {
		return int(stmt.End())
	}
	return lePos
}

func metaForAttr(rm pythonresource.Manager, model pythonexpr.Model, rast *pythonanalyzer.ResolvedAST, attr *pythonast.AttributeExpr) string {
	var sym pythonresource.Symbol
	var children []pythonresource.Symbol
	var rows []int32
	for _, s := range python.GetExternalSymbols(kitectx.Background(), rm, rast.References[attr.Value]) {
		var err error
		rows, children, err = model.AttrCandidates(rm, s)
		if err == nil {
			sym = s
			break
		}
	}

	if sym.Nil() {
		return "no attribute info found"
	}

	var cs []string
	for i, child := range children {
		row := rows[i]
		cs = append(cs, fmt.Sprintf("  %s (row %d)", child.Path().Last(), row))
	}

	return fmt.Sprintf("Sym: %s, Candidates:\n%s\n",
		sym.PathString(),
		strings.Join(cs, "\n"),
	)
}

func metaForCall(rm pythonresource.Manager, model pythonexpr.Model, rast *pythonanalyzer.ResolvedAST, call *pythonast.CallExpr) string {
	var sym pythonresource.Symbol
	for _, s := range python.GetExternalSymbols(kitectx.Background(), rm, rast.References[call.Func]) {
		_, err := model.FuncInfo(rm, s)
		if err == nil {
			sym = s
			break
		}
	}

	if sym.Nil() {
		return "no function info found"
	}

	fs := pythoncall.SymbolForFunc(rm, sym)

	patterns := traindata.NewCallPatterns(rm, fs)
	if patterns == nil {
		return "no patterns found"
	}

	ps := []string{"Patterns:"}
	for _, sig := range patterns.Signatures {
		var kws []string
		if py := sig.LanguageDetails.Python; py != nil {
			for _, pe := range py.Kwargs {
				kws = append(kws, pe.Name)
			}
		}

		ps = append(ps, fmt.Sprintf("Positional: %d, Keyword: %s", len(sig.Args), strings.Join(kws, ",")))
	}

	ps = append(ps, "Args:")
	argStrs := func(arg *traindata.Arg, i int) {
		var types, toks []string
		if i > -1 {
			types, toks = patterns.Feed("", i)
		} else {
			types, toks = patterns.Feed(arg.Name, 0)
		}
		ps = append(ps, fmt.Sprintf("Arg %d '%s':", i, arg.Name))

		var tokStrs []string
		for _, tok := range toks {
			tokStrs = append(tokStrs, fmt.Sprintf("%s (%d)", tok, model.MetaInfo().NameSubtokenIndex.Index(tok)))
		}
		ps = append(ps, fmt.Sprintf("  Subtoks: %s", strings.Join(tokStrs, " , ")))

		var typeStrs []string
		for _, t := range types {
			typeStrs = append(typeStrs, fmt.Sprintf("%s (%d)", t, model.MetaInfo().TypeSubtokenIndex.Index(t)))
		}

		ps = append(ps, fmt.Sprintf("  Types: %s", strings.Join(typeStrs, " , ")))

		typeStrs = nil
		for _, t := range arg.Types {
			typeStrs = append(typeStrs, fmt.Sprintf("%s (%d)", t.Path, t.Count))
		}

		ps = append(ps, fmt.Sprintf("  Counted types: %s", strings.Join(typeStrs, " , ")))

	}
	for i, arg := range patterns.Positional {
		argStrs(arg, i)
	}

	for _, arg := range patterns.ArgsByName {
		argStrs(arg, -1)
	}

	as := rm.ArgSpec(fs)

	var spec string
	if as == nil {
		spec = "ArgSpec: nil"
	} else {
		spec = fmt.Sprintf("ArgSpec has %d args", len(as.Args))
	}

	sig := fmt.Sprintf("Sigs: %d", len(patterns.Signatures))

	fstr := `Sym %s -> %s
%s
%s
%s
`

	return fmt.Sprintf(fstr,
		sym.PathString(),
		fs.PathString(),
		spec,
		sig,
		strings.Join(ps, "\n"),
	)
}
