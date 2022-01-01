package golang

import (
	"fmt"
	"io"
	"strings"
	"time"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
)

// Config is the set of formatting options to pass to Prettify to format
// an golang AST back to source code.
type Config struct {
	Indent          string
	SpaceAfterComma bool // true=always, false=never
}

var (
	startMappingBeforeBegin = 10
	keepMappingAfterEnd     = 10
)

// Prettify writes the source code represented by the treesitter node n to w,
// using the formatting configuration options in conf. It returns the position
// mapping of each node's start offset to the output's start offset, or an
// error if it fails to write to w.
func Prettify(w io.Writer, conf Config, src []byte, begin, end int, n *sitter.Node) ([]render.OffsetMapping, error) {
	defer status.PrettifyDuration.DeferRecord(time.Now())
	p := &prettyPrinter{
		w:            w,
		conf:         conf,
		src:          src,
		snippetBegin: begin,
		snippetEnd:   end,
	}
	// debugging: treesitter.Print(n, os.Stdout, "  ")
	treesitter.Walk(p, n)
	return p.mappings, p.err
}

type prettyPrinter struct {
	w    io.Writer
	pos  int // current pos in bytes of the writes to w
	conf Config
	src  []byte

	// The positions of the snippets
	snippetBegin int
	snippetEnd   int

	mappings []render.OffsetMapping

	// whitespace is first stored in pendingWrite to e.g. collapse spaces, drop them
	// if a newline replaces it, etc. It gets written when a non-whitespace write (i.e.
	// a token's content) gets printed.
	pendingWrite string

	// keep track of the last printed node, to ensure mandatory spacing is inserted
	// between identifiers/keywords (including number literals).
	lastPrintedNode *sitter.Node

	// current depth of statements, i.e. number of indent spacing required before
	// writing non-whitespace on a line.
	depth int

	// err is set to the first error encountered when writing to w. Once an error
	// is set, the rest of the writes are skipped and that error will be returned.
	err                error
	pendingDepthChange int
}

