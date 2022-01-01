package ast

// TODO(mna): this could probably use a common AST with epytext,
// if we wanted.

// Node is any node in the abstract syntax tree.
type Node interface {
	node()
}

// Doc represents a full numpydoc document.
type Doc struct {
	Content []Node
}

func (d *Doc) node() {}

// Section is an underlined section in the document.
type Section struct {
	Header  string
	Content []Node
}

func (s *Section) node() {}

// Directive is a sphinx directive. It has a name (the part after the
// ".." but before the "::") and a content.
type Directive struct {
	Name    string
	Content []Node
}

func (d *Directive) node() {}

// Paragraph is a paragraph of free-form text (successive non-blank lines).
// The text may contain inline markup.
type Paragraph struct {
	Content []Node
}

func (p *Paragraph) node() {}

// Definition is a definition list item. The Type is optional,
// and Content may be empty.
type Definition struct {
	Subject []Node
	Type    []Node
	Content []Node
}

func (d *Definition) node() {}

// Doctest contains lines that are meant to be runnable code
// along with expected output. The text cannot contain markup,
// so the field is a raw string.
type Doctest struct {
	Text string
}

func (d *Doctest) node() {}

// Text is a string.
type Text string

func (t Text) node() {}

// Markup identifies the type of inline markup.
type Markup byte

// List of supported inline markup.
const (
	Italics   Markup = 'i'
	Bold      Markup = 'b'
	Monospace Markup = 'm'
	Code      Markup = 'c'
)

// Inline represents a section of text with inline markup.
type Inline struct {
	// Markup identifies the type of markup to apply to Text.
	Markup Markup
	// Inline markup may not nest, from the quick ref page:
	// > inline markup may not be nested
	Text string
}

func (i *Inline) node() {}
