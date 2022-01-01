package pigeon

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/ast"
)

// TODO(mna): refactor so that epytext and numpydoc share indent-stack
// management code (internal package shared by both, with tests).

// DefinitionListSectionsKey is the key value for the set of section names
// that must be parsed as definition lists. This is stored in the global
// state of the parser.
const DefinitionListSectionsKey = "definitionListSections"

const (
	indentStackSize          = 64
	indentStackKey           = "indentStack"
	indentStackIndexKey      = "indentStackIndex"
	directiveIndentLevelKey  = "directiveIndentLevel"
	doctestIndentLevelKey    = "doctestIndentLevel"
	definitionIndentLevelKey = "definitionIndentLevel"
	sectionNameKey           = "sectionName"
)

// type returned from actions where indentation matters.
type indentedText struct {
	indent int
	text   string
}

// use an array, not a slice, so the cloning of state in pigeon
// is fast and automatic.
type indentStack [indentStackSize]int

func initState(c *current) error {
	c.state[indentStackKey] = indentStack{}
	c.state[indentStackIndexKey] = 1 // 0 is the automatically pushed indentof 0
	c.state[directiveIndentLevelKey] = 0
	c.state[doctestIndentLevelKey] = 0
	c.state[definitionIndentLevelKey] = 0
	c.state[sectionNameKey] = ""
	return nil
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

func grammarAction(c *current, items []interface{}) (*ast.Doc, error) {
	nodes := make([]ast.Node, len(items))
	for i, it := range items {
		nodes[i] = it.(ast.Node)
	}
	return &ast.Doc{Content: nodes}, nil
}

func underlineHeaderPredicate(c *current, header, underline indentedText) (bool, error) {
	return header.indent == underline.indent &&
		len(header.text) == len(underline.text), nil
}

func underlineHeaderState(c *current, header indentedText) error {
	c.state[sectionNameKey] = header.text
	return nil
}

func underlineSectionAction(c *current, header indentedText, content []ast.Node) (*ast.Section, error) {
	return &ast.Section{
		Header:  header.text,
		Content: content,
	}, nil
}

func underlineLineAction(c *current, chars []interface{}) (indentedText, error) {
	return indentedText{
		indent: currentIndent(c),
		text:   stringFromSliceOfStrings(chars, ""),
	}, nil
}

func underlineSectionContentDefinitionPredicate(c *current) (bool, error) {
	section := c.state[sectionNameKey].(string)
	// make the lookup check case-insensitive
	section = strings.TrimSpace(strings.ToLower(section))
	defSections := c.globalStore[DefinitionListSectionsKey].(map[string]bool)
	return defSections[section], nil
}

func underlineSectionContentAction(c *current, content []interface{}) ([]ast.Node, error) {
	nodes := make([]ast.Node, len(content))
	for i, v := range content {
		ar := toIfaceSlice(v)
		if len(ar) != 2 {
			panic("expected len(ar) == 2")
		}
		// [0] == !UnderlineHeader, [1] == ( Doctest / Directive / ... )
		nodes[i] = ar[1].(ast.Node)
	}
	return nodes, nil
}

func paragraphAction(c *current, lines []interface{}) (*ast.Paragraph, error) {
	return &ast.Paragraph{
		Content: []ast.Node{
			ast.Text(stringFromSliceOfStrings(lines, " ")),
		},
	}, nil
}

func directiveAction(c *current, dir *ast.Directive, content *ast.Paragraph) (*ast.Directive, error) {
	dir.Content = []ast.Node{content}
	return dir, nil
}

func directiveLeadState(c *current) error {
	c.state[directiveIndentLevelKey] = currentIndent(c)
	return nil
}

func semicolonDirectiveLeadAction(c *current, name []interface{}) (*ast.Directive, error) {
	return &ast.Directive{
		Name: stringFromSliceOfStrings(name, ""),
	}, nil
}

func bracketDirectiveLeadAction(c *current, name []interface{}) (*ast.Directive, error) {
	if len(name) != 3 {
		panic("expected len(name) == 3")
	}

	// [0] == "[", [1] == DiretiveNameChars+, [2] == "]"
	var buf bytes.Buffer
	buf.Write(name[0].([]byte))
	chars := toIfaceSlice(name[1])
	writeStringFromSliceOfStrings(&buf, chars, "")
	buf.Write(name[2].([]byte))

	return &ast.Directive{
		Name: buf.String(),
	}, nil
}

func directiveContentPredicate(c *current, line indentedText) (bool, error) {
	directiveIndent := c.state[directiveIndentLevelKey].(int)
	return line.indent > directiveIndent, nil
}

func directiveContentAction(c *current, first string, rest []interface{}) (*ast.Paragraph, error) {
	var buf bytes.Buffer
	buf.WriteString(first)
	writeStringFromSliceOfSlices(&buf, rest, 2, 0, " ")
	return &ast.Paragraph{
		Content: []ast.Node{
			ast.Text(buf.String()),
		},
	}, nil
}

func doctestFirstLineState(c *current, first indentedText) error {
	c.state[doctestIndentLevelKey] = first.indent
	return nil
}

func doctestNextLinePredicate(c *current, line indentedText) (bool, error) {
	doctestIndent := c.state[doctestIndentLevelKey].(int)
	return line.indent == doctestIndent, nil
}

func doctestLinesAction(c *current, first indentedText, rest []interface{}) (*ast.Doctest, error) {
	var buf bytes.Buffer
	buf.WriteString(first.text)
	writeStringFromSliceOfSlices(&buf, rest, 2, 0, "\n")
	return &ast.Doctest{
		Text: buf.String(),
	}, nil
}

func firstDoctestLineAction(c *current, text []interface{}) (indentedText, error) {
	if len(text) != 3 {
		panic("expected len(text) == 3")
	}

	var buf bytes.Buffer
	// [0] == ">>>", [1] == Whitespace, [2] == ( !EOL . )*
	buf.Write(text[0].([]byte))
	buf.Write(text[1].([]byte))
	chars := toIfaceSlice(text[2])
	writeStringFromSliceOfSlices(&buf, chars, 2, 1, "")

	return indentedText{
		indent: currentIndent(c),
		text:   buf.String(),
	}, nil
}

func definitionAction(c *current, first *ast.Definition, content []ast.Node) (*ast.Definition, error) {
	first.Content = content
	return first, nil
}

func definitionContentAction(c *current, content []interface{}) ([]ast.Node, error) {
	nodes := make([]ast.Node, len(content))
	for i, v := range content {
		nodes[i] = v.(ast.Node)
	}
	return nodes, nil
}

func definitionParagraphPredicate(c *current, line indentedText) (bool, error) {
	defIndent := c.state[definitionIndentLevelKey].(int)
	return line.indent > defIndent, nil
}

func definitionParagraphAction(c *current, lines []interface{}) (*ast.Paragraph, error) {
	return &ast.Paragraph{
		Content: []ast.Node{
			ast.Text(stringFromSliceOfSlices(lines, 2, 0, " ")),
		},
	}, nil
}

func firstDefinitionLineState(c *current) error {
	c.state[definitionIndentLevelKey] = currentIndent(c)
	return nil
}

func firstDefinitionLineAction(c *current, subject, typ string) (*ast.Definition, error) {
	var typeNodes []ast.Node
	if typ != "" {
		typeNodes = []ast.Node{ast.Text(typ)}
	}
	return &ast.Definition{
		Subject: []ast.Node{ast.Text(subject)},
		Type:    typeNodes,
	}, nil
}

func definitionTypeAction(c *current, text []interface{}) (string, error) {
	return stringFromSliceOfSlices(text, 2, 1, ""), nil
}

func restOfLineAction(c *current, text []interface{}) (string, error) {
	return stringFromSliceOfSlices(text, 2, 1, ""), nil
}

func nonBlankLineAction(c *current, text []interface{}) (indentedText, error) {
	return indentedText{
		indent: currentIndent(c),
		text:   stringFromSliceOfSlices(text, 2, 1, ""),
	}, nil
}

// converts common PEG constructs such as `.+` to a
// string. It supports []byte and indentedText. If sep is
// not empty, it is inserted between each string part.
func stringFromSliceOfStrings(slice []interface{}, sep string) string {
	var buf bytes.Buffer
	writeStringFromSliceOfStrings(&buf, slice, sep)
	return buf.String()
}

// same as stringFromSliceOfStrings except that it writes to buf instead
// of returning the string. It returns the number of bytes written.
func writeStringFromSliceOfStrings(buf *bytes.Buffer, slice []interface{}, sep string) int {
	var cnt int
	for _, v := range slice {
		if sep != "" && buf.Len() > 0 {
			n, _ := buf.WriteString(sep)
			cnt += n
		}
		switch str := v.(type) {
		case []byte:
			n, _ := buf.Write(str)
			cnt += n
		case indentedText:
			n, _ := buf.WriteString(str.text)
			cnt += n
		default:
			panic(fmt.Sprintf("writeStringFromSliceOfStrings: unexpected type: %T", str))
		}
	}
	return cnt
}

// converts common PEG constructs such as `( !EOL . )*` to a
// string. Converts each value of slice to a slice itself, validates
// that it has the expectedLen, extracts the string value at stringIndex,
// and writes that value (it supports []byte and indentedText). If sep is
// not empty, it is inserted between each string part.
func stringFromSliceOfSlices(slice []interface{}, expectedLen, stringIndex int, sep string) string {
	var buf bytes.Buffer
	writeStringFromSliceOfSlices(&buf, slice, expectedLen, stringIndex, sep)
	return buf.String()
}

// same as stringFromSliceOfSlices except that it writes to buf instead
// of returning the string. It returns the number of bytes written.
func writeStringFromSliceOfSlices(buf *bytes.Buffer, slice []interface{}, expectedLen, stringIndex int, sep string) int {
	var cnt int
	for _, v := range slice {
		ar := toIfaceSlice(v)
		if len(ar) != expectedLen {
			panic("expected len(ar) == " + strconv.Itoa(expectedLen))
		}
		if sep != "" && buf.Len() > 0 {
			n, _ := buf.WriteString(sep)
			cnt += n
		}
		switch str := ar[stringIndex].(type) {
		case []byte:
			n, _ := buf.Write(str)
			cnt += n
		case indentedText:
			n, _ := buf.WriteString(str.text)
			cnt += n
		default:
			panic(fmt.Sprintf("writeStringFromSliceOfSlices: unexpected type: %T", str))
		}
	}
	return cnt
}

// toIfaceSlice is a helper function for the PEG grammar parser. It converts
// v to a slice of empty interfaces.
func toIfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}
