package ast

import (
	"bytes"
	"testing"
)

func TestOptimize(t *testing.T) {
	cases := []struct {
		n    Node
		want string
	}{
		{&DocBlock{}, "Doc\n"},

		{&ParagraphBlock{Nodes: []Node{&BasicMarkup{Type: B}}}, "Paragraph\n"},

		{&ParagraphBlock{Nodes: []Node{
			Text("a"),
			&BasicMarkup{Type: B, Nodes: []Node{Text("b")}},
			Text("c"),
			&BasicMarkup{Type: B, Nodes: []Node{Text("")}},
		}}, "Paragraph\n\tText[a]\n\tBasicMarkup[B]\n\t\tText[b]\n\tText[c]\n"},

		{&ParagraphBlock{Nodes: []Node{
			Text("a"),
			&BasicMarkup{Type: B, Nodes: []Node{
				Text("b"),
				&BasicMarkup{Type: I, Nodes: []Node{
					Text(""),
					&BasicMarkup{Type: B, Nodes: []Node{Text("")}},
				}},
				Text("c"),
			}},
		}}, "Paragraph\n\tText[a]\n\tBasicMarkup[B]\n\t\tText[bc]\n"},

		{&ParagraphBlock{Nodes: []Node{
			Text("a"),
			Text(""),
			Text("b"),
		}}, "Paragraph\n\tText[ab]\n"},

		{&ParagraphBlock{Nodes: []Node{
			Text("a"),
		}}, "Paragraph\n\tText[a]\n"},

		{&ParagraphBlock{Nodes: []Node{
			Text("a"),
			Text("b"),
		}}, "Paragraph\n\tText[ab]\n"},

		{&ParagraphBlock{Nodes: []Node{
			&ParagraphBlock{Nodes: []Node{Text("A")}},
			Text("a"),
			Text("b"),
		}}, "Paragraph\n\tParagraph\n\t\tText[A]\n\tText[ab]\n"},

		{&ParagraphBlock{Nodes: []Node{
			Text("a"),
			&ParagraphBlock{Nodes: []Node{Text("A")}},
			Text("b"),
		}}, "Paragraph\n\tText[a]\n\tParagraph\n\t\tText[A]\n\tText[b]\n"},

		{&ParagraphBlock{Nodes: []Node{
			Text("a"),
			Text("b"),
			&ParagraphBlock{Nodes: []Node{Text("A")}},
		}}, "Paragraph\n\tText[ab]\n\tParagraph\n\t\tText[A]\n"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			Optimize(c.n)

			var buf bytes.Buffer
			Print(c.n, &buf, "\t")
			if got := buf.String(); got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}
