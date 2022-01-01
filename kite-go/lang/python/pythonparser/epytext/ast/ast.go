// Package ast implements the Abstract Syntax Tree.
package ast

// TODO: potentially move the package up if it is used by more than
// just epytext.

// NOTE: any new AST node created here must be added to printer.Visit
// (print.go) and to appendNodes (../internal/pigeon/parser_ast.go).

// Node is an AST node.
type Node interface {
	node()
}

// NestingNode is a Node that can contain nested nodes.
type NestingNode interface {
	Node

	// for AST walk/transform
	children() []Node
	setChildren([]Node)
}

// LeafNode is a Node that cannot contain nested node.
type LeafNode interface {
	Node
	Text() string
}

// BasicMarkup represents a basic markup node. Type
// indicates the kind of markup. E.g.
// 'I{Italicized text}'.
type BasicMarkup struct {
	Type  Markup
	Nodes []Node
}

func (m *BasicMarkup) node()                       {}
func (m *BasicMarkup) children() []Node            { return m.Nodes }
func (m *BasicMarkup) setChildren(children []Node) { m.Nodes = children }

// Markup is the type of basic markup.
type Markup byte

// List of valid inline markup.
const (
	I Markup = 'I' // Italics
	B Markup = 'B' // Bold
	C Markup = 'C' // Code
	M Markup = 'M' // Math
	X Markup = 'X' // Indexed term

	// NOTE: the Escape markup is not present, because it is processed
	// while parsing the inline markup and the resulting text is stored
	// in the Text node, with the E{} markup discarded.
)

// URLMarkup indicates that the nested nodes link to the URL.
// E.g.: 'U{www.python.org}' or 'U{The epydoc
// homepage<http://epydoc.sourceforge.net>}'.
type URLMarkup struct {
	URL   string
	Nodes []Node
}

func (m *URLMarkup) node()                       {}
func (m *URLMarkup) children() []Node            { return m.Nodes }
func (m *URLMarkup) setChildren(children []Node) { m.Nodes = children }

// CrossRefMarkup indicates that the nested nodes link to the
// specified Python object. E.g.:
// 'L{search<re.search>}'.
type CrossRefMarkup struct {
	Object string
	Nodes  []Node
}

func (m *CrossRefMarkup) node()                       {}
func (m *CrossRefMarkup) children() []Node            { return m.Nodes }
func (m *CrossRefMarkup) setChildren(children []Node) { m.Nodes = children }

// Text represents a literal text AST node.
type Text string

func (t Text) node() {}

// Text returns the raw text
func (t Text) Text() string { return string(t) }

// DocBlock is the top-level block that contains all epydoc
// nodes.
type DocBlock struct {
	Nodes []Node
}

func (b *DocBlock) node()                       {}
func (b *DocBlock) children() []Node            { return b.Nodes }
func (b *DocBlock) setChildren(children []Node) { b.Nodes = children }

// Header is the header level of a SectionBlock.
type Header byte

// List of valid Header levels.
const (
	H1 Header = '='
	H2 Header = '-'
	H3 Header = '~'
)

// SectionBlock is a section. E.g.:
// 'Section 1
//  ========='.
type SectionBlock struct {
	Header Header
	Nodes  []Node
}

func (b *SectionBlock) node()                       {}
func (b *SectionBlock) children() []Node            { return b.Nodes }
func (b *SectionBlock) setChildren(children []Node) { b.Nodes = children }

// ParagraphBlock is a paragraph.
type ParagraphBlock struct {
	Nodes []Node
}

func (b *ParagraphBlock) node()                       {}
func (b *ParagraphBlock) children() []Node            { return b.Nodes }
func (b *ParagraphBlock) setChildren(children []Node) { b.Nodes = children }

// ListType represents the type of list (ordered or not).
type ListType int

// List of valid list types.
const (
	UnorderedList ListType = iota
	OrderedList
)

// ListBlock is a single list item. E.g.:
// '1. This is an ordered list item.'.
type ListBlock struct {
	// Bullet is the literal string representing the bullet, e.g. "-"
	// or "1.".
	Bullet   string
	ListType ListType
	Nodes    []Node
}

func (b *ListBlock) node()                       {}
func (b *ListBlock) children() []Node            { return b.Nodes }
func (b *ListBlock) setChildren(children []Node) { b.Nodes = children }

// FieldBlock is a single field definition, e.g.:
// '@param x: this is a parameter.' where
// 'param' is the `Name`, `x` is the `Arg` and
// 'this is a parameter.' is a `ParagraphBlock`
// in `Nodes`.
type FieldBlock struct {
	Name  string
	Arg   string // may be empty
	Nodes []Node
}

func (b *FieldBlock) node()                       {}
func (b *FieldBlock) children() []Node            { return b.Nodes }
func (b *FieldBlock) setChildren(children []Node) { b.Nodes = children }

// DoctestBlock is a doctest. It contains a raw string and does not
// have nested nodes.
type DoctestBlock struct {
	RawText string
}

func (b *DoctestBlock) node() {}

// Text returns the raw text
func (b *DoctestBlock) Text() string { return b.RawText }

// LiteralBlock is a literal. It contains a raw string and does not
// have nested nodes.
type LiteralBlock struct {
	RawText string
}

func (b *LiteralBlock) node() {}

// Text returns the raw text
func (b *LiteralBlock) Text() string { return b.RawText }