func (p *prettyPrinter) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil || p.err != nil {
		return nil
	}

	// Look at the first-level nodes. Simply print the ones before snippet,
	// go through the ones that contain the snippet,
	// and ignore the ones after snippet.
	if render.SafeSymbol(render.SafeParent(n)) == symSourceFile {
		if int(n.EndByte()) < p.snippetBegin {
			p.print(n.Content(p.src))
			return nil
		}
		if int(n.StartByte()) > p.snippetEnd {
			return nil
		}
	}

	children := int(n.ChildCount())
	sym := int(n.Symbol())
	nodeChildren := make([]*sitter.Node, children)
	for i := 0; i < children; i++ {
		nodeChildren[i] = n.Child(i)
	}

	// fmt.Printf("%s | sym: %d (type: %s) | parent sym: %d | %d-%d | nchild: %d | %q\n", n, int(n.Symbol()), n.Type(), render.SafeSymbol(render.SafeParent(n)), n.StartByte(), n.EndByte(), n.ChildCount(), n.Content(p.src))
	switch sym {
	case symERROR:
		exactlyAfter := make(map[int]string)
		for i := 1; i < children; i++ {
			if nodeChildren[i-1].EndPoint() != nodeChildren[i].StartPoint() {
				exactlyAfter[i] = string(p.src[nodeChildren[i-1].EndByte():nodeChildren[i].StartByte()])
			}
		}
		// on exit of the spacer, if parentNode is ERROR, it writes
		// whatever part of the source that was not parsed into a child node,
		return &spacerVisitor{p: p, parentNode: n, childCount: children, maxSpaces: -1, exactlyAfter: exactlyAfter}

	case anonSymLf:
		// Handle newline symbols, print them one by one to make sure the depth works as expected
		content := n.Content(p.src)
		for i := range content {
			p.print(content[i : i+1])
		}
		return nil

	case symInterpretedStringLiteral, symRawStringLiteral:
		// Print strings as it is
		p.printNodeSrc(n)
		return nil

	case symAssignmentStatement, symShortVarDeclaration, symSendStatement, symFieldDeclaration, symTypeAlias, symTypeSpec:
		// Simply separate all children with spaces
		return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}

	case symKeyedElement:
		if children > 2 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: map[int]string{2: " "}}
		}

	case symBlock, symImportSpecList, symFieldDeclarationList, symMethodSpecList:
		if children > 2 {
			p.print(" ")
			exactlyAfter := map[int]string{1: "\n", children - 1: "\n"}
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symInterfaceType, symStructType:
		// If it's type declaration it should be `type foo interface {...}`
		// If it's parameter declaration `foo(bar interface{})`
		// We do see `type foo struct{}` too, but it's probably okay to have an extra space when the model
		// only predicts up to `struct {...}`
		pSym := int(render.SafeSymbol(render.SafeParent(n)))
		if pSym == symTypeSpec || pSym == symCompositeLiteral {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: map[int]string{1: " "}, childCount: children}
		}

	case symParameterDeclaration:
		if children > 1 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: map[int]string{children - 1: " "}, childCount: children}
		}

	case symVariadicParameterDeclaration:
		return &spacerVisitor{p: p, parentNode: n, exactlyAfter: map[int]string{1: " "}, childCount: children}

	case symRangeClause, symForStatement:
		return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}

	case symIfStatement:
		exactlyAfter := make(map[int]string)
		for i := 0; i < children-1; i++ {
			if int(nodeChildren[i+1].Symbol()) != anonSymSemi {
				exactlyAfter[i+1] = " "
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symBinaryExpression:
		// In some occasions we want the binary expression to be compact
		// The rules are a bit complicated, so we only have simple heuristics here for now
		// TODO (Caren): replicate the logic in https://golang.org/src/go/printer/nodes.go (L#660)
		parent := render.SafeParent(n)
		pSym := render.SafeSymbol(parent)
		var compact bool
		if (pSym == symExpressionList && parent.ChildCount() > 1) ||
			((pSym == symArgumentList || pSym == symSpecialArgumentList) && parent.ChildCount() > 3) ||
			pSym == symSliceExpression || pSym == symIndexExpression {
			// Check is the middle is an operator
			if children == 3 && isOperator(nodeChildren[1]) {
				compact = true
			}
		}
		if !compact {
			return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}
		}

	case symConstDeclaration, symVarDeclaration, symTypeDeclaration:
		if children > 2 && int(nodeChildren[1].Symbol()) == anonSymLparen {
			exactlyAfter := make(map[int]string)
			exactlyAfter[1] = " "
			for i := 1; i < children-1; i++ {
				exactlyAfter[i+1] = "\n"
			}
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symVarSpec, symConstSpec:
		exactlyAfter := make(map[int]string)
		for i := 0; i < children-1; i++ {
			if int(nodeChildren[i+1].Symbol()) != anonSymComma {
				exactlyAfter[i+1] = " "
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symFunctionDeclaration, symMethodDeclaration:
		exactlyAfter := make(map[int]string)
		for i, c := range nodeChildren {
			cSym := int(c.Symbol())
			if cSym != aliasSymFieldIdentifier && cSym != symIdentifier && cSym != symBlock {
				exactlyAfter[i+1] = " "
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symFunctionType, symFuncLiteral, symMethodSpec:
		exactlyAfter := make(map[int]string)
		for i, c := range nodeChildren {
			if int(c.Symbol()) == symParameterList {
				exactlyAfter[i+1] = " "
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symExpressionSwitchStatement, symSelectStatement, symTypeSwitchStatement:
		var lbracePos int
		exactlyAfter := make(map[int]string)
		for i, n := range nodeChildren {
			cSym := int(n.Symbol())
			if cSym == anonSymColonEq {
				exactlyAfter[i] = " "
				exactlyAfter[i+1] = " "
			}
			if cSym == anonSymLbrace {
				lbracePos = i
				break
			}
		}
		if lbracePos > 0 && lbracePos < children-1 {
			exactlyAfter[lbracePos] = " "
			for i := lbracePos; i < children-1; i++ {
				exactlyAfter[i+1] = "\n"
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symCommunicationCase, symExpressionCase, symDefaultCase, symTypeCase:
		// Figure out the position of the colon
		var colonPos int
		for i, c := range nodeChildren {
			if int(c.Symbol()) == anonSymColon {
				colonPos = i
			}
		}

		exactlyAfter := make(map[int]string)
		depthAfter := make(map[int]int)
		exactlyAfter[colonPos+1] = "\n"

		// There are statements after colon, each one needs to have indentation
		if children >= colonPos+2 {
			for i := colonPos; i < children; i++ {
				depthAfter[i+1] = 1
			}
		}

		if len(depthAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter, depthAfter: depthAfter}
		}

	case symSliceExpression:
		// If one of `start` and `end` is non-identifier-like, meaning it's an expression, put space around colon
		start := n.ChildByFieldName("start")
		end := n.ChildByFieldName("end")
		if start != nil && end != nil {
			sSym, eSym := int(start.Symbol()), int(end.Symbol())
			// At least one of them is not simple identifier or int literal
			if (sSym != symIdentifier && sSym != symIntLiteral) || (eSym != symIdentifier && eSym != symIntLiteral) {
				exactlyAfter := make(map[int]string)
				for i, n := range nodeChildren {
					if int(n.Symbol()) == anonSymColon {
						exactlyAfter[i] = " "
						exactlyAfter[i+1] = " "
					}
				}
				return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
			}
		}

	case symForClause:
		exactlyAfter := make(map[int]string)
		for i, c := range nodeChildren {
			if int(c.Symbol()) == anonSymSemi {
				exactlyAfter[i+1] = " "
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symLabeledStatement, symEmptyLabeledStatement:
		if children > 2 {
			// Put the label one depth ahead, then walk the children node
			p.recordDepthDecrease(1)
			treesitter.Walk(p, nodeChildren[0])
			treesitter.Walk(p, nodeChildren[1])
			p.print("\n")
			// Walk the body of the labeled statement
			p.recordDepthIncrease(1)
			for _, c := range nodeChildren[2:] {
				treesitter.Walk(p, c)
			}
			return nil
		}

	case symLiteralValue:
		// Basic rules: if the user started typing, respect it.
		// Otherwise, all slice types are in same line, everything else are in multiple lines.
		exactlyAfter := make(map[int]string)
		if children > 2 && p.ifLiteralValueNewLine(n, nodeChildren) {
			exactlyAfter[1] = "\n"
			for i := 1; i < children-1; i++ {
				if int(nodeChildren[i].Symbol()) == anonSymComma {
					exactlyAfter[i+1] = "\n"
				}
			}
			exactlyAfter[children-1] = "\n"
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symArgumentList:
		// Basic rules: if the user started typing and all arguments are in different lines, we do the same.
		// Otherwise, everything is in the same line.
		exactlyAfter := make(map[int]string)
		if p.ifArgumentsNewLine(n, nodeChildren) {
			exactlyAfter[1] = "\n"
			for i := 1; i < children-1; i++ {
				if int(nodeChildren[i].Symbol()) == anonSymComma {
					exactlyAfter[i+1] = "\n"
				}
			}
			exactlyAfter[children-1] = "\n"
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}
	}

	if children == 0 {
		p.printNodeSrc(n)
		return nil
	}
	return p
}

func (p *prettyPrinter) ifArgumentsNewLine(n *sitter.Node, nodeChildren []*sitter.Node) bool {
	children := len(nodeChildren)

	// Respect the user's current setting if cursor position is in the middle of it
	// And user has already started typing
	if children > 2 && render.CursorInsideNode(n, p.snippetBegin) {
		// `{}` are on the same line
		if nodeChildren[0].StartPoint().Row == nodeChildren[children-1].StartPoint().Row {
			return false
		}

		// Now `(` and `)` are at different lines
		// Check if `(` and all commas are in different lines
		var seenInSameLine bool
		lastCommaRow := -1
		for _, c := range nodeChildren {
			if int(c.StartByte()) >= p.snippetBegin {
				break
			}
			cSym := int(c.Symbol())
			if cSym == anonSymComma || cSym == anonSymLparen {
				currentCommaRow := int(c.StartPoint().Row)
				if currentCommaRow == lastCommaRow {
					seenInSameLine = true
					break
				}
				lastCommaRow = currentCommaRow
			}
		}
		return !seenInSameLine
	}
	return false
}

func (p *prettyPrinter) ifLiteralValueNewLine(n *sitter.Node, nodeChildren []*sitter.Node) bool {
	children := len(nodeChildren)

	// Respect the user's current setting if cursor position is in the middle of it
	// And user has already started typing
	if children >= 2 && render.CursorInsideNode(n, p.snippetBegin) && p.snippetBegin > int(nodeChildren[1].StartByte()) {
		return nodeChildren[0].StartPoint().Row != nodeChildren[children-1].StartPoint().Row
	}

	// Otherwise we put slice literals in the same line
	// Everything else in different lines
	prev := n.PrevSibling()
	return prev != nil && render.SafeSymbol(prev) != symSliceType
}

func (p *prettyPrinter) recordMapping(startb, endb int, content string) {
	// record the mappings of the node's old position to the new position, without recording
	// two different mappings for the same position.
	if endb >= p.snippetBegin-startMappingBeforeBegin && startb <= p.snippetEnd+keepMappingAfterEnd {
		p.mappings = append(p.mappings, render.OffsetMapping{
			StartBefore: startb,
			StartAfter:  p.pos - len(content),
			EndBefore:   endb,
			EndAfter:    p.pos,
		})
	}
}

type spacerVisitor struct {
	p *prettyPrinter

	// the node that triggered the use of this spacerVisitor. If this is a statement and
	// p.conf.StatementNewline is true, a newline is inserted on exit.
	parentNode *sitter.Node

	// if exactlyAfter is not set, then this spacing character is inserted between each child node
	// up to maxSpaces.
	spaceChar rune

	// if set, inserts the corresponding space char after those nodes, otherwise
	// inserts between each node, up until maxSpaces.
	exactlyAfter map[int]string

	// depthAfter increments p.depth by certain value after those nodes, and decrements it immediately
	// after walking that node.
	depthAfter map[int]int

	// 0=no limit, < 0 = no space inserted (unless exactlyAfter is set), otherwise max number
	// of spaceChar to insert between child nodes.
	maxSpaces int

	seen       int
	childCount int
}

func isKeyword(n *sitter.Node) bool {
	switch render.SafeSymbol(n) {
	case anonSymBreak, anonSymCase, anonSymChan, anonSymConst, anonSymContinue,
		anonSymDefault, anonSymDefer, anonSymElse, anonSymFallthrough, anonSymFor,
		anonSymFunc, anonSymGo, anonSymGoto, anonSymIf, anonSymImport,
		anonSymInterface, anonSymMap, anonSymPackage, anonSymRange, anonSymReturn,
		anonSymSelect, anonSymStruct, anonSymSwitch, anonSymType, anonSymVar:
		return true
	default:
		return false
	}
}

func spaceAfter(n *sitter.Node) bool {
	switch render.SafeSymbol(n) {
	case anonSymBreak, anonSymCase, anonSymChan, anonSymConst, anonSymContinue,
		anonSymDefault, anonSymDefer, anonSymElse, anonSymFallthrough, anonSymFor,
		anonSymGo, anonSymGoto, anonSymIf, anonSymImport,
		anonSymPackage, anonSymRange, anonSymReturn,
		anonSymSelect, anonSymSwitch, anonSymVar:
		return true
	default:
		return false
	}
}

func isIdentifierLike(n *sitter.Node) bool {
	switch render.SafeSymbol(n) {
	case symFloatLiteral, symIntLiteral, symRawStringLiteral, symInterpretedStringLiteral,
		symIdentifier, symBlankIdentifier, symRuneLiteral, symImaginaryLiteral, symPointerType,
		aliasSymTypeIdentifier, aliasSymPackageIdentifier, aliasSymFieldIdentifier,
		symTrue, symFalse, symNil, aliasSymLabelName:
		return true
	default:
		return isKeyword(n)
	}
}

func isOperator(n *sitter.Node) bool {
	switch render.SafeSymbol(n) {
	case anonSymPlus, anonSymDash, anonSymStar, anonSymSlash, anonSymPercent,
		anonSymAmp, anonSymCaret, anonSymPipe, anonSymAmpCaret, anonSymLtLt, anonSymGtGt:
		return true
	default:
		return false
	}
}

func (s *spacerVisitor) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		if render.SafeSymbol(s.parentNode) == symERROR && s.p.lastPrintedNode != nil {
			// we're done visiting the children of an ERROR node, print any remaining content
			// that was not covered by children
			if s.p.lastPrintedNode.EndByte() < s.parentNode.EndByte() {
				content := string(s.p.src[s.p.lastPrintedNode.EndByte():s.parentNode.EndByte()])
				s.p.print(content)
				s.p.recordMapping(int(s.p.lastPrintedNode.EndByte()), int(s.parentNode.EndByte()), content)
			}
		}
		return nil
	}

	if len(s.exactlyAfter) == 0 {
		// insert the spaceChar between each child
		if s.seen > 0 && s.seen < s.childCount && (s.maxSpaces == 0 || s.seen <= s.maxSpaces) {
			s.p.print(string(s.spaceChar))
		}
	} else if ch, ok := s.exactlyAfter[s.seen]; ok {
		s.p.print(ch)
	}

	if n, ok := s.depthAfter[s.seen]; ok {
		s.p.recordDepthIncrease(n)
	}

	s.seen++
	treesitter.Walk(s.p, n)

	if n, ok := s.depthAfter[s.seen-1]; ok {
		s.p.recordDepthDecrease(n)
	}
	return nil
}

func (p *prettyPrinter) printNodeSrc(n *sitter.Node) {
	content := n.Content(p.src)
	sym := render.SafeSymbol(n)

	if isIdentifierLike(p.lastPrintedNode) && isIdentifierLike(n) {
		p.print(" ")
	}

	if spaceAfter(p.lastPrintedNode) && sym != anonSymColon {
		p.print(" ")
	}

	if content == "]" || content == ")" {
		p.recordDepthDecrease(1)
	}

	// No indent and dedent for `case` in `switch` and `select` block
	if content == "}" {
		pSym := render.SafeSymbol(render.SafeParent(n))
		if pSym != symSelectStatement && pSym != symExpressionSwitchStatement && pSym != symTypeSwitchStatement {
			p.recordDepthDecrease(1)
		}
	}

	// for single-line comments, add a space before the comment
	if sym == symComment && strings.HasPrefix(content, "//") {
		p.print(" ")
	}

	p.print(content)

	// record the mappings of the node's old position to the new position, without recording
	// two different mappings for the same position.
	startb, endb := int(n.StartByte()), int(n.EndByte())
	p.recordMapping(startb, endb, content)

	if sym == anonSymComma && p.conf.SpaceAfterComma {
		p.print(" ")
	}

	if content == "[" || content == "(" {
		p.recordDepthIncrease(1)
	}

	// No indent and dedent for `case` in `switch` and `select` block
	if content == "{" {
		pSym := render.SafeSymbol(render.SafeParent(n))
		if pSym != symSelectStatement && pSym != symExpressionSwitchStatement && pSym != symTypeSwitchStatement {
			p.recordDepthIncrease(1)
		}
	}

	// for single-line comments, add a space before the comment
	if sym == symComment && strings.HasPrefix(content, "//") {
		p.print("\n")
	}

	p.lastPrintedNode = n
}

// The following two methods record the intention of increasing/decreasing
// depth - the actual increase or decrease is only done when a newline is
// encountered in print, and then the number of increase vs decrease encountered
// is checked to determine if depth did increase or decrease.
func (p *prettyPrinter) recordDepthIncrease(n int) {
	p.pendingDepthChange += n
}
func (p *prettyPrinter) recordDepthDecrease(n int) {
	p.pendingDepthChange -= n
}

func (p *prettyPrinter) print(s string) {
	if p.err != nil || s == "" {
		return
	}

	if strings.TrimSpace(s) == "" {
		// this is whitespace only, apply the pending write logic with potential
		// whitespace collapse/drop.
		switch {
		case p.pendingWrite == " " && s == " ":
			// drop the duplicate whitespace, nothing to do - the single space will
			// be written on the next non-whitespace call to print.
			return
		case p.pendingWrite == "\n" && s == "\n":
			// drop the duplicate whitespace, nothing to do - the single newline will
			// be written on the next non-whitespace call to print.
			// NOTE: this means that there will be no separating blank lines in the output.
			return
		case p.pendingWrite == "\n" && s == " ":
			// drop the new space, a newline plays the role of a space
			return
		case p.pendingWrite == " " && s == "\n":
			// replace the pending write with the newline, replaces the space
			p.pendingWrite = s
			return
		case p.pendingWrite == "":
			// no pending write, so s becomes the pending write
			p.pendingWrite = s
			return
		default:
			p.err = errors.Errorf("unexpected situation: pending write=%q, new whitespace write=%q", p.pendingWrite, s)
		}
	}

	// if the pending write is a newline, the indent must also be written
	if p.pendingWrite == "\n" {
		depthChange := p.pendingDepthChange
		p.pendingDepthChange = 0

		switch {
		case depthChange > 0:
			p.depth += depthChange

		case depthChange < 0:
			p.depth += depthChange
			if p.depth < 0 {
				p.depth = 0
			}
		}
		p.pendingWrite += strings.Repeat(p.conf.Indent, p.depth)
	} else {
		// If pending write is not "\n", reset pending depth change
		p.pendingDepthChange = 0
	}

	var n int
	n, p.err = fmt.Fprint(p.w, p.pendingWrite+s)
	p.pos += n
	p.pendingWrite = ""
}
