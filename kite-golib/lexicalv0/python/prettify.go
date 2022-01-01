package python

import (
	"fmt"
	"io"
	"strings"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
)

const symERROR = 65535

// Config is the set of formatting options to pass to Prettify to format
// an python AST back to source code.
type Config struct {
	Indent                      string
	SpaceAfterComma             bool // true=always, false=never
	SpaceAfterColonInPair       bool // true=always, false=never
	SpaceAfterColonInSlice      bool // true=always, false=never
	SpaceAfterColonInTypedParam bool // true=always, false=never
	SpaceAfterColonInLambda     bool // true=always, false=never
	SpaceAroundArrow            bool // true=always, false=never
	SpaceInfixOps               bool // true=always, false=never
	SpaceInKeywordArguments     bool // true=always, false=never
	BlankLinesBeforeClassDef    int  // standard practice is 2
	BlankLinesBeforeTopFuncDef  int  // standard practice is 2
	BlankLinesBetweenMethods    int  // standard practice is 1
	ListItemsNewLine            int  // 1=always, 0=never, -1=flexible
	DictionaryItemsNewLine      int  // 1=always, 0=never, -1=flexible
	FuncParamsNewLine           int  // 1=always, 0=never, -1=flexible
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

	// begin and end of completion snippet
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
	lockedDepthDecr    bool
}

func (p *prettyPrinter) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil || p.err != nil {
		return nil
	}
	// Look at the first-level nodes. Simply print the ones before snippet,
	// go through the ones that contain the snippet,
	// and ignore the ones after snippet.
	if render.SafeSymbol(render.SafeParent(n)) == symModule {
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
	// fmt.Printf("%s | sym: %d (type: %s) | parent sym: %d | %d-%d | nchild: %d | %q\n", n, int(n.Symbol()), n.Type(), render.SafeSymbol(render.SafeParent(n)), n.StartByte(), n.EndByte(), n.ChildCount(), n.Content(p.src))

	switch {
	case sym == symERROR:
		errChildren := make([]*sitter.Node, children)
		for i := 0; i < children; i++ {
			errChildren[i] = n.Child(i)
		}
		exactlyAfter := make(map[int]string)
		for i := 1; i < children; i++ {
			if errChildren[i-1].EndPoint() != errChildren[i].StartPoint() {
				exactlyAfter[i] = string(p.src[errChildren[i-1].EndByte():errChildren[i].StartByte()])
			}
		}
		// on exit of the spacer, if parentNode is ERROR, it writes
		// whatever part of the source that was not parsed into a child node,
		return &spacerVisitor{p: p, parentNode: n, childCount: children, maxSpaces: -1, exactlyAfter: exactlyAfter}

	case sym == symDefaultParameter || sym == symKeywordArgument || sym == symTypedDefaultParameter:
		exactlyAfter := make(map[int]string)
		if p.conf.SpaceInKeywordArguments {
			for i := 0; i < children; i++ {
				if int(n.Child(i).Symbol()) == anonSymEq {
					exactlyAfter[i] = " "
					exactlyAfter[i+1] = " "
					break
				}
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case sym == symString:
		p.printNodeSrc(n)
		return nil

	case sym == symList || sym == symDictionary || sym == symParameters || sym == symArgumentList:
		var cList []*sitter.Node
		for i := 0; i < children; i++ {
			cList = append(cList, n.Child(i))
		}
		var config int
		var depth int
		switch sym {
		case symList:
			config = p.conf.ListItemsNewLine
			depth = 1
		case symDictionary:
			config = p.conf.DictionaryItemsNewLine
			depth = 1
		case symArgumentList:
			config = p.conf.FuncParamsNewLine
			depth = 1
		case symParameters:
			config = p.conf.FuncParamsNewLine
			depth = 2
		}
		if p.ifMultiLine(n, cList, config) && children > 2 {
			exactlyAfter, depthAfter := newLinesInParen(cList, depth)
			if len(exactlyAfter) > 0 {
				return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter, depthAfter: depthAfter}
			}
		}

	case sym == symAssignment || sym == symBinaryOperator || sym == symComparisonOperator ||
		sym == symConditionalExpression || sym == symAugmentedAssignment:
		if p.conf.SpaceInfixOps {
			return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}
		}

	case pyStatements[sym] == true || pyClauses[sym] == true:
		p.extraBlankLines(n)
		exactlyAfter := make(map[int]string)
		depthAfter := make(map[int]int)
		for i := 0; i < children; i++ {
			childSym := int(n.Child(i).Symbol())
			if childSym == symBlock {
				exactlyAfter[i] = "\n"
				depthAfter[i] = 1
			}
			if childSym == symDecorator {
				exactlyAfter[i+1] = "\n"
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, maxSpaces: -1, depthAfter: depthAfter, exactlyAfter: exactlyAfter}
		}
		return &spacerVisitor{p: p, parentNode: n, childCount: children, maxSpaces: -1}
	}

	if children == 0 {
		p.printNodeSrc(n)
		return nil
	}
	return p
}

