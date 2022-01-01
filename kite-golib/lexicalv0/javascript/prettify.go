package javascript

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
// an javascript AST back to source code.
//
// Based on https://eslint.org/docs/rules/#stylistic-issues
// leaving out things that alter the naming of identifiers, comments,
// and consistency (as we only have a partial view of the whole).
type Config struct {
	ArrayBracketNewline            int  // 1=always, 0=never, -1=auto-heuristics e.g. `[1,2,3]` -> `[\n1,2,3\n]`
	ArrayBracketSpacing            bool // true=always, false=never e.g. `[1,2,3]` -> `[ 1,2,3 ]`
	ArrayElementNewline            int  // 1=always, 0=never, -1=auto-heuristics e.g. `[1,2,3]` -> `[1,\n2,\n3]`
	ArrowSpacingBefore             bool // true=always, false=never e.g. `()=>x` -> `() =>x`
	ArrowSpacingAfter              bool // true=always, false=never e.g. `()=>x` -> `()=> x`
	BlockSpacing                   bool // true=always, false=never e.g. `if (x) {return y;}` -> `if (x) { return y; }`
	CommaSpacingBefore             bool // true=always, false=never e.g. `(1,2)` -> `(1 ,2)`
	CommaSpacingAfter              bool // true=always, false=never e.g. `(1,2)` -> `(1, 2)`
	ComputedPropertySpacing        bool // true=always, false=never e.g. `obj[name]` -> `obj[ name ]`
	FuncCallSpacing                bool // true=always, false=never e.g. `f()` -> `f ()`
	FuncParamArgumentNewline       int  // 1=always, 0=never, -1=auto-heuristics e.g. `f(a,b)` -> `f(a,\nb)`
	FuncParenNewline               int  // 1=always, 0=never, -1=auto-heuristics e.g. `f(a,b)` -> `f(\na,b\n)`
	ImplicitArrowLinebreak         bool // true=below, false=beside e.g. `()=>1` -> `()=>\n1`
	Indent                         int  // 0=none, > 0 = number of spaces, < 0 = number of tabs
	KeySpacingBeforeColon          bool // true=space, false=no space e.g. `{k:v}` -> `{k :v}`
	KeySpacingAfterColon           bool // true=space, false=no space e.g. `{k:v}` -> `{k: v}`
	KeywordSpacingBefore           bool // true=space, false=no space (only after '}' and ')', e.g. `if(x){}else{}` -> `if(x){} else{}`)
	KeywordSpacingAfter            bool // true=space, false=no space (only before '{' and '(', e.g. `if(x){}else{}` -> `if (x){}else {}`)
	NonBlockStatementBodyLinebreak bool // true=below, false=beside e.g. `if(x) y;` -> `if(x)\ny;`
	ObjectCurlyNewline             int  // 1=always, 0=never, -1=auto-heuristics
	ObjectCurlySpacing             bool // true=always, false=never
	ObjectPropertyNewline          int  // 1=always, 0=never, -1=auto-heuristics
	Semicolon                      bool // true=always, false=never
	SpaceBeforeBlocks              bool // true=always, false=never
	SpaceBeforeFuncParen           bool // true=always, false=never e.g. `function f()` -> `function f ()`
	SpaceInParens                  bool // true=always, false=never e.g. `f(x)` -> `f( x )`
	SpaceInfixOps                  bool // true=always, false=never
	SpaceUnaryOpsWords             bool // true=always, false=never
	SpaceUnaryOpsNonWords          bool // true=always, false=never
	StatementNewline               bool // true=always, false=never put each statement on its own line, incl. block statements
	SwitchColonSpacingBefore       bool // true=always, false=never
	SwitchColonSpacingAfter        bool // true=always, false=never
	SwitchColonNewLine             bool // true=always, false=never
	TemplateTagSpacing             bool // true=always, false=never
	JsxFragmentChildrenNewline     bool // true=always, false=never
	JsxElementChildrenNewline      int  // 1=always, 0=never, -1=auto-heuristics
	JsxAttributeNewline            int  // 1=always, 0=never, -1=auto-heuristics

	// Internal use
	currentJsxElementNewline bool // Keep track of the current status of `JsxElementNewline`
}

// NOTE: those could be added for more flexibility, leaving them out at the moment as
// they are not necessary to validate the approach.
//BraceStyle                     string // 1tbs, stroustrup or allman
//CommaDangle                    bool   // true=always-multiline, false=never
//CommaStyleFirst                bool   // true=first, false=last
//NewlinePerChainedCall          int    // 0=no, > 0 = newline when chain longer than this depth

var unaryWords = map[string]bool{
	"delete": true,
	"new":    true,
	"typeof": true,
	"void":   true,
	"yield":  true,
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

		asiFromFuncOrClassDeclEndLine: -1,
		jsxElementNewLine:             make(map[uint32]bool),
	}
	// treesitter.Print(n, os.Stdout, "  ")
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

	// Field to indicate that we should process the special-case automatic
	// semicolon insertion after a function or class declaration. That rule
	// states that after such a declaration, if the trailing '}' is followed by a
	// newline before the next node (and that node doesn't start with '(' or
	// '['), insert a semicolon.
	//
	// The field contains the ending line of the declaration, so that when
	// processing the subsequent node, we can check if there is a newline between
	// the two. It is set to -1 when there is no such special case to handle
	// because treesitter reports lines with a 0-based index.
	asiFromFuncOrClassDeclEndLine int

	// whitespace is first stored in pendingWrite to e.g. collapse spaces, drop them
	// if a newline replaces it, etc. It gets written when a non-whitespace write (i.e.
	// a token's content) gets printed.
	pendingWrite string

	// whitespace (newline) inserted to play the role of semicolon is first added
	// to pendingSemi and is only written if there was no newline added by
	// pendingWrite and the next non-whitespace char is not a closing brace '}'.
	pendingSemi string

	// keep track of the last printed node, to ensure mandatory spacing is inserted
	// between identifiers/keywords (including number literals).
	lastPrintedNode *sitter.Node

	// keep track if the last non-whitespace printed value was an
	// automatically-added semicolon, to prevent writing two consecutive
	// semicolons. Non-automatically-added semicolons are ok, e.g. in
	// `for(;;){...}`.
	lastPrintWasAddedSemi bool

	// current depth of statements, i.e. number of indent spacing required before
	// writing non-whitespace on a line.
	depth int

	// recordings of the new-line option for all JSX elements nodes (indexed by the StartByte)
	jsxElementNewLine map[uint32]bool

	// keeps track of the depth changes that have been encountered, but not applied yet.
	// When an opening or closing paren, brace or bracket is seen, the depth is not
	// changed immediately - it is changed only on the next newline printed. In the
	// meantime, this field keeps track of whether the depth should be incremented
	// (if positive) or decremented (if negative).
	pendingDepthChange int
	// until a newline (or an indent via opening brace/paren/bracket) is seen,
	// depth cannot decrease again. This is for the case where we have e.g.
	//   [{
	//     ... increased content
	//   }] // <- here only the first closing brace should decrease depth, the second one shouldn't
	lockedDepthDecr bool

	// err is set to the first error encountered when writing to w. Once an error
	// is set, the rest of the writes are skipped and that error will be returned.
	err error
}

