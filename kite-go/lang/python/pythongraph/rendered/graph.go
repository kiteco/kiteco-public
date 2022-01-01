//go:generate bash -c "go-bindata $BINDATAFLAGS -pkg rendered -o bindata.go templates/..."

package rendered

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-golib/templateset"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
)

var templates = templateset.NewSet(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}, "templates", nil)

// Graph rendered as html
type Graph struct {
	Graph template.HTML
	Head  template.HTML
}

// NewGraph ...
func NewGraph(s pythongraph.SavedBundle) (Graph, error) {
	head, err := Asset("head.html")
	if err != nil {
		return Graph{}, fmt.Errorf("error loading head: %v", err)
	}

	rg := renderGraph(s)

	buf, err := json.Marshal(rg)
	if err != nil {
		return Graph{}, fmt.Errorf("error marshaling rendered graph to json: %v", err)
	}

	var b bytes.Buffer
	err = templates.Render(&b, "graph.html", map[string]interface{}{
		"Graph": template.HTML(buf),
	})

	if err != nil {
		return Graph{}, fmt.Errorf("error rendering graph template: %v", err)
	}

	return Graph{
		Graph: template.HTML(b.String()),
		Head:  template.HTML(string(head)),
	}, nil
}

type renderedNode struct {
	ID       int    `json:"id"`
	Label    string `json:"label"`
	NodeType string `json:"node_type"`
	Level    int    `json:"level"`
	Title    string `json:"title"`
}

type renderedEdge struct {
	From     int     `json:"from"`
	To       int     `json:"to"`
	EdgeType string  `json:"edge_type"`
	Label    string  `json:"label"`
	Value    float64 `json:"value"`
	Info     string  `json:"info"`
}

type renderedGraph struct {
	Nodes []renderedNode `json:"nodes"`
	Edges []renderedEdge `json:"edges"`
}

func renderGraph(s pythongraph.SavedBundle) renderedGraph {
	if s.Graph == nil {
		return renderedGraph{}
	}

	var nodes []renderedNode
	for _, sn := range s.Graph.Nodes {
		node := sn.Node
		rn := renderedNode{
			ID:       int(node.ID),
			NodeType: string(node.Type),
			Level:    sn.Level,
			Title:    sn.Hover,
		}

		switch node.Type {
		case pythongraph.ASTTerminalNode:
			rn.Label = fmt.Sprintf("%d::%s::%s", node.ID, node.Attrs.Literal, strings.Join(node.Attrs.Types, ","))
		case pythongraph.ASTInternalNode:
			rn.Label = fmt.Sprintf("%d::%s::%s", node.ID, node.Attrs.ASTNodeType, strings.Join(node.Attrs.Types, ","))
		case pythongraph.VariableUsageNode:
			rn.Label = fmt.Sprintf("%d::USAGE::%s::%s", node.ID, node.Attrs.Literal, strings.Join(node.Attrs.Types, ","))
		case pythongraph.ScopeNode:
			rn.Label = fmt.Sprintf("%d::SCOPE::%s::%s", node.ID, node.Attrs.Literal, strings.Join(node.Attrs.Types, ","))
		default:
			panic(fmt.Sprintf("unsupported node type: %s", node.Type))
		}

		if l := s.NodeLabels[node.ID]; l != "" {
			rn.Label = fmt.Sprintf("%s::%s", l, rn.Label)
		}

		nodes = append(nodes, rn)
	}

	var edges []renderedEdge
	for _, edge := range s.Graph.Edges {
		// If we don't need to visualize the edge weights, having just forward edges is enough
		if len(s.EdgeValues) == 0 && !edge.Forward {
			continue
		}

		re := renderedEdge{
			From: int(edge.From.Node.ID),
			To:   int(edge.To.Node.ID),
		}

		re.Label = string(edge.Type)
		if len(s.EdgeValues) > 0 {
			edgeValue := s.EdgeValues[pythongraph.EdgeToIDStr(edge)]
			re.Value = edgeValue.Normalized
			re.Label = pythongraph.EdgeKey(edge.Type, edge.Forward)
			re.Info = fmt.Sprintf("raw=%.5f, normalized=%.5f", edgeValue.Raw, edgeValue.Normalized)
		}
		edges = append(edges, re)
	}

	return renderedGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

// GraphWithExtras rendered as html
// TODO: better name...
type GraphWithExtras struct {
	Head template.HTML
	Body template.HTML
}

// NewGraphWithExtras ...
func NewGraphWithExtras(s pythongraph.SavedBundle, ast string) (GraphWithExtras, error) {
	head, err := Asset("templates/head.html")
	if err != nil {
		return GraphWithExtras{}, fmt.Errorf("error loading head: %v", err)
	}

	rg := renderGraph(s)

	buf, err := json.Marshal(rg)
	if err != nil {
		return GraphWithExtras{}, fmt.Errorf("error marshaling rendered graph to json: %v", err)
	}

	weightsBuf, err := json.Marshal(s.Weights)
	if err != nil {
		return GraphWithExtras{}, fmt.Errorf("error marshaling weights to json: %v", err)
	}

	var b bytes.Buffer
	err = templates.Render(&b, "graphwithextras.html", map[string]interface{}{
		"Graph":   template.HTML(buf),
		"AST":     ast,
		"Buffer":  string(s.Buffer),
		"Weights": template.HTML(weightsBuf),
	})

	if err != nil {
		return GraphWithExtras{}, fmt.Errorf("error rendering graph template: %v", err)
	}

	return GraphWithExtras{
		Body: template.HTML(b.String()),
		Head: template.HTML(string(head)),
	}, nil
}