// Python style guide asks for certain number of blank lines between code blocks
func (p *prettyPrinter) extraBlankLines(n *sitter.Node) {
	sym := int(n.Symbol())
	parent := render.SafeParent(n)
	parentSym := render.SafeSymbol(parent)
	if sym == symClassDefinition {
		for i := 0; i < p.conf.BlankLinesBeforeClassDef; i++ {
			p.print("\n")
		}
		return
	}

	if sym == symDecoratedDefinition ||
		(sym == symFunctionDefinition && parentSym != symDecoratedDefinition) {
		// If it's a method of some class and it's not the first child
		// Apply BlankLinesBetweenMethods
		if parentSym == symBlock && render.SafeSymbol(render.SafeParent(parent)) == symClassDefinition {
			if !render.SafeEqual(n, render.SafeChild(parent, 0)) {
				for i := 0; i < p.conf.BlankLinesBetweenMethods; i++ {
					p.print("\n")
				}
				return
			}
		}
		// Top-level function, and not the first child
		// Apply BlankLinesBeforeTopFuncDef
		if parentSym == symModule && !render.SafeEqual(n, render.SafeChild(parent, 0)) {
			for i := 0; i < p.conf.BlankLinesBeforeTopFuncDef; i++ {
				p.print("\n")
			}
			return
		}
	}
}

func newLinesInParen(cList []*sitter.Node, depth int) (map[int]string, map[int]int) {
	children := len(cList)
	exactlyAfter := make(map[int]string)
	depthAfter := make(map[int]int)
	exactlyAfter[1] = "\n"
	depthAfter[1] = depth
	exactlyAfter[children-1] = "\n"
	for i := 1; i < children-1; i++ {
		if int(cList[i].Symbol()) == anonSymComma {
			exactlyAfter[i+1] = "\n"
			depthAfter[i+1] = depth
		}
	}
	return exactlyAfter, depthAfter
}