func (p *prettyPrinter) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil || p.err != nil {
		return nil
	}

	// Look at the first-level nodes. Simply print the ones before snippet,
	// go through the ones that contain the snippet,
	// and ignore the ones after snippet.
	if render.SafeSymbol(render.SafeParent(n)) == symProgram {
		if int(n.EndByte()) < p.snippetBegin {
			p.print(n.Content(p.src), true)
			return nil
		}
		if int(n.StartByte()) > p.snippetEnd {
			return nil
		}
	}

	if p.asiFromFuncOrClassDeclEndLine >= 0 {
		end := p.asiFromFuncOrClassDeclEndLine
		p.asiFromFuncOrClassDeclEndLine = -1

		if p.conf.Semicolon {
			// special ASI case: after a function/class declaration, if the trailing '}' is followed
			// by a newline before the next node (and that node doesn't start with '(' or '['),
			// insert a semicolon.
			start := int(n.StartPoint().Row)
			if start > end {
				// there is a newline between the declaration and the next token
				if s := n.Content(p.src); len(s) > 0 && s[0] != '(' && s[0] != '[' && s[0] != ';' {
					p.printAddedSemi()
				}
			}
		}
	}
	// fmt.Printf("%s | sym: %d (type: %s) | parent sym: %d | %d-%d | nchild: %d | %q\n", n, n.Symbol(), n.Type(), render.SafeSymbol(render.SafeParent(n)), n.StartByte(), n.EndByte(), n.ChildCount(), n.Content(p.src))
	currentSymbol := int(n.Symbol())
	children := int(n.ChildCount())
	nodeChildren := make([]*sitter.Node, children)
	for i := 0; i < children; i++ {
		nodeChildren[i] = n.Child(i)
	}

