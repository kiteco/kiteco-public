package main

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
)

type searchNodeLink struct {
	Hash string `json:"hash"`
	Node int    `json:"node"`
}

type searchNode struct {
	ID    int            `json:"id"`
	Label string         `json:"label"`
	Level int            `json:"level"`
	Link  searchNodeLink `json:"link"`

	Samples     []renderedSample     `json:"-"`
	SampleLinks []renderedSampleLink `json:"-"`
	Text        string               `json:"-"`
}

type searchEdge struct {
	From  int    `json:"from"`
	To    int    `json:"to"`
	Label string `json:"label"`
}

type searchGraph struct {
	Nodes []*searchNode `json:"nodes"`
	Edges []searchEdge  `json:"edges"`
}

type saver struct {
	Saved []pythongraph.SavedBundle
}

func (s *saver) Save(sb pythongraph.SavedBundle) {
	s.Saved = append(s.Saved, sb)
}

func newSearchGraph(hash string, s pythongraph.SavedBundle) (*searchGraph, error) {
	var nodes []*searchNode
	addNode := func(s pythongraph.SavedBundle) (*searchNode, error) {
		nid := len(nodes)
		samples, err := renderSavedSamples(s.Entries...)
		if err != nil {
			return nil, err
		}

		links := renderLinks(hash, nid, samples...)

		node := &searchNode{
			ID:    nid,
			Label: s.Label,
			Level: -1,
			Link: searchNodeLink{
				Hash: hash,
				Node: nid,
			},
			Samples:     samples,
			SampleLinks: links,
			Text:        string(s.Buffer),
		}

		nodes = append(nodes, node)
		return node, nil
	}

	root, err := addNode(s)
	if err != nil {
		return nil, err
	}

	var edges []searchEdge
	var recur func(*searchNode, pythongraph.SavedBundle) error
	recur = func(n *searchNode, s pythongraph.SavedBundle) error {
		for _, child := range s.Children {
			cn, err := addNode(child)
			if err != nil {
				return err
			}

			edges = append(edges, searchEdge{
				From:  n.ID,
				To:    cn.ID,
				Label: fmt.Sprintf("%.4f", child.Prob),
			})
			if err := recur(cn, child); err != nil {
				return err
			}
		}

		return nil
	}

	if err := recur(root, s); err != nil {
		return nil, err
	}

	return &searchGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}
