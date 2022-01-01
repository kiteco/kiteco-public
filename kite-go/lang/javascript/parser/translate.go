package parser

import (
	"fmt"
	"go/token"

	"github.com/kiteco/kiteco/kite-go/lang/javascript/ast"
	"github.com/kiteco/kiteco/kite-go/lang/javascript/parser/internal/pigeon"
	"github.com/kr/pretty"
)

func translate(module *pigeon.Node, src []byte) (*ast.Node, error) {
	if module == nil {
		return nil, fmt.Errorf("got nil module")
	}
	return translateImpl(module, src)
}

func translateImpl(nr interface{}, src []byte) (*ast.Node, error) {
	n := nr.(*pigeon.Node)
	node := &ast.Node{
		Begin: token.Pos(n.Begin),
		End:   token.Pos(n.Begin + n.Len),
		Type:  n.Type,
	}
	node.Literal = src[node.Begin:node.End]

	for _, child := range flatten(n.Children) {
		translated, err := translateImpl(child, src)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, translated)
	}

	return fold(node, src)
}

// fold flattened nodes back into recursive structure.
func fold(node *ast.Node, src []byte) (*ast.Node, error) {
	var err error
	switch node.Type {
	case ast.BinaryExpression:
		switch len(node.Children) {
		case 0:
			return nil, fmt.Errorf("got BinaryExpression with no child nodes: %s", pretty.Sprintf("%#v", node))
		case 1, 2:
			// update length/literal for base binary expression in expressions such as foo | bar | car
			last := len(node.Children) - 1
			node.End = node.Children[last].End
			node.Literal = src[node.Begin:node.End]
		default:
			node, err = split(node, node.Type, src)
		}
	case ast.MemberExpression:
		switch len(node.Children) {
		case 0:
			return nil, fmt.Errorf("got MemberExpression with no child nodes: [%d...%d][%s]", node.Begin, node.End, string(node.Literal))
		case 1:
			node = node.Children[0]
		default:
			node, err = split(node, ast.MemberExpression, src)
		}
	case ast.Call:
		// we need to deal with cases like
		// foo.bar.car.zar
		// foo(bar)(car)(mar)
		// foo(bar)(car).zar
		switch len(node.Children) {
		case 0, 1:
			return nil, fmt.Errorf("got Call with %d child nodes: %s", len(node.Children), pretty.Sprintf("%#v", node))
		case 2:
			// update length/literal for base call in expressions such as foo(bar)(car)
			node.End = node.Children[1].End
			node.Literal = src[node.Begin:node.End]
		default:
			switch node.Children[2].Type {
			case ast.Arguments:
				// foo(bar,car,...)(zar)
				node, err = split(node, ast.Call, src)
			case ast.Identifier:
				// foo(bar,car).zar
				node, err = split(node, ast.MemberExpression, src)
			}
		}
	}

	if err != nil {
		return nil, err
	}

	return node, nil
}

func split(node *ast.Node, t ast.Type, src []byte) (*ast.Node, error) {
	if len(node.Children) < 2 {
		return nil, fmt.Errorf("split called on node with less than 2 children: %s", pretty.Sprintf("%#v", node))
	}

	tail := node.Children[len(node.Children)-1]
	node.Children = node.Children[:len(node.Children)-1]
	base, err := fold(node, src)
	if err != nil {
		return nil, err
	}

	node = &ast.Node{
		Begin:    base.Begin,
		End:      tail.End,
		Type:     t,
		Children: []*ast.Node{base, tail},
	}
	node.Literal = src[node.Begin:node.End]
	return node, nil
}

func flatten(i interface{}) []interface{} {
	var flattened []interface{}
	switch i := i.(type) {
	case nil:
	case []interface{}:
		for _, ii := range i {
			flattened = append(flattened, flatten(ii)...)
		}
	case []uint8: // terminal symbols
	case interface{}:
		flattened = append(flattened, i)
	}
	return flattened
}