switchAgain:
	switch currentSymbol {
	// Handle ERROR node: recur into the children of the node and render the children properly
	// In between children - just keep it the original way
	case symERROR:
		// if this looks like a JSX element, re-process it as such
		if children > 2 && int(nodeChildren[2].Symbol()) == symJsxAttribute {
			content := n.Content(p.src)
			if strings.HasPrefix(content, "<") {
				currentSymbol = symJsxOpeningElement
				goto switchAgain
			}
		}
		exactlyAfter := make(map[int]rune)
		for i := 1; i < children; i++ {
			if nodeChildren[i-1].EndPoint() != nodeChildren[i].StartPoint() {
				exactlyAfter[i] = ' '
			}
		}

		// on exit of the spacer, if parentNode is ERROR, it writes
		// whatever part of the source that was not parsed into a child node,
		// e.g.:
		//   (ERROR (identifier)) | 0-12 | "render('abc)"
		//   (identifier) | 0-6 | "render"
		//   ("(") | 6-7 | "("
		//   ("'") | 7-8 | "'"
		// this should print "abc)" on exit, as that part of the ERROR is not
		// covered by any child.
		return &spacerVisitor{p: p, parentNode: n, childCount: children, maxSpaces: -1, exactlyAfter: exactlyAfter}

	case symIfStatement:
		// check if the if or else is a single non-block statement, and if so apply
		// the non-block statement rule. The if is different than the other statements
		// to which this rule applies, as it potentially has 2 places to apply it.
		spaceChar := spacingChar(p.conf.NonBlockStatementBodyLinebreak)
		exactlyAfter := make(map[int]rune)
		depthAfter := make(map[int]bool)
		if isNonBlockBodyField(n, "consequence") {
			exactlyAfter[2] = spaceChar
			depthAfter[2] = true
		}
		if isNonBlockBodyField(n, "alternative") {
			exactlyAfter[children-1] = spaceChar

			// special-case: if the else part is another if, do not increment depth
			if body := n.ChildByFieldName("alternative"); render.SafeSymbol(body) != symIfStatement {
				depthAfter[children-1] = true
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, depthAfter: depthAfter, childCount: children}
		}

	case symWhileStatement:
		ch := spacingChar(p.conf.NonBlockStatementBodyLinebreak)
		exactlyAfter := map[int]rune{2: ch}
		if v := p.checkNonBlockStatementBody(n, children, exactlyAfter); v != nil {
			return v
		}

	case symDoStatement:
		ch := spacingChar(p.conf.NonBlockStatementBodyLinebreak)
		exactlyAfter := map[int]rune{1: ch}
		if v := p.checkNonBlockStatementBody(n, children, exactlyAfter); v != nil {
			return v
		}

	case symForInStatement, symForOfStatement, symForStatement, symWithStatement:
		ch := spacingChar(p.conf.NonBlockStatementBodyLinebreak)
		exactlyAfter := map[int]rune{children - 1: ch}
		if v := p.checkNonBlockStatementBody(n, children, exactlyAfter); v != nil {
			return v
		}

	case symArrowFunction:
		nonBlockChar := spacingChar(p.conf.ImplicitArrowLinebreak)
		exactlyAfter := map[int]rune{children - 1: nonBlockChar}
		if p.conf.ArrowSpacingBefore {
			exactlyAfter[children-2] = ' '
		}
		if v := p.checkNonBlockBodyRule(n, children, exactlyAfter); v != nil {
			return v
		}

		// if it is a block body, must still apply the before/after arrow spacing rules
		if p.conf.ArrowSpacingAfter {
			exactlyAfter[children-1] = ' '
		} else {
			delete(exactlyAfter, children-1)
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symArray, aliasSymArrayPattern:
		exactlyAfter := make(map[int]rune)
		if p.conf.ArrayBracketSpacing && children > 2 {
			exactlyAfter[1] = ' '
			exactlyAfter[children-1] = ' '
		}
		if p.ifArrayNewLine(n, nodeChildren) && children > 2 {
			exactlyAfter[1] = '\n'
			exactlyAfter[children-1] = '\n'
			exactlyAfter = betweenCommaSepChildren(n, children, nodeChildren, false, '\n', exactlyAfter)
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symObject, aliasSymObjectPattern, symNamedImports, symExportClause:
		exactlyAfter := make(map[int]rune)
		// if there is at least 1 property in addition to '{' and '}'
		if p.conf.ObjectCurlySpacing && children > 2 { // no spacing when empty object
			exactlyAfter[1] = ' '
			exactlyAfter[children-1] = ' '
		}
		if p.ifObjectNewLine(n, nodeChildren) && children > 2 {
			exactlyAfter[1] = '\n'
			exactlyAfter[children-1] = '\n'
			exactlyAfter = betweenCommaSepChildren(n, children, nodeChildren, false, '\n', exactlyAfter)
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symPair:
		if p.conf.KeySpacingBeforeColon || p.conf.KeySpacingAfterColon {
			exactlyAfter := make(map[int]rune)
			if p.conf.KeySpacingBeforeColon {
				exactlyAfter[1] = ' '
			}
			if p.conf.KeySpacingAfterColon {
				exactlyAfter[2] = ' '
			}
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symFormalParameters:
		// formal parameters start and end with the paren
		if p.conf.SpaceBeforeFuncParen {
			p.print(" ", false)
		}

		exactlyAfter := make(map[int]rune)
		if p.conf.FuncParenNewline == 1 && children > 2 {
			exactlyAfter[1] = '\n'
			exactlyAfter[children-1] = '\n'
		}
		if p.conf.FuncParamArgumentNewline == 1 && children > 2 {
			exactlyAfter = betweenCommaSepChildren(n, children, nodeChildren, false, '\n', exactlyAfter)
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symArguments:
		if currentSymbol == symArguments && p.conf.FuncCallSpacing {
			p.print(" ", false)
		}
		exactlyAfter := make(map[int]rune)
		// After `(` and before `)`
		if p.conf.FuncCallSpacing && children > 2 {
			exactlyAfter[1] = ' '
			exactlyAfter[children-1] = ' '
		}
		if p.ifFuncParamNewLine(n, nodeChildren) && children > 2 {
			exactlyAfter[1] = '\n'
			exactlyAfter[children-1] = '\n'
			exactlyAfter = betweenCommaSepChildren(n, children, nodeChildren, false, '\n', exactlyAfter)
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symParenthesizedExpression:
		if children > 2 {
			exactlyAfter := make(map[int]rune)
			var hasJsx bool
			for i := 1; i < children-1; i++ {
				childSym := int(nodeChildren[i].Symbol())
				if childSym == symJsxElement || childSym == symJsxSelfClosingElement || childSym == symJsxOpeningElement {
					hasJsx = true
					break
				}
			}
			if hasJsx {
				exactlyAfter[1] = '\n'
				exactlyAfter[children-1] = '\n'
				exactlyAfter = betweenCommaSepChildren(n, children, nodeChildren, false, '\n', exactlyAfter)
			}
			if len(exactlyAfter) > 0 {
				return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
			}
		}

	case symClassBody:
		if p.conf.BlockSpacing || p.conf.StatementNewline {
			spaceChar := spacingChar(p.conf.StatementNewline)
			if p.conf.StatementNewline {
				// in that case, insert a newline between each child of the class body
				return &spacerVisitor{p: p, parentNode: n, childCount: children, spaceChar: spaceChar}
			}
			// otherwise insert just the spacing around the block
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: map[int]rune{1: spaceChar, children - 1: spaceChar}}
		}

		// return the spacerVisitor that will take care of the special case of adding semicolons to class body
		// members when StatementNewline is false.
		return &spacerVisitor{p: p, parentNode: n, maxSpaces: -1, childCount: children}

	case symStatementBlock, symSwitchBody:
		if p.conf.BlockSpacing || p.conf.StatementNewline {
			spaceChar := spacingChar(p.conf.StatementNewline)
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: map[int]rune{1: spaceChar, children - 1: spaceChar}}
		}

	case symSwitchCase, symSwitchDefault:
		// Figure out the position of the colon
		var colonPos int
		if currentSymbol == symSwitchCase {
			colonPos = 2
		} else {
			colonPos = 1
		}
		exactlyAfter := make(map[int]rune)
		depthAfter := make(map[int]bool)
		// If it has colon
		if children >= colonPos+1 {
			if p.conf.SwitchColonSpacingBefore {
				exactlyAfter[colonPos] = ' '
			}
		}
		// Node after colon is a statement block
		if children == colonPos+2 && int(nodeChildren[colonPos+1].Symbol()) == symStatementBlock {
			if p.conf.SwitchColonSpacingAfter {
				exactlyAfter[colonPos+1] = ' '
			}
		}
		// Non-block statements after colon, each one needs to have indentation
		if children >= colonPos+2 && int(nodeChildren[colonPos+1].Symbol()) != symStatementBlock {
			if p.conf.SwitchColonSpacingAfter || p.conf.SwitchColonNewLine {
				exactlyAfter[colonPos+1] = spacingChar(p.conf.SwitchColonNewLine)
				depthAfter[colonPos+1] = true
			}
			for i := colonPos + 1; i < children; i++ {
				exactlyAfter[i+1] = '\n'
				depthAfter[i+1] = true
			}
		}
		if len(exactlyAfter) > 0 {
			if len(depthAfter) > 0 {
				return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter, depthAfter: depthAfter}
			}
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symComputedPropertyName:
		if p.conf.ComputedPropertySpacing {
			return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}
		}

	case symCallExpression:
		if p.conf.TemplateTagSpacing {
			if children == 2 && int(nodeChildren[1].Symbol()) == symTemplateString {
				return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', maxSpaces: 1, childCount: children}
			}
		}

	case symBinaryExpression, symTernaryExpression, symAssignmentExpression,
		symAugmentedAssignmentExpression, symAssignmentPattern:
		if p.conf.SpaceInfixOps {
			return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}
		}

	case symVariableDeclarator:
		if p.conf.SpaceInfixOps {
			if children == 3 {
				return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}
			}
			exactlyAfter := make(map[int]rune)
			for i := 0; i < children; i++ {
				if i != 0 && int(nodeChildren[i].Symbol()) == anonSymEq {
					exactlyAfter[i] = ' '
					exactlyAfter[i+1] = ' '
					break
				}
			}
			if len(exactlyAfter) > 0 {
				return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
			}
		}

	case symPublicFieldDefinition:
		if p.conf.SpaceInfixOps && children == 3 {
			return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', childCount: children}
		}

	case symUnaryExpression:
		if p.conf.SpaceUnaryOpsWords || p.conf.SpaceUnaryOpsNonWords {
			isWord := unaryWords[nodeChildren[0].Type()]
			if (isWord && p.conf.SpaceUnaryOpsWords) || (!isWord && p.conf.SpaceUnaryOpsNonWords) {
				return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', maxSpaces: 1, childCount: children}
			}
		}

	case symUpdateExpression:
		if p.conf.SpaceUnaryOpsNonWords {
			return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', maxSpaces: 1, childCount: children}
		}

	case symNewExpression, symYieldExpression:
		if p.conf.SpaceUnaryOpsWords {
			return &spacerVisitor{p: p, parentNode: n, spaceChar: ' ', maxSpaces: 1, childCount: children}
		}

	case symRegex:
		p.printNodeSrc(n)
		return nil

	case symString, symTemplateString:
		// For string and template, the parser doesn't generate the content
		// Need to manually print them and log the position mapping
		var nextPos uint32
		for _, child := range nodeChildren {
			start := child.StartByte()
			if start > nextPos && nextPos != 0 {
				content := string(p.src[nextPos:start])
				p.recordMappingForString(int(nextPos), int(start), content)
			}
			treesitter.Walk(p, child)
			nextPos = child.EndByte()
		}
		return nil

	case symJsxSelfClosingElement:
		exactlyAfter := make(map[int]rune)
		depthAfter := make(map[int]bool)
		newLine := p.ifJsxAttributeNewLine(n, nodeChildren)
		exactlyAfter[2] = spacingChar(newLine)
		depthAfter[2] = true
		for i := 2; i < children-3; i++ {
			if int(nodeChildren[i].Symbol()) == symJsxAttribute {
				exactlyAfter[i+1] = spacingChar(newLine)
				depthAfter[i+1] = true
			}
		}
		if newLine {
			exactlyAfter[children-2] = '\n'
		}
		return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, depthAfter: depthAfter, childCount: children}

	case symJsxFragment:
		exactlyAfter := make(map[int]rune)
		if p.conf.JsxFragmentChildrenNewline {
			// After `<>` and before `</>`
			for i := 1; i < children-3; i++ {
				exactlyAfter[i+1] = '\n'
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, childCount: children, exactlyAfter: exactlyAfter}
		}

	case symJsxElement:
		exactlyAfter := make(map[int]rune)
		newLine := p.ifJsxElementNewLine(n, nodeChildren)
		p.jsxElementNewLine[n.StartByte()] = newLine
		if newLine {
			// In between children
			for i := 0; i < children-1; i++ {
				exactlyAfter[i+1] = '\n'
			}
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, childCount: children}
		}

	case symJsxOpeningElement:
		exactlyAfter := make(map[int]rune)
		depthAfter := make(map[int]bool)
		newLine := p.ifJsxAttributeNewLine(n, nodeChildren)
		if children > 3 {
			exactlyAfter[2] = spacingChar(newLine)
			depthAfter[2] = true
			for i := 2; i < children-2; i++ {
				if int(nodeChildren[i].Symbol()) == symJsxAttribute {
					exactlyAfter[i+1] = spacingChar(newLine)
					depthAfter[i+1] = true
				}
			}
		}
		if newLine {
			exactlyAfter[children-1] = '\n'
		}
		if len(exactlyAfter) > 0 {
			return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, depthAfter: depthAfter, childCount: children}
		}
	}

	if n.ChildCount() == 0 {
		p.printNodeSrc(n)
		return nil
	}

	if jsNodeTypeRequireASI[n.Type()] || (isStatement(n) && !isBlockOfContainerStatement(n)) {
		return &spacerVisitor{p: p, parentNode: n, maxSpaces: -1, childCount: children}
	}

	return p
}

