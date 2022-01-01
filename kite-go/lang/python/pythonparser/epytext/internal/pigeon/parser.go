package pigeon

import (
	"bytes"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
)

const (
	indentStackSize           = 64
	indentStackKey            = "indentStack"
	indentStackIndexKey       = "indentStackIndex"
	currentParagraphIndentKey = "currentParagraphIndent"
	currentDoctestIndentKey   = "currentDoctestIndent"
	currentLiteralIndentKey   = "currentLiteralIndent"
)

// use an array, not a slice, so the cloning of state in pigeon
// is fast and automatic.
type indentStack [indentStackSize]int

func initState(c *current) error {
	c.state[indentStackKey] = indentStack{}
	c.state[indentStackIndexKey] = 1 // 0 is the automatically pushed indentof 0
	return nil
}

func grammarAction(c *current, vs []interface{}) (*ast.DocBlock, error) {
	blocks := toIfaceBlocks(vs)
	doc := toHierarchicalAST(blocks)
	return doc, nil
}

func indentState(c *current, indent []interface{}) error {
	index := c.state[indentStackIndexKey].(int)
	stack := c.state[indentStackKey].(indentStack)
	currentIndent := stack[index-1]

	// compute the new indent
	newIndent := 0
	for _, v := range indent {
		b := v.([]byte)[0]
		switch b {
		case ' ':
			newIndent++
		case '\t':
			tab := 8 - (newIndent % 8)
			newIndent += tab
		}
	}

	// if it is greater than currentIndex, push it onto the stack,
	// if it is less, pop until it is >=, if it is the same, nothing
	// to do.
	switch {
	case newIndent > currentIndent:
		stack[index] = newIndent
		index++

	case newIndent < currentIndent:
		// currentIndent is at index-1, so start loop at index-2
		for i := index - 2; i >= 0; i-- {
			if newIndent > stack[i] {
				// add the new indent after this entry
				index = i + 1
				stack[index] = newIndent
				index++
				break
			}
			if newIndent == stack[i] {
				// set the index to the next slot on the stack
				index = i + 1
				break
			}
		}
	}

	if index >= indentStackSize {
		panic("indent stack overflow")
	}

	c.state[indentStackIndexKey] = index
	c.state[indentStackKey] = stack
	return nil
}

func currentIndent(c *current) int {
	index := c.state[indentStackIndexKey].(int)
	stack := c.state[indentStackKey].(indentStack)
	return stack[index-1]
}

func literalIntroPredicate(c *current) (bool, error) {
	lIndent, ok := c.state[currentLiteralIndentKey].(int)
	return ok && lIndent >= 0, nil
}

func literalLinePredicate(c *current, line plainText) (bool, error) {
	lIndent := c.state[currentLiteralIndentKey].(int)
	return line.i > lIndent, nil
}

func literalAction(c *current, lines []interface{}) (literal, error) {
	lIndent := c.state[currentLiteralIndentKey].(int)

	var buf bytes.Buffer
	for i, v := range lines {
		if i > 0 {
			buf.WriteByte('\n')
		}

		// in blankLineNilAr:
		// - index 0 is a slice of `blank` lines
		// - index 1 is the `line` (plainText)
		// - index 2 is `nil` (&{})
		blankLineNilAr := toIfaceSlice(v)

		// count the number of blank lines to insert
		blankLines := len(toIfaceSlice(blankLineNilAr[0]))
		if blankLines > 0 {
			buf.WriteString(strings.Repeat("\n", blankLines))
		}

		// add the non-blank line
		line := blankLineNilAr[1].(plainText)
		// insert spaces for the extra indent
		space := strings.Repeat(" ", line.i-lIndent)
		buf.WriteString(space + line.t)
	}
	return makeLiteral(lIndent, buf.String()), nil
}

func literalPostState(c *current, lit literal) error {
	delete(c.state, currentLiteralIndentKey)
	return nil
}

func sectionMatchPredicate(c *current, header, underline plainText) (bool, error) {
	if header.i != underline.i {
		return false, nil
	}
	return len(header.t) == len(underline.t), nil
}

func sectionAction(c *current, header, underline plainText) (section, error) {
	level := underline.t[0]
	return makeSection(header.indent(), header.text(), level), nil
}

func sectionUnderlineAction(c *current) (plainText, error) {
	return makePlainText(currentIndent(c), string(c.text)), nil
}

func listAction(c *current, bullet string, text []interface{}, hasBlank bool) (list, error) {
	var buf bytes.Buffer

	if len(text) > 0 {
		// index 0 == Whitespace, index 1 == []interface{}
		postWhitespaceAr := toIfaceSlice(text[1])
		for _, v := range postWhitespaceAr {
			nilAndCharAr := toIfaceSlice(v)
			// index 0 is nil (!EOL), index 1 is the char (.) as a []byte
			buf.Write(nilAndCharAr[1].([]byte))
		}
	}

	inlineP := strings.TrimLeft(buf.String(), " \t")
	canMergeP := len(inlineP) > 0 && !hasBlank
	l := makeList(currentIndent(c), bullet, inlineP, canMergeP, false)
	l = detectLiteralIntroduction(c, l).(list)
	return l, nil
}

func listPostState(c *current, l list) error {
	if l.litIntro {
		c.state[currentLiteralIndentKey] = l.i
	}
	return nil
}

