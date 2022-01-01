package inline

import (
	"bytes"
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/ast"
)

const (
	markupTypeKey           = "markupType"
	startMarkupSeparatorKey = "startMarkupSeparator"
)

func initState(c *current) error {
	c.state[markupTypeKey] = ast.Markup(0)
	c.state[startMarkupSeparatorKey] = ""
	return nil
}

func grammarAction(c *current, first ast.Node, rest []interface{}) (*ast.Paragraph, error) {
	var nodes []ast.Node
	if first != nil {
		nodes = append(nodes, first)
	}
	for _, v := range rest {
		switch v := v.(type) {
		case []ast.Node:
			// merge the ast.Text at index v[0] with the last one
			// in nodes if it is an ast.Text.
			if len(nodes) > 0 {
				ix := len(nodes) - 1
				if last, ok := nodes[ix].(ast.Text); ok {
					nodes[ix] = ast.Text(last + v[0].(ast.Text))
					nodes = append(nodes, v[1])
					continue
				}
			}

			nodes = append(nodes, v...)
		case ast.Node:
			nodes = append(nodes, v)
		default:
			panic(fmt.Sprintf("grammarAction: unexpected type %T", v))
		}
	}
	return &ast.Paragraph{
		Content: nodes,
	}, nil
}

func textAction(c *current) (ast.Text, error) {
	return ast.Text(string(c.text)), nil
}

func startMarkupSeparatorState(c *current, quote string) error {
	c.state[startMarkupSeparatorKey] = quote
	return nil
}

func markupTextAction(c *current, sep string, inline *ast.Inline) ([]ast.Node, error) {
	return []ast.Node{
		ast.Text(sep),
		inline,
	}, nil
}

func bofMarkupTextAction(c *current, content []interface{}, end string) (*ast.Inline, error) {
	var buf bytes.Buffer
	for _, v := range content {
		buf.WriteString(v.(string))
	}
	buf.WriteString(end)
	return &ast.Inline{
		Markup: c.state[markupTypeKey].(ast.Markup),
		Text:   buf.String(),
	}, nil
}

func startMarkupState(c *current, markup string) error {
	c.state[markupTypeKey] = markupStringToMarkup(markup)
	return nil
}

func matchingQuoteSeparatorPredicate(c *current, quote string) (bool, error) {
	startSep := c.state[startMarkupSeparatorKey].(string)
	switch startSep {
	case "'", `"`:
		return quote == startSep, nil
	case "(":
		return quote == ")", nil
	case "[":
		return quote == "]", nil
	case "{":
		return quote == "}", nil
	case "<":
		return quote == ">", nil
	default:
		return false, nil
	}
}

func endMarkupPredicate(c *current, markup string) (bool, error) {
	m := markupStringToMarkup(markup)
	want := c.state[markupTypeKey].(ast.Markup)
	return m == want, nil
}

func endMarkupAction(c *current, last []interface{}) (string, error) {
	if len(last) != 3 {
		panic("len(last) != 3")
	}
	return string(last[2].([]byte)), nil
}

func markupStringToMarkup(s string) ast.Markup {
	switch s {
	case "``":
		return ast.Monospace
	case "`":
		return ast.Code
	case "**":
		return ast.Bold
	case "*":
		return ast.Italics
	default:
		panic(fmt.Sprintf("unknown markup type: %s", s))
	}
}

// toIfaceSlice is a helper function for the PEG grammar parser. It converts
// v to a slice of empty interfaces.
func toIfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}