func (p *prettyPrinter) ifFuncParamNewLine(n *sitter.Node, nodeChildren []*sitter.Node) bool {
	children := int(n.ChildCount())
	// If hard-coded as yes or no
	if p.conf.FuncParenNewline == 1 && p.conf.FuncParamArgumentNewline == 1 {
		return true
	}
	if p.conf.FuncParenNewline == 0 && p.conf.FuncParamArgumentNewline == 0 {
		return false
	}

	// Respect the user's current setting if cursor position is in the middle of ()
	if children >= 2 && render.CursorInsideNode(n, p.snippetBegin) {
		// The opening and closing parens are not in the same row
		if nodeChildren[0].StartPoint().Row != nodeChildren[children-1].StartPoint().Row {
			return true
		}
		// Parens are in the same line and the user has already types something in between
		// indicates that there should not be new line
		if children > 2 && p.snippetBegin > int(nodeChildren[1].StartByte()) {
			return false
		}
	}

	// Apply heuristics otherwise
	if children >= 9 {
		return true
	}
	for i := 1; i < children-1; i++ {
		child := nodeChildren[i]
		sym := int(child.Symbol())
		if isComplexNode(sym) {
			return true
		}
		// If the argument is an {} object that has more than one child in between
		if sym == symObject && child.ChildCount() > 3 {
			return true
		}
	}
	return false
}

