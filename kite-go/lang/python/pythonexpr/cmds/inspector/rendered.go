package main

import (
	"bytes"
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/rendered"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type renderedSample struct {
	Name  string
	Graph rendered.GraphWithExtras
}

func renderSavedSamples(saved ...pythongraph.SavedBundle) ([]renderedSample, error) {
	var rs []renderedSample
	for i, s := range saved {
		if pythonast.IsNil(s.AST) {
			s.AST, _ = pythonparser.Parse(kitectx.Background(), []byte(s.Buffer), parseOpts)
		}

		var ast string
		if !pythonast.IsNil(s.AST) {
			var b bytes.Buffer
			pythonast.Print(s.AST, &b, "\t")
			ast = b.String()
		}

		var graph rendered.GraphWithExtras
		if s.Graph != nil {
			var err error
			graph, err = rendered.NewGraphWithExtras(s, ast)
			if err != nil {
				return nil, err
			}
		}

		rs = append(rs, renderedSample{
			Name:  fmt.Sprintf("%d-%s", i, s.Label),
			Graph: graph,
		})
	}

	return rs, nil
}

func renderLinks(hash string, nid int, rss ...renderedSample) []renderedSampleLink {
	var ls []renderedSampleLink
	for _, rs := range rss {
		ls = append(ls, renderedSampleLink{
			Name: rs.Name,
			Hash: hash,
			Node: nid,
		})
	}
	return ls
}

type renderedSampleLink struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
	Node int    `json:"node"`
}
