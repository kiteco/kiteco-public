package ast

import (
	"bytes"
	"testing"
)

func TestSprint(t *testing.T) {
	cases := []struct {
		name string
		in   Node
		out  string
	}{
		{"doc", &DocBlock{}, "Doc\n"},
		{"doc+section", &DocBlock{
			Nodes: []Node{
				&SectionBlock{Nodes: []Node{Text("H1")}, Header: H1},
			},
		}, "Doc\n\tSection[=]\n\t\tText[H1]\n"},
		{"doc+paragraph", &DocBlock{
			Nodes: []Node{
				&ParagraphBlock{Nodes: []Node{Text("p")}},
			},
		}, "Doc\n\tParagraph\n\t\tText[p]\n"},
		{"doc+section+paragraph", &DocBlock{
			Nodes: []Node{
				&SectionBlock{Header: H2, Nodes: []Node{
					Text("H2"),
					&ParagraphBlock{Nodes: []Node{Text("p")}},
				}},
			},
		}, "Doc\n\tSection[-]\n\t\tText[H2]\n\t\tParagraph\n\t\t\tText[p]\n"},
		{"doc+list", &DocBlock{
			Nodes: []Node{
				&ListBlock{Bullet: "-", ListType: UnorderedList},
			},
		}, "Doc\n\tList[- (0)]\n"},
		{"doc+list+paragraph", &DocBlock{
			Nodes: []Node{
				&ListBlock{Bullet: "1.2.", ListType: OrderedList, Nodes: []Node{
					&ParagraphBlock{Nodes: []Node{Text("p")}},
				}},
			},
		}, "Doc\n\tList[1.2. (1)]\n\t\tParagraph\n\t\t\tText[p]\n"},
		{"doc+field", &DocBlock{
			Nodes: []Node{
				&FieldBlock{Name: "f"},
			},
		}, "Doc\n\tField[f ()]\n"},
		{"doc+doctest", &DocBlock{
			Nodes: []Node{
				&DoctestBlock{RawText: ">>> doctest"},
			},
		}, "Doc\n\tDoctest[>>> doctest]\n"},
		{"doc+paragraph+literal", &DocBlock{
			Nodes: []Node{
				&ParagraphBlock{Nodes: []Node{
					Text("p"),
					&LiteralBlock{RawText: "lit"},
				}},
			},
		}, "Doc\n\tParagraph\n\t\tText[p]\n\t\tLiteral[lit]\n"},

		{"doc+complex", &DocBlock{
			Nodes: []Node{
				&ParagraphBlock{Nodes: []Node{Text("before section")}},
				&SectionBlock{Header: H1, Nodes: []Node{
					Text("H1"),
					&ParagraphBlock{Nodes: []Node{
						Text("inside H1:"),
						&LiteralBlock{RawText: "lit"},
					}},
					&SectionBlock{Header: H2, Nodes: []Node{
						Text("H2"),
						&ListBlock{Bullet: "-", ListType: UnorderedList, Nodes: []Node{
							&ParagraphBlock{Nodes: []Node{Text("list 1")}},
						}},
						&ListBlock{Bullet: "-", ListType: UnorderedList, Nodes: []Node{
							&ParagraphBlock{Nodes: []Node{Text("list 2")}},
							&DoctestBlock{RawText: ">>> doctest"},
						}},
					}},
				}},
				&FieldBlock{Name: "f1"},
				&FieldBlock{Name: "f2", Arg: "arg", Nodes: []Node{
					&ParagraphBlock{Nodes: []Node{Text("field 2")}},
				}},
			},
		}, "Doc\n\tParagraph\n\t\tText[before section]\n\tSection[=]\n\t\tText[H1]\n\t\tParagraph\n\t\t\tText[inside H1:]\n\t\t\tLiteral[lit]\n\t\tSection[-]\n\t\t\tText[H2]\n\t\t\tList[- (0)]\n\t\t\t\tParagraph\n\t\t\t\t\tText[list 1]\n\t\t\tList[- (0)]\n\t\t\t\tParagraph\n\t\t\t\t\tText[list 2]\n\t\t\t\tDoctest[>>> doctest]\n\tField[f1 ()]\n\tField[f2 (arg)]\n\t\tParagraph\n\t\t\tText[field 2]\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			Print(c.in, &buf, "\t")
			if got := buf.String(); got != c.out {
				t.Errorf("\nwant:\n%s\ngot:\n%s", c.out, got)
			}
		})
	}
}