func (p *prettyPrinter) ifObjectNewLine(n *sitter.Node, nodeChildren []*sitter.Node) bool {
	sym := int(n.Symbol())
	children := int(n.ChildCount())

	// If hard-coded as yes or no
	if p.conf.ObjectPropertyNewline == 1 && p.conf.ObjectCurlyNewline == 1 {
		return true
	}
	if p.conf.ObjectPropertyNewline == 0 && p.conf.ObjectCurlyNewline == 0 {
		return false
	}

	// If it's part of arrow-function, function or class bodies, treat is as StatementBlock
	pp := render.SafeParent(n)
	pSym := render.SafeSymbol(pp)
	if pSym == symArrowFunction || pSym == symFunctionDeclaration || pSym == symClass {
		return p.conf.StatementNewline
	}

	// Respect the user's current setting if cursor position is in the middle of it
	if children >= 2 && render.CursorInsideNode(n, p.snippetBegin) {
		// The opening and closing `{}` are not on the same row
		if nodeChildren[0].StartPoint().Row != nodeChildren[children-1].StartPoint().Row {
			return true
		}
		// `{}` are on the same line and the user has already typed something in between
		if children > 2 && p.snippetBegin > int(nodeChildren[1].StartByte()) {
			return false
		}
	}

	// Apply heuristics otherwise
	if sym == symNamedImports || sym == symExportClause {
		return false
	}
	// At least three properties
	if children >= 7 {
		return true
	}
	for i := 1; i < children-1; i++ {
		child := nodeChildren[i]
		childSym := int(child.Symbol())
		// If the property is something complicated
		if isComplexNode(childSym) {
			return true
		}
		if childSym == symPair {
			valueSym := render.SafeSymbol(child.ChildByFieldName("value"))
			if isComplexNode(valueSym) {
				return true
			}
		}
	}
	return false
}

func (p *prettyPrinter) ifArrayNewLine(n *sitter.Node, nodeChildren []*sitter.Node) bool {
	children := int(n.ChildCount())

	// If hard-coded as yes or no
	if p.conf.ArrayBracketNewline == 1 && p.conf.ArrayElementNewline == 1 {
		return true
	}
	if p.conf.ArrayBracketNewline == 0 && p.conf.ArrayElementNewline == 0 {
		return false
	}

	// Respect the user's current setting if cursor position is in the middle of it
	if children >= 2 && render.CursorInsideNode(n, p.snippetBegin) {
		// The opening and closing `[]` are not on the same row
		if nodeChildren[0].StartPoint().Row != nodeChildren[children-1].StartPoint().Row {
			return true
		}
		// `[]` are on the same line and the user has already typed something in between
		if children > 2 && p.snippetBegin > int(nodeChildren[1].StartByte()) {
			return false
		}
	}

	// Apply heuristics otherwise
	if children >= 7 {
		// At least three elements
		return true
	}
	for i := 1; i < children-1; i++ {
		child := nodeChildren[i]
		childSym := int(child.Symbol())
		// If the element is something complicated
		if isComplexNode(childSym) {
			return true
		}
	}
	return false
}

func (p *prettyPrinter) ifJsxElementNewLine(n *sitter.Node, nodeChildren []*sitter.Node) bool {
	// If hard-coded as yes or no
	if p.conf.JsxElementChildrenNewline == 1 {
		return true
	}
	if p.conf.JsxElementChildrenNewline == 0 {
		return false
	}

	children := len(nodeChildren)
	// If the opening element has attributes in new line, so is the current element
	if children > 0 {
		opening := nodeChildren[0]
		grandChildren := make([]*sitter.Node, opening.ChildCount())
		for i := 0; i < int(opening.ChildCount()); i++ {
			grandChildren[i] = opening.Child(i)
		}
		if p.ifJsxAttributeNewLine(opening, grandChildren) {
			return true
		}
	}

	// Respect the user's current setting if cursor position is in the middle of it
	if children >= 2 && render.CursorInsideNode(n, p.snippetBegin) {
		// The opening and closing tags are not in the same row
		if nodeChildren[0].StartPoint().Row != nodeChildren[children-1].StartPoint().Row {
			return true
		}
		// Opening anf closing tags are on the same line and the user has already in between
		if children > 2 && p.snippetBegin > int(nodeChildren[1].StartByte()) {
			return false
		}
	}

	var nonEmptyChildren int
	for i := 1; i < children-1; i++ {
		child := nodeChildren[i]
		sym := int(child.Symbol())
		// Has embedded jsx element, self-closing element
		if isComplexNode(sym) {
			return true
		}
		if sym == symJsxText && strings.TrimSpace(child.Content(p.src)) == "" {
			continue
		}
		nonEmptyChildren++
	}
	if nonEmptyChildren >= 2 {
		return true
	}
	return false
}