func fieldAction(c *current, tag fieldTag, text []interface{}, hasBlank bool) (field, error) {
	var buf bytes.Buffer
	for _, v := range text {
		nilAndCharAr := toIfaceSlice(v)
		// index 0 is nil (!EOL), index 1 is the char (.) as a []byte
		buf.Write(nilAndCharAr[1].([]byte))
	}

	inlineP := strings.Trim(buf.String(), " \t")
	canMergeP := len(inlineP) > 0 && !hasBlank
	f := makeField(currentIndent(c), tag.name, tag.arg, inlineP, canMergeP, false)
	f = detectLiteralIntroduction(c, f).(field)
	return f, nil
}

func fieldTagAction(c *current, field string, rest []interface{}) (fieldTag, error) {
	var arg string
	if len(rest) > 0 {
		arg = rest[1].(string)
	}
	return fieldTag{field, arg}, nil
}

func fieldPostState(c *current, f field) error {
	if f.litIntro {
		c.state[currentLiteralIndentKey] = f.i
	}
	return nil
}

func paragraphFirstLineState(c *current, line plainText) error {
	c.state[currentParagraphIndentKey] = line.i
	return nil
}

func paragraphNextLinePredicate(c *current, line plainText) (bool, error) {
	pIndent := c.state[currentParagraphIndentKey].(int)
	return line.i == pIndent, nil
}

func paragraphAction(c *current, first plainText, rest []interface{}) (paragraph, error) {
	var buf bytes.Buffer

	buf.WriteString(first.t)
	for _, v := range rest {
		// v is []interface{} with v[0] == line, v[1] == nil (predicate)
		lineAndNilAr := toIfaceSlice(v)
		buf.WriteByte('\n')
		line := lineAndNilAr[0].(plainText).t
		buf.WriteString(line)
	}
	p := makeParagraph(first.i, buf.String(), false)
	p = detectLiteralIntroduction(c, p).(paragraph)
	return p, nil
}

func paragraphPostState(c *current, p paragraph) error {
	delete(c.state, currentParagraphIndentKey)
	if p.litIntro {
		c.state[currentLiteralIndentKey] = p.i
	}
	return nil
}

func detectLiteralIntroduction(c *current, b block) block {
	check := func(t string) (string, bool) {
		ok := false
		litIndex := strings.LastIndex(t, "::")
		if litIndex > -1 && strings.HasSuffix(strings.TrimSpace(t), "::") {
			t = t[:litIndex+1]
			ok = true
		}
		return t, ok
	}

	switch b := b.(type) {
	case paragraph:
		t, ok := check(b.t)
		if ok {
			b.litIntro = true
			b.t = t
		}
		return b

	case list:
		if b.inlineP != "" {
			t, ok := check(b.inlineP)
			if ok {
				b.litIntro = true
				b.inlineP = t
			}
		}
		return b

	case field:
		if b.inlineP != "" {
			t, ok := check(b.inlineP)
			if ok {
				b.litIntro = true
				b.inlineP = t
			}
		}
		return b

	default:
		return b
	}
}

func doctestFirstLineState(c *current, line plainText) error {
	c.state[currentDoctestIndentKey] = line.i
	return nil
}

func doctestNextLinePredicate(c *current, line plainText) (bool, error) {
	pIndent := c.state[currentDoctestIndentKey].(int)
	return line.i == pIndent, nil
}

func doctestLinesAction(c *current, first plainText, rest []interface{}) (doctest, error) {
	var buf bytes.Buffer

	buf.WriteString(first.t)
	for _, v := range rest {
		// v is []interface{} with v[0] == line, v[1] == nil (predicate)
		lineAndNilAr := toIfaceSlice(v)
		buf.WriteByte('\n')
		buf.WriteString(lineAndNilAr[0].(plainText).t)
	}
	return makeDoctest(first.i, buf.String()), nil
}

func doctestPostState(c *current, doc doctest) error {
	delete(c.state, currentDoctestIndentKey)
	return nil
}

func firstDoctestLineAction(c *current, text []interface{}) (plainText, error) {
	var buf bytes.Buffer

	// 0: ">>>"
	// 1: Whitespace
	// 2: []interface{} for rest
	//   0: nil
	//   1: char
	buf.Write(text[0].([]byte))
	buf.Write(text[1].([]byte))
	rest := toIfaceSlice(text[2])
	for _, v := range rest {
		nilAndCharAr := toIfaceSlice(v)
		buf.Write(nilAndCharAr[1].([]byte))
	}
	return makePlainText(currentIndent(c), buf.String()), nil
}

func nonBlankLineAction(c *current, text []interface{}) (plainText, error) {
	var buf bytes.Buffer
	for _, v := range text {
		nilAndCharAr := toIfaceSlice(v)
		// index 0 is nil (!EOL), index 1 is the char (.) as a []byte
		buf.Write(nilAndCharAr[1].([]byte))
	}
	return makePlainText(currentIndent(c), buf.String()), nil
}

// toIfaceSlice is a helper function for the PEG grammar parser. It converts
// v to a slice of empty interfaces.
func toIfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}

func toIfaceBlocks(vs []interface{}) []block {
	blocks := make([]block, len(vs))
	for i, v := range vs {
		blocks[i] = v.(block)
	}
	return blocks
}