// Same logic for deciding if we want new lines in list/dict/call
func (p *prettyPrinter) ifMultiLine(n *sitter.Node, cList []*sitter.Node, config int) bool {
	children := int(n.ChildCount())
	if children <= 2 || config == 0 {
		return false
	}
	if config == 1 {
		return true
	}

	// Respect the current setting
	if p.snippetBegin >= int(cList[0].EndByte()) && p.snippetEnd < int(cList[children-1].StartByte()) ||
		p.snippetBegin > int(cList[0].EndByte()) && p.snippetEnd <= int(cList[children-1].StartByte()) {
		lastSymbolRow := -1
		multiLines := true
		for i, child := range cList {
			// Only inspect the children outside of the snippet
			if p.snippetBegin <= int(child.EndByte()) && p.snippetEnd >= int(child.StartByte()) {
				continue
			}
			if i == 0 || i == children-1 || int(child.Symbol()) == anonSymComma {
				if lastSymbolRow == int(child.StartPoint().Row) {
					multiLines = false
					break
				}
				lastSymbolRow = int(child.StartPoint().Row)
			}
		}
		if multiLines {
			return true
		}
		return false
	}

	// When the snippet fully include the arguments
	if children >= 11 || cList[children-1].EndPoint().Column >= 80 {
		return true
	}
	return false
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

func isSingleStatement(n *sitter.Node) bool {
	sym := render.SafeSymbol(n)
	if !pyStatements[sym] {
		return false
	}
	children := int(n.ChildCount())
	for i := 0; i < children; i++ {
		childSym := int(n.Child(i).Symbol())
		if childSym == symBlock || pyStatements[childSym] {
			return false
		}
	}
	return true
}

func isIdentifierLike(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	sym := int(n.Symbol())
	if sym == symIdentifier || sym == symInteger || sym == symFloat ||
		sym == symString || sym == symTrue || sym == symFalse || sym == symNone {
		return true
	}

	return pyKeywords[sym]
}

func isKeyword(n *sitter.Node) bool {
	if n == nil {
		return false
	}
	return pyKeywords[int(n.Symbol())]
}

func (s *spacerVisitor) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		if isSingleStatement(s.parentNode) && render.SafeSymbol(render.SafeParent(s.parentNode)) != symERROR {
			s.p.print("\n")
		}
		if render.SafeSymbol(s.parentNode) == symERROR && s.p.lastPrintedNode != nil {
			// we're done visiting the children of an ERROR node, print any remaining content
			// that was not covered by children
			if s.p.lastPrintedNode.EndByte() < s.parentNode.EndByte() {
				s.p.print(string(s.p.src[s.p.lastPrintedNode.EndByte():s.parentNode.EndByte()]))
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

var allowedMissingContent = map[string]bool{
	"]": true,
	"}": true,
	")": true,
}

func (p *prettyPrinter) printNodeSrc(n *sitter.Node) {
	content := n.Content(p.src)
	sym := int(n.Symbol())

	if n.IsMissing() {
		// this is a MISSING node, which has the symbol and type of the missing
		// punctuation (e.g. } or ]). Whitelist only those specific missing
		// nodes, to avoid adding otherwise non-MISSING content.
		if typ := n.Type(); allowedMissingContent[typ] {
			content = typ
		}
	}

	if isIdentifierLike(p.lastPrintedNode) && isIdentifierLike(n) {
		p.print(" ")
	} else if isKeyword(p.lastPrintedNode) && content != ":" {
		p.print(" ")
	}

	if sym == anonSymRarrow && p.conf.SpaceAroundArrow {
		p.print(" ")
	}

	if p.lastPrintedNode != nil && sym == anonSymAs {
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

	// Handle colons, different for typed parameter, lambda, slice and pair
	if sym == anonSymColon {
		parent := n.Parent()
		if parent != nil {
			pSym := int(parent.Symbol())
			if (pSym == symPair && p.conf.SpaceAfterColonInPair) ||
				(pSym == symSlice && p.conf.SpaceAfterColonInSlice) ||
				((pSym == symTypedParameter || pSym == symTypedDefaultParameter) && p.conf.SpaceAfterColonInTypedParam) ||
				(pSym == symLambda && p.conf.SpaceAfterColonInLambda) {
				p.print(" ")
			}
		}
	}

	if sym == anonSymRarrow && p.conf.SpaceAroundArrow {
		p.print(" ")
	}

	p.lastPrintedNode = n
}

// The following two methods record the intention of increasing/decreasing
// depth - the actual increase or decrease is only done when a newline is
// encountered in print, and then the number of increase vs decrease encountered
// is checked to determine if depth did increase or decrease.
func (p *prettyPrinter) recordDepthIncrease(n int) {
	if p.lockedDepthDecr {
		p.lockedDepthDecr = false
		p.pendingDepthChange = 1
		return
	}
	p.pendingDepthChange += n
}
func (p *prettyPrinter) recordDepthDecrease(n int) {
	if p.lockedDepthDecr {
		return
	}
	p.pendingDepthChange -= n
}

func (p *prettyPrinter) print(s string) {
	if p.err != nil || s == "" {
		return
	}

	if strings.TrimSpace(s) == "" {
		if p.lockedDepthDecr && strings.Contains(s, "\n") {
			p.lockedDepthDecr = false
		}

		// this is whitespace only, apply the pending write logic with potential
		// whitespace collapse/drop.
		switch {
		case p.pendingWrite == " " && s == " ":
			// drop the duplicate whitespace, nothing to do - the single space will
			// be written on the next non-whitespace call to print.
			return
		case emptyAndNewLine(p.pendingWrite) && s == " ":
			// drop the new space, a newline plays the role of a space
			return
		case emptyAndNewLine(p.pendingWrite) && emptyAndNewLine(s):
			p.write()
			p.pendingWrite = s
			return
		case p.pendingWrite == " " && emptyAndNewLine(s):
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
	p.write()
	p.pendingWrite = s
	p.write()
	p.pendingWrite = ""
}

func (p *prettyPrinter) write() {
	// if the pending write nothing but new line, the indent must also be written
	if emptyAndNewLine(p.pendingWrite) {
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
			p.lockedDepthDecr = true
		}
		p.pendingWrite += strings.Repeat(p.conf.Indent, p.depth)
	}

	var n int
	n, p.err = fmt.Fprint(p.w, p.pendingWrite)
	p.pos += n
}

func emptyAndNewLine(s string) bool {
	return strings.TrimSpace(s) == "" && strings.Contains(s, "\n")
}