func (p *prettyPrinter) ifJsxAttributeNewLine(n *sitter.Node, nodeChildren []*sitter.Node) bool {
	// For Jsx opening element `<foo bar="baz">`
	// and Jsx self-closing element `<foo bar="baz" />`
	if p.conf.JsxAttributeNewline == 1 {
		return true
	}
	if p.conf.JsxAttributeNewline == 0 {
		return false
	}

	// Respect the user's current setting if cursor position is in the middle of it
	children := len(nodeChildren)
	sym := int(n.Symbol())
	if sym == symJsxOpeningElement && children >= 3 || sym == symJsxSelfClosingElement && children >= 4 {
		if render.CursorInsideNode(n, p.snippetBegin) {
			// The `<` and `>` are not in the same row
			if nodeChildren[0].StartPoint().Row != nodeChildren[children-1].StartPoint().Row {
				return true
			}
			// `<` and `>`are on the same line and the user has already in between
			if children > 3 && p.snippetBegin > int(nodeChildren[2].StartByte()) {
				return false
			}
		}
	}

	// Apply heuristics
	var numAttributes int
	for i := 0; i < children; i++ {
		if render.SafeSymbol(nodeChildren[i]) == symJsxAttribute {
			numAttributes++
		}
	}
	return numAttributes >= 3
}

func isComplexNode(sym int) bool {
	return sym == symJsxElement || sym == symJsxSelfClosingElement ||
		sym == symJsxFragment || sym == symCallExpression || sym == symFunction ||
		sym == symArray || sym == symObject || sym == symArrowFunction
}

func (p *prettyPrinter) checkNonBlockStatementBody(n *sitter.Node, childCount int, exactlyAfter map[int]rune) treesitter.Visitor {
	return p.checkNonBlockBodyRule(n, childCount, exactlyAfter)
}

func (p *prettyPrinter) checkNonBlockBodyRule(n *sitter.Node, childCount int, exactlyAfter map[int]rune) treesitter.Visitor {
	if isNonBlockBodyField(n, "body") {
		depthAfter := make(map[int]bool, len(exactlyAfter))
		for k := range exactlyAfter {
			depthAfter[k] = true
		}
		return &spacerVisitor{p: p, parentNode: n, exactlyAfter: exactlyAfter, depthAfter: depthAfter, childCount: childCount}
	}
	return nil
}

func spacingChar(newline bool) rune {
	if newline {
		return '\n'
	}
	return ' '
}

// inserts locations where space or newline must be added in the exactlyAfter map between each
// element separated by a comma in the children of n. Whether the spacing is added before or
// after the comma is controlled by beforeComma. It returns the map received as argument, possibly
// with additional pairs inserted.
func betweenCommaSepChildren(n *sitter.Node, childCount int, nodeChildren []*sitter.Node, beforeComma bool, spaceChar rune, exactlyAfter map[int]rune) map[int]rune {
	if n == nil {
		return exactlyAfter
	}

	var lastCommaIx int
	// ignore first and last, as those are the list "wrappers" (e.g. {...}, [...] or (...))
	for i := 1; i < childCount-1; i++ {
		child := nodeChildren[i]
		if child != nil && child.Type() == "," {
			lastCommaIx = i
			continue
		}
		if lastCommaIx > 0 {
			if beforeComma {
				exactlyAfter[lastCommaIx] = spaceChar
			} else {
				exactlyAfter[lastCommaIx+1] = spaceChar
			}
			lastCommaIx = 0
		}
	}
	return exactlyAfter
}

func isNonBlockBodyField(n *sitter.Node, field string) bool {
	if n == nil {
		return false
	}

	body := n.ChildByFieldName(field)
	if body == nil || int(body.Symbol()) == symStatementBlock {
		return false
	}
	return true
}

func isStatementOfForLoopHeader(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	if !isStatement(n) {
		return false
	}
	p := n.Parent()
	if p == nil {
		return false
	}
	if sym := int(p.Symbol()); sym == symForStatement {
		return true
	}
	return false
}

func isBlockOfContainerStatement(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	if sym := int(n.Symbol()); sym != symStatementBlock {
		return false
	}

	p := n.Parent()
	if p == nil {
		return false
	}
	switch sym := int(p.Symbol()); sym {
	case symForInStatement, symForOfStatement, symForStatement, symWhileStatement, symDoStatement,
		symTryStatement, symIfStatement, symSwitchStatement:
		return true
	case symCatchClause, symFinallyClause:
		return true
	}
	return true
}

func isASISpecialCaseDecl(n *sitter.Node) bool {
	switch render.SafeSymbol(n) {
	case symFunctionDeclaration, symGeneratorFunctionDeclaration, symClassDeclaration:
		return true
	default:
		return false
	}
}

func isStatement(n *sitter.Node) bool {
	switch render.SafeSymbol(n) {
	case symExportStatement, symImportStatement, symDebuggerStatement, symExpressionStatement,
		symStatementBlock, symIfStatement, symSwitchStatement, symForStatement, symForInStatement,
		symForOfStatement, symWhileStatement, symDoStatement, symTryStatement, symWithStatement,
		symBreakStatement, symContinueStatement, symReturnStatement, symThrowStatement, symEmptyStatement,
		symLabeledStatement, symSwitchCase, symSwitchDefault:

		return true

	default:
		return isDeclaration(n)
	}
}

func isDeclaration(n *sitter.Node) bool {
	switch render.SafeSymbol(n) {
	case symFunctionDeclaration, symGeneratorFunctionDeclaration, symClassDeclaration,
		symLexicalDeclaration, symVariableDeclaration:

		return true

	default:
		return false
	}
}

func isKeyword(n *sitter.Node, src []byte) bool {
	if n == nil {
		return false
	}
	typ, content := n.Type(), n.Content(src)
	if typ != content || strings.Contains(typ, "identifier") {
		return false
	}
	if len(typ) < 2 {
		return false
	}
	first := typ[0]
	return 'a' <= first && first <= 'z'
}

func isLiteral(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	sym := int(n.Symbol())
	return sym == symString ||
		sym == symTemplateString ||
		sym == symRegex ||
		sym == symNumber ||
		sym == anonSymDquote ||
		sym == anonSymSquote ||
		sym == anonSymBquote // because strings are decomposed, if n is a quote, it is the same as a string/template
}

func isIdentifierLike(n *sitter.Node, src []byte) bool {
	if n == nil {
		return false
	}

	sym := int(n.Symbol())
	if sym == symIdentifier ||
		sym == symJsxIdentifier ||
		sym == aliasSymShorthandPropertyIdentifier ||
		sym == aliasSymPropertyIdentifier ||
		sym == aliasSymStatementIdentifier ||
		sym == symNumber ||
		sym == symRegex {
		return true
	}
	return isKeyword(n, src)
}

