package pythonparser

import (
	"bytes"
	"go/token"
	"regexp"

	"github.com/golang-collections/go-datastructures/augmentedtree"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/calls"
	pyscan "github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// ApproximateBadRegions finds BadStmt and BadExpr nodes and uses the regex parser to create "approximate" AST nodes.
// These nodes will be inserted into the BadStmt.Approximation and BadExpr.Approximation fields.
func ApproximateBadRegions(ctx kitectx.Context, mod *pythonast.Module, src []byte, words []pyscan.Word) {
	ctx.CheckAbort()

	// remove comments to avoid false positives with regexes
	src = removeComments(src, words)

	// combine consecutive BadStmt's as needed, we do not need to worry about the
	// children of BadStmt's since they have not been added yet.
	pythonast.Inspect(mod, func(node pythonast.Node) bool {
		ctx.CheckAbort()

		if pythonast.IsNil(node) {
			return false
		}
		switch node := node.(type) {
		case *pythonast.Module:
			node.Body = combineBadStmts(node.Body)
		case *pythonast.ClassDefStmt:
			node.Body = combineBadStmts(node.Body)
		case *pythonast.FunctionDefStmt:
			node.Body = combineBadStmts(node.Body)
		case *pythonast.Branch:
			node.Body = combineBadStmts(node.Body)
		case *pythonast.IfStmt:
			node.Else = combineBadStmts(node.Else)
		case *pythonast.ForStmt:
			node.Body = combineBadStmts(node.Body)
			node.Else = combineBadStmts(node.Else)
		case *pythonast.WhileStmt:
			node.Body = combineBadStmts(node.Body)
			node.Else = combineBadStmts(node.Else)
		case *pythonast.ExceptClause:
			node.Body = combineBadStmts(node.Body)
		case *pythonast.TryStmt:
			node.Body = combineBadStmts(node.Body)
			node.Else = combineBadStmts(node.Else)
			node.Finally = combineBadStmts(node.Finally)
		case *pythonast.WithStmt:
			node.Body = combineBadStmts(node.Body)
		case *pythonast.BadStmt:
			// track which intervals have been used by an ast node in the bad stmt region
			intervals := augmentedtree.New(1)

			regionWords := wordsInRegion(words, node.Begin(), node.End())
			region := src[node.Begin():node.End()]
			offset := int(node.Begin())

			addStatements(intervals, extractImportFrom(region, offset), node)

			addStatements(intervals, extractImportName(region, offset), node)

			addStatements(intervals,
				extractReturnAssignments(ctx, regionWords, region, offset), node)

			addStatements(intervals,
				extractNamedDefinitions(ctx, regionWords, region, offset), node)

			addStatements(intervals,
				extractKeywordStatements(ctx, regionWords, region, offset), node)

			addStatements(intervals,
				toStmts(extractFunctionCalls(ctx, regionWords, region, offset)), node)

			addStatements(intervals,
				toStmts(extractDotExprs(ctx, regionWords, region, offset)), node)

			// do not recurse into the approximate nodes we just added
			return false

		case *pythonast.BadExpr:
			// track which intervals have been used by an ast node in the bad expr region
			intervals := augmentedtree.New(1)

			regionWords := wordsInRegion(words, node.Begin(), node.End())
			region := src[node.Begin():node.End()]
			offset := int(node.Begin())

			addExpressions(intervals,
				extractFunctionCalls(ctx, regionWords, region, offset), node)

			addExpressions(intervals,
				extractDotExprs(ctx, regionWords, region, offset), node)

			// do not recurse into the approximate nodes we just added
			return false
		}
		return true
	})
}

// combineBadStmts combines consecutive BadStmt's as needed, we do not need to worry about the
// children of BadStmt's since they have not been added yet.
func combineBadStmts(stmts []pythonast.Stmt) []pythonast.Stmt {
	var newStmts []pythonast.Stmt
	var current *pythonast.BadStmt
	for _, stmt := range stmts {
		if bad, isBad := stmt.(*pythonast.BadStmt); isBad {
			if current == nil {
				current = bad
				continue
			}
			current.To = bad.End()
			continue
		}

		if current != nil {
			newStmts = append(newStmts, current)
			current = nil
		}
		newStmts = append(newStmts, stmt)
	}
	if current != nil {
		newStmts = append(newStmts, current)
	}
	return newStmts
}

func addStatements(intervals augmentedtree.Tree, stmts []pythonast.Stmt, node *pythonast.BadStmt) {
	for _, stmt := range stmts {
		interval := &interval{
			begin: int64(stmt.Begin()),
			end:   int64(stmt.End()),
			id:    intervals.Len(),
		}
		if len(intervals.Query(interval)) == 0 {
			intervals.Add(interval)
			node.Approximation = append(node.Approximation, stmt)
		}
	}
}

func addExpressions(intervals augmentedtree.Tree, exprs []pythonast.Expr, node *pythonast.BadExpr) {
	for _, expr := range exprs {
		interval := &interval{
			begin: int64(expr.Begin()),
			end:   int64(expr.End()),
			id:    intervals.Len(),
		}
		if len(intervals.Query(interval)) == 0 {
			intervals.Add(interval)
			node.Approximation = append(node.Approximation, expr)
		}
	}
}

type interval struct {
	begin int64
	end   int64
	id    uint64
}

func (i *interval) LowAtDimension(uint64) int64 {
	return i.begin
}

func (i *interval) HighAtDimension(uint64) int64 {
	return i.end
}

func (i *interval) OverlapsAtDimension(ii augmentedtree.Interval, d uint64) bool {
	if ii.LowAtDimension(0) <= i.begin {
		if ii.HighAtDimension(0) >= i.begin {
			return true
		}
		return false
	}

	// must have ii.begin > i.begin
	if ii.LowAtDimension(0) < i.end {
		return true
	}

	return false
}

func (i *interval) ID() uint64 {
	return i.id
}

var (
	// use [\t ] instead of \s to capture non-newline whitespace

	// imports
	// need (^|\n) capture group for parsing ImportNameStmts to avoid triggering on the "import" keyword in ImportFromStmts
	pyImportRegexp     = regexp.MustCompile(`(?:^|\n)[\t ]*(import)\s*(?P<name>[a-zA-Z0-9._, ]+)*`)
	pyImportFromRegexp = regexp.MustCompile(`(?:^|\n)[\t ]*(from)(?:[\t ]+(\.*)(?P<from>[a-zA-Z_][a-zA-Z0-9._]*)?([\t ]+(import)[\t ]+(?P<import>[a-zA-Z0-9\._, ]+|\*)?)?)?`)
	pyAsRegexp         = regexp.MustCompile(`(?P<import>[a-zA-Z_][a-zA-Z0-9._]*)\s+as\s+(?P<as>[a-zA-Z_][a-zA-Z0-9._]*)`)

	pyDotExprRegexp      = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9._]*`)
	pyFunctionCallRegexp = regexp.MustCompile(`(?P<funcName>[a-zA-Z0-9._]+)(?P<lparen>[\(]{1})`)
	pyReturnAssignRegexp = regexp.MustCompile(`(?P<lhs>[a-zA-Z_][a-zA-Z0-9._]*)\s+=\s+(?P<rhs>[a-zA-Z_][a-zA-Z0-9._]*)\({1}`)

	pyDefNameStmtRegexp = regexp.MustCompile(`(?:^|\n)[\t ]*(class|def)\s+(?P<name>[a-zA-Z_][a-zA-Z0-9_]*)`)
	pyKeywordStmtRegexp = regexp.MustCompile(`(?:^|\n)[\t ]*(if|with|while|for)\s+`)
)

// parseDotExpr parses one or more dot-separated identifiers (e.g. "foo.bar.baz")
// will include white space if included in region
func parseDotExpr(ctx kitectx.Context, region []byte, offset int) pythonast.Expr {
	ctx.CheckAbort()

	idx := bytes.LastIndex(region, []byte("."))

	if idx == -1 {
		if len(region) == 0 {
			return nil
		}
		return &pythonast.NameExpr{
			Ident: &pyscan.Word{
				Token:   pyscan.Ident,
				Literal: string(region),
				Begin:   token.Pos(offset),
				End:     token.Pos(offset + len(region)),
			},
		}
	}

	value := region[:idx]
	var attrib string
	if idx+1 < len(region) {
		attrib = string(region[idx+1:])
	}

	dot := &pyscan.Word{
		Token: pyscan.Period,
		Begin: token.Pos(offset + idx),
		End:   token.Pos(offset + idx + 1),
	}

	base := parseDotExpr(ctx, value, offset)
	if base == nil {
		return nil
	}

	// TOOD(juan): what to do with empty attrib? BadToken?
	return &pythonast.AttributeExpr{
		Value: base,
		Dot:   dot,
		Attribute: &pyscan.Word{
			Token:   pyscan.Ident,
			Literal: attrib,
			Begin:   token.Pos(offset + idx + 1),
			End:     token.Pos(offset + len(region)),
		},
	}
}

// extractDotExprs parses all dotted name (e.g. "foo.bar.baz") in the given region
func extractDotExprs(ctx kitectx.Context, regionWords []pyscan.Word, region []byte, offset int) []pythonast.Expr {
	ctx.CheckAbort()

	var exprs []pythonast.Expr
	for _, m := range pyDotExprRegexp.FindAllIndex(region, -1) {
		begin, end := m[0], m[1]

		// check if in a string
		if inString(regionWords, offset+begin) {
			continue
		}

		expr := parseDotExpr(ctx, region[begin:end], offset+begin)

		// do not include name expressions that
		// just contain the word import, or from, or as.
		if expr, ok := expr.(*pythonast.NameExpr); ok {
			if _, found := pyscan.Keywords[expr.Ident.Literal]; found {
				continue
			}
		}

		exprs = append(exprs, expr)
	}
	return exprs
}

type byPos []token.Pos

func (bp byPos) Len() int           { return len(bp) }
func (bp byPos) Swap(i, j int)      { bp[i], bp[j] = bp[j], bp[i] }
func (bp byPos) Less(i, j int) bool { return bp[i] < bp[j] }

// extractFunctionCalls parses all function calls (e.g. "foo.bar(baz)") in the given region
func extractFunctionCalls(ctx kitectx.Context, regionWords []pyscan.Word, region []byte, offset int) []pythonast.Expr {
	ctx.CheckAbort()

	var callExprs []pythonast.Expr
	for _, m := range pyFunctionCallRegexp.FindAllSubmatchIndex(region, -1) {
		// _, _, namebegin, nameend, lbegin, lend := m[0], m[1], m[2], m[3], m[4], m[5]
		_, _, namebegin, _, _, _ := m[0], m[1], m[2], m[3], m[4], m[5]

		if inString(regionWords, offset+namebegin) {
			continue
		}

		call, _ := calls.Parse(region[namebegin:], calls.MaxLines(1))
		if call == nil {
			continue
		}

		pythonast.Inspect(call, func(n pythonast.Node) bool {
			ctx.CheckAbort()

			if n != nil {
				n.AddOffset(offset + namebegin)
			}
			return true
		})
		callExprs = append(callExprs, call)
	}
	return callExprs
}

// extractNamedDefinitions parses all class and function definitions (e.g. "class X: ...") in the given region
func extractNamedDefinitions(ctx kitectx.Context, regionWords []pyscan.Word, region []byte, offset int) []pythonast.Stmt {
	ctx.CheckAbort()

	var defs []pythonast.Stmt
	for _, m := range pyDefNameStmtRegexp.FindAllSubmatchIndex(region, -1) {
		begin := m[2]

		if inString(regionWords, offset+begin) {
			continue
		}

		// For now, parse only the class/function definition line (no body)
		def, _ := calls.ParseStmt(region[begin:], calls.MaxLines(1))
		if def == nil {
			continue
		}

		pythonast.Inspect(def, func(n pythonast.Node) bool {
			ctx.CheckAbort()

			if n != nil {
				n.AddOffset(offset + begin)
			}
			return true
		})
		defs = append(defs, def)
	}
	return defs
}

// extractKeywordStatements parses all if/with/while/for statements (e.g. "with x as y: ...") in the given region
func extractKeywordStatements(ctx kitectx.Context, regionWords []pyscan.Word, region []byte, offset int) []pythonast.Stmt {
	ctx.CheckAbort()

	var stmts []pythonast.Stmt
	for _, m := range pyKeywordStmtRegexp.FindAllSubmatchIndex(region, -1) {
		begin := m[2]

		if inString(regionWords, offset+begin) {
			continue
		}

		// For now, parse only the top statement line (no body/else/etc.)
		stmt, _ := calls.ParseStmt(region[begin:], calls.MaxLines(1))
		if stmt == nil {
			continue
		}

		pythonast.Inspect(stmt, func(n pythonast.Node) bool {
			ctx.CheckAbort()

			if n != nil {
				n.AddOffset(offset + begin)
			}
			return true
		})
		stmts = append(stmts, stmt)
	}
	return stmts
}

// extractReturnAssignments parses all return assignments (e.g. "x = foo.bar(baz)") in the given region
func extractReturnAssignments(ctx kitectx.Context, regionWords []pyscan.Word, region []byte, offset int) []pythonast.Stmt {
	ctx.CheckAbort()

	var assigns []pythonast.Stmt
	for _, m := range pyReturnAssignRegexp.FindAllSubmatchIndex(region, -1) {
		_, _, lhsbegin, _, _, _ := m[0], m[1], m[2], m[3], m[4], m[5]

		if inString(regionWords, offset+lhsbegin) {
			continue
		}

		stmt, _ := calls.ParseStmt(region[lhsbegin:], calls.MaxLines(1))
		if stmt == nil {
			continue
		}
		switch stmt.(type) {
		case *pythonast.AssignStmt:
		case *pythonast.AugAssignStmt:
		default:
			// if ParseStmt parses something else, ignore - we only care for
			// assignments here.
			continue
		}

		pythonast.Inspect(stmt, func(n pythonast.Node) bool {
			ctx.CheckAbort()

			if n != nil {
				n.AddOffset(offset + lhsbegin)
			}
			return true
		})
		assigns = append(assigns, stmt)
	}

	return assigns
}

// will include spaces if present
func parseDottedExpr(region []byte, offset int) *pythonast.DottedExpr {
	var dotted pythonast.DottedExpr
	parts := bytes.Split(region, []byte("."))
	for i, part := range parts {
		name := &pythonast.NameExpr{
			Ident: &pyscan.Word{
				Token:   pyscan.Ident,
				Literal: string(part),
				Begin:   token.Pos(offset),
				End:     token.Pos(offset + len(part)),
			},
		}
		dotted.Names = append(dotted.Names, name)

		if i < len(parts)-1 {
			dotted.Dots = append(dotted.Dots, &pyscan.Word{
				Token: pyscan.Period,
				Begin: name.End(),
				End:   name.End() + 1,
			})
		}

		// +1 for dot
		offset += len(part) + 1
	}
	return &dotted
}

// NOTE: will include spaces if present
func parseNameExpr(region []byte, offset int) *pythonast.NameExpr {
	return &pythonast.NameExpr{
		Ident: &pyscan.Word{
			Token:   pyscan.Ident,
			Literal: string(region),
			Begin:   token.Pos(offset),
			End:     token.Pos(offset + len(region)),
		},
	}
}

func extractImportAsName(region []byte, offset int) ([]*pythonast.ImportAsName, []*pyscan.Word) {
	var names []*pythonast.ImportAsName
	var commas []*pyscan.Word
	var commaPos int
	origOffset := offset
	for _, part := range bytes.Split(region, []byte(",")) {
		if m := pyAsRegexp.FindSubmatchIndex(part); len(m) > 0 {
			namebegin, nameend, asbegin, asend := m[2], m[3], m[4], m[5]

			names = append(names, &pythonast.ImportAsName{
				External: parseNameExpr(part[namebegin:nameend], offset+namebegin),
				Internal: parseNameExpr(part[asbegin:asend], offset+asbegin),
			})
		} else if m := pyDotExprRegexp.FindSubmatchIndex(part); len(m) > 0 {
			begin, end := m[0], m[1]

			names = append(names, &pythonast.ImportAsName{
				External: parseNameExpr(part[begin:end], offset+begin),
			})
		}

		commaPos += len(part)
		if commaPos < len(region) && region[commaPos] == ',' {
			commas = append(commas, &pyscan.Word{
				Begin: token.Pos(origOffset + commaPos),
				End:   token.Pos(origOffset + commaPos + 1),
				Token: pyscan.Comma,
			})
		}
		// +1 for comma
		commaPos++

		// +1 for comma
		offset += len(part) + 1
	}
	return names, commas
}

// NOTE: does not support:
//     - left or right parens
//     - FromDots
//     - Wildcard
// \s*(from)[\t ]+(\.*)(?P<from>[a-zA-Z_][a-zA-Z0-9._]*)?([\t ]+(import)[\t ]+(?P<import>[a-zA-Z0-9\._, ]+|\*)?)?
func extractImportFrom(region []byte, offset int) []pythonast.Stmt {
	var imports []pythonast.Stmt
	for _, m := range pyImportFromRegexp.FindAllSubmatchIndex(region, -1) {
		begin, dotsBegin, dotsEnd, frombegin, fromend, importbegin, importend, namesbegin, namesend := m[2], m[4], m[5], m[6], m[7], m[10], m[11], m[12], m[13]

		imp := &pythonast.ImportFromStmt{
			From: &pyscan.Word{
				Token: pyscan.From,
				Begin: token.Pos(offset + begin),
				End:   token.Pos(offset + begin + 4),
			},
		}

		// add dots; if none exist, then dotsBegin == dotsEnd == -1, so the loop body won't run
		for i := dotsBegin; i < dotsEnd; i++ {
			imp.Dots = append(imp.Dots, &pyscan.Word{
				Token: pyscan.Period,
				Begin: token.Pos(offset + i),
				End:   token.Pos(offset + i + 1),
			})
		}

		// check if user started typing package
		// if frombegin > -1 then fromend > -1 also
		if frombegin > -1 {
			imp.Package = parseDottedExpr(region[frombegin:fromend], offset+frombegin)
		}

		// check if user started typing "import"
		// if importbegin > -1 then importend > -1
		if importbegin > -1 {
			imp.Import = &pyscan.Word{
				Token: pyscan.Import,
				Begin: token.Pos(offset + importbegin),
				End:   token.Pos(offset + importend),
			}
		}

		// check if user has started typing a clause
		// if namesbegin > -1 then namesend > -1
		if namesbegin > -1 {
			imp.Names, imp.Commas = extractImportAsName(region[namesbegin:namesend], offset+namesbegin)
		}

		imports = append(imports, imp)
	}
	return imports
}

func parseDottedAsName(region []byte, offset int) *pythonast.DottedAsName {
	if m := pyAsRegexp.FindSubmatchIndex(region); len(m) > 0 {
		namebegin, nameend, asbegin, asend := m[2], m[3], m[4], m[5]
		return &pythonast.DottedAsName{
			External: parseDottedExpr(region[namebegin:nameend], namebegin+offset),
			Internal: parseNameExpr(region[asbegin:asend], asbegin+offset),
		}
	} else if m := pyDotExprRegexp.FindSubmatchIndex(region); len(m) > 0 {
		begin, end := m[0], m[1]
		return &pythonast.DottedAsName{
			External: parseDottedExpr(region[begin:end], begin+offset),
		}
	}
	return nil
}

// supports imports of the form: import foo,bar and import foo as bar
// `(^|\n)[\s]*(import)\s*(?P<name>[a-zA-Z0-9._, ]+)*`
func extractImportName(region []byte, offset int) []pythonast.Stmt {
	var imports []pythonast.Stmt
	origOffset := offset
	for _, m := range pyImportRegexp.FindAllSubmatchIndex(region, -1) {
		importbegin, importend, namebegin, nameend := m[2], m[3], m[4], m[5]

		imp := &pythonast.ImportNameStmt{
			Import: &pyscan.Word{
				Begin: token.Pos(importbegin + offset),
				End:   token.Pos(importend + offset),
				Token: pyscan.Import,
			},
		}

		// namebegin and nameend only non negative if user has typed more than
		// just the word import with spaces, if namebegin is non negative then
		// nameend must also be.
		if namebegin > -1 {
			commaPos := namebegin
			parts := bytes.Split(region[namebegin:nameend], []byte(","))
			for _, part := range parts {
				name := parseDottedAsName(part, namebegin+offset)
				if name != nil {
					imp.Names = append(imp.Names, name)
				}

				commaPos += len(part)
				if commaPos < len(region) && region[commaPos] == ',' {
					imp.Commas = append(imp.Commas, &pyscan.Word{
						Begin: token.Pos(origOffset + commaPos),
						End:   token.Pos(origOffset + commaPos + 1),
						Token: pyscan.Comma,
					})
				}
				commaPos++ // +1 for comma

				offset += len(part) + 1 // +1 for comma
			}
		}

		imports = append(imports, imp)
	}
	return imports
}

// -- helpers

func inString(words []pyscan.Word, pos int) bool {
	for _, word := range words {
		if token.Pos(pos) >= word.Begin && token.Pos(pos) < word.End && word.Token == pyscan.String {
			return true
		}
	}
	return false
}

// removeComments finds comment tokens and replaces them with
// an equal number of whitespace characters
func removeComments(src []byte, words []pyscan.Word) []byte {
	var rewritten []byte
	for _, word := range words {
		switch word.Token {
		case pyscan.Comment, pyscan.Magic:
			begin, end := int(word.Begin), int(word.End)
			rewritten = append(rewritten, src[len(rewritten):begin]...)
			rewritten = append(rewritten, bytes.Repeat([]byte(" "), end-begin)...)
		}
	}
	rewritten = append(rewritten, src[len(rewritten):]...)
	return rewritten
}

func toStmts(exprs []pythonast.Expr) []pythonast.Stmt {
	var stmts []pythonast.Stmt
	for _, expr := range exprs {
		stmts = append(stmts, &pythonast.ExprStmt{
			Value: expr,
		})
	}
	return stmts
}

func wordsInRegion(words []pyscan.Word, from, to token.Pos) []pyscan.Word {
	start, end := -1, -1
	for i := range words {
		if words[i].Begin >= to {
			end = i
			break
		}
		if start == -1 && words[i].Begin >= from {
			start = i
		}
	}
	if start == -1 {
		return nil
	}
	if end == -1 {
		return words[start:len(words)]
	}
	return words[start:end]
}

// debug method, leaving in since keeps getting re written.
func wordsToStrings(words []pyscan.Word) []string {
	var strs []string
	for _, word := range words {
		strs = append(strs, word.Token.String())
	}
	return strs
}