var opCharConflicts = map[byte]bool{
	'.': true,
	'+': true,
	'-': true,
	'*': true,
	'/': true,
	'=': true,
	'>': true,
	'<': true,
	'&': true,
	'|': true,
}

var doNotCheckingForConflicts = map[int]bool{
	symJsxClosingElement:     true,
	symJsxFragment:           true,
	symJsxSelfClosingElement: true,
}

func hasOperatorConflict(src []byte, left, right *sitter.Node) bool {
	if left == nil || right == nil || len(src) == 0 {
		return false
	}

	// special-case: calling methods on regex literals is fine, e.g.
	// /abc\d/.test(someString)
	if int(left.Symbol()) == symRegex && int(right.Symbol()) == anonSymDot {
		return false
	}

	lc, rc := left.Content(src), right.Content(src)
	if len(lc) == 0 || len(rc) == 0 {
		return false
	}

	// Special case for JSX stuff like `</div>`
	if doNotCheckingForConflicts[int(left.Parent().Symbol())] && doNotCheckingForConflicts[int(right.Parent().Symbol())] {
		return false
	}

	// conflict if last character of left can conflict and first character of right can conflict
	return opCharConflicts[lc[len(lc)-1]] && opCharConflicts[rc[0]]
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
	exactlyAfter map[int]rune

	// depthAfter increments p.depth after those nodes, and decrements it immediately
	// after walking that node. This is used to handle the non-block statements like if and
	// while.
	depthAfter map[int]bool

	// 0=no limit, < 0 = no space inserted (unless exactlyAfter is set), otherwise max number
	// of spaceChar to insert between child nodes.
	maxSpaces int

	seen       int
	childCount int
}

func (s *spacerVisitor) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		if jsNodeTypeRequireASI[s.parentNode.Type()] {
			s.p.printAddedSemi()
		}
		if s.p.conf.StatementNewline && isStatement(s.parentNode) && !isBlockOfContainerStatement(s.parentNode) && !isStatementOfForLoopHeader(s.parentNode) {
			s.p.print("\n", false)
		}
		if isASISpecialCaseDecl(s.parentNode) {
			s.p.asiFromFuncOrClassDeclEndLine = int(s.parentNode.EndPoint().Row)
		}
		if render.SafeSymbol(s.parentNode) == symERROR && s.p.lastPrintedNode != nil {
			// we're done visiting the children of an ERROR node, print any remaining content
			// that was not covered by children
			if s.p.lastPrintedNode.EndByte() < s.parentNode.EndByte() {
				startb := int(s.p.lastPrintedNode.EndByte())
				endb := int(s.parentNode.EndByte())
				content := string(s.p.src[startb:endb])
				lastSym := int(s.p.lastPrintedNode.Symbol())
				if lastSym == anonSymSquote || lastSym == anonSymDquote || lastSym == anonSymBquote {
					s.p.recordMappingForString(startb, endb, content)
				} else {
					s.p.print(content, false)
					s.p.recordMapping(startb, endb, content)
				}
			}
		}

		return nil
	}

	if len(s.exactlyAfter) == 0 {
		// insert the spaceChar between each child
		if s.seen > 0 && s.seen < s.childCount && (s.maxSpaces == 0 || s.seen <= s.maxSpaces) {
			s.p.print(string(s.spaceChar), false)
		}
	} else if ch := s.exactlyAfter[s.seen]; ch > 0 {
		s.p.print(string(ch), false)
	}

	// special-case for the class body: if StatementNewline is false, must insert a semicolon between
	// each body's member so that it can be parsed properly again.
	if !s.p.conf.StatementNewline && int(s.parentNode.Symbol()) == symClassBody {
		// the class body starts with "{" and ends with "}"
		if s.seen > 1 && s.seen < s.childCount {
			s.p.printAddedSemi()
		}
	}

	if s.depthAfter[s.seen] {
		s.p.recordDepthIncrease()
	}

	s.seen++
	treesitter.Walk(s.p, n)

	if s.depthAfter[s.seen-1] {
		s.p.recordDepthDecrease()
	}
	return nil
}

func (p *prettyPrinter) printNodeSrc(n *sitter.Node) {
	content := n.Content(p.src)
	sym := int(n.Symbol())
	parent := render.SafeParent(n)
	parentSym := render.SafeSymbol(parent)

	if (content == ")" && p.conf.SpaceInParens) ||
		// JSX expression {} should be compact
		(content == "{" && p.conf.SpaceBeforeBlocks &&
			render.SafeSymbol(parent) != symJsxExpression &&
			p.lastPrintedNode != nil &&
			p.lastPrintedNode.Content(p.src) != "(") ||
		(content == "," && p.conf.CommaSpacingBefore) {
		p.print(" ", false)
	}

	if isIdentifierLike(p.lastPrintedNode, p.src) && isIdentifierLike(n, p.src) {
		p.print(" ", false)
	} else if hasOperatorConflict(p.src, p.lastPrintedNode, n) {
		p.print(" ", false)
	} else if p.conf.KeywordSpacingBefore && isKeyword(n, p.src) && p.lastPrintedNode != nil {
		if last := p.lastPrintedNode.Content(p.src); last == ")" || last == "}" {
			p.print(" ", false)
		} else if isLiteral(p.lastPrintedNode) {
			p.print(" ", false)
		} else if render.SafeSymbol(p.lastPrintedNode) == anonSymStar {
			// e.g. import * as when printing "as"
			p.print(" ", false)
		}
	} else if p.conf.KeywordSpacingAfter && p.lastPrintedNode != nil && isKeyword(p.lastPrintedNode, p.src) {
		if content == "(" && render.SafeSymbol(render.SafeParent(n)) != symCallExpression {
			// Space after keyword in `while (x)` but not `super(x)`
			p.print(" ", false)
		} else if content == "{" {
			p.print(" ", false)
		} else if isLiteral(n) {
			p.print(" ", false)
		} else if sym == anonSymStar {
			// e.g. import * as when printing "*"
			p.print(" ", false)
		}
	}

	if content == "}" || content == "]" || content == ")" {
		p.recordDepthDecrease()
	}

	// Before jsx closing element `</div>`
	if content == "<" && parentSym == symJsxClosingElement {
		p.recordDepthDecrease()
	}

	// Before jsx fragment ending `</>`
	if content == "<" && parentSym == symJsxFragment &&
		render.SafeEqual(render.SafeChild(parent, int(parent.ChildCount())-3), n) {
		p.recordDepthDecrease()
	}

	if sym == symComment && strings.HasPrefix(content, "//") {
		// for single-line comments, add a space before the comment
		p.print(" ", false)
	}

	if sym == symJsxText && parentSym == symJsxElement {
		if p.jsxElementNewLine[parent.StartByte()] {
			content = strings.TrimSpace(content)
		} else {
			if render.SafeEqual(render.SafeChild(parent, 1), n) {
				content = strings.TrimLeft(content, " ")
			}
			if render.SafeEqual(render.SafeChild(parent, int(parent.ChildCount())-2), n) {
				content = strings.TrimRight(content, " ")
			}
		}
	}

	if sym == symJsxText && parentSym == symJsxFragment {
		if p.conf.JsxFragmentChildrenNewline {
			content = strings.TrimSpace(content)
		} else {
			if render.SafeEqual(render.SafeChild(parent, 2), n) {
				content = strings.TrimLeft(content, " ")
			}
			if render.SafeEqual(render.SafeChild(parent, int(parent.ChildCount())-4), n) {
				content = strings.TrimRight(content, " ")
			}
		}
	}

	p.print(content, false)
	startb, endb := int(n.StartByte()), int(n.EndByte())
	p.recordMapping(startb, endb, content)

	p.lastPrintWasAddedSemi = false

	if (sym == symComment && strings.HasPrefix(content, "//")) || sym == symHashBangLine {
		// must add a newline after write, this is a single-line comment
		p.print("\n", false)
	} else if sym == symComment && strings.HasPrefix(content, "/*") {
		// for multi-line /* ... */ comment, if the comment starts on its own line (is not at the end or
		// in-between code tokens), add a newline after the comment. This is to avoid ugly formatting like
		//   /* some comment */const x = y;
		if p.lastPrintedNode == nil || p.lastPrintedNode.EndPoint().Row < n.StartPoint().Row {
			p.print("\n", false)
		}
	}

	if (content == "(" && p.conf.SpaceInParens) ||
		(content == "," && p.conf.CommaSpacingAfter) {
		p.print(" ", false)
	}

	if content == "{" || content == "[" || content == "(" {
		p.recordDepthIncrease()
	}

	// After jsx opening element `<div>`
	if n.Content(p.src) == ">" && parentSym == symJsxOpeningElement {
		p.recordDepthIncrease()
	}

	// After Jsx fragment `<>`
	if n.Content(p.src) == ">" && parentSym == symJsxFragment && render.SafeEqual(render.SafeChild(parent, 1), n) {
		p.recordDepthIncrease()
	}

	p.lastPrintedNode = n
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

func (p *prettyPrinter) recordMappingForString(startb, endb int, content string) {
	parts := SplitString(content)
	current := startb
	for _, part := range parts {
		p.print(part, true)
		p.recordMapping(current, current+len(part), part)
		current += len(part)
	}
}

// The following two methods record the intention of increasing/decreasing
// depth - the actual increase or decrease is only done when a newline is
// encountered in print, and then the number of increase vs decrease encountered
// is checked to determine if depth did increase or decrease.
func (p *prettyPrinter) recordDepthIncrease() {
	if p.lockedDepthDecr {
		p.lockedDepthDecr = false
		p.pendingDepthChange = 1
		return
	}
	p.pendingDepthChange++
}
func (p *prettyPrinter) recordDepthDecrease() {
	if p.lockedDepthDecr {
		return
	}
	p.pendingDepthChange--
}

func (p *prettyPrinter) printAddedSemi() {
	if p.lastPrintWasAddedSemi || render.SafeSymbol(p.lastPrintedNode) == anonSymSemi {
		return
	}
	if p.conf.Semicolon {
		p.print(";", false)
		p.lastPrintWasAddedSemi = true
	} else {
		p.pendingSemi = "\n"
		p.lockedDepthDecr = false
	}
}

func (p *prettyPrinter) print(s string, unaltered bool) {
	if p.err != nil || s == "" {
		return
	}

	if strings.TrimSpace(s) == "" && !unaltered {
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

	if p.pendingSemi != "" {
		// we have a newline pending to play the role of semicolon, add it only if there is not already
		// a newline to be written in p.pendingWrite, and if the non-whitespace write is not a semicolon
		// nor a closing brace.
		if !strings.Contains(p.pendingWrite, p.pendingSemi) && s != ";" && s != "}" {
			p.pendingWrite += p.pendingSemi
		}
		p.pendingSemi = ""
	}

	// if the pending write is a newline, the indent must also be written
	if p.pendingWrite == "\n" {
		depthChange := p.pendingDepthChange
		p.pendingDepthChange = 0

		switch {
		case depthChange > 0:
			p.depth++

		case depthChange < 0:
			p.depth += depthChange
			if p.depth < 0 {
				p.depth = 0
			}
			p.lockedDepthDecr = true
		}

		if p.conf.Indent < 0 {
			p.pendingWrite += strings.Repeat(strings.Repeat("\t", -p.conf.Indent), p.depth)
		} else if p.conf.Indent > 0 {
			p.pendingWrite += strings.Repeat(strings.Repeat(" ", p.conf.Indent), p.depth)
		}
	}

	var n int
	n, p.err = fmt.Fprint(p.w, p.pendingWrite+s)
	p.pos += n
	p.pendingWrite = ""
}
