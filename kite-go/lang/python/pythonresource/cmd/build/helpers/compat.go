package helpers

import (
	"compress/gzip"
	"encoding/json"
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Compat represents a map from import graph canonical names to Symbols
type Compat map[int64]keytypes.Symbol

// LoadCompat loads a Compat index from file
func LoadCompat(fpath string) (Compat, error) {
	r, err := fileutil.NewCachedReader(fpath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	gzR, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzR.Close()

	index := make(Compat)
	if err := json.NewDecoder(gzR).Decode(&index); err != nil {
		return nil, err
	}

	return index, nil
}

// Lookup looks up the corresponding pythonresource Symbols for the given import graph node ID.
// If it returns an error, the Compat index is out of date or the compat builder has a bug.
func (i Compat) Lookup(rm pythonresource.Manager, nodeID int64) (pythonresource.Symbol, error) {
	if sym, ok := i[nodeID]; ok {
		return rm.NewSymbol(sym.Dist, sym.Path)
	}
	return pythonresource.Symbol{}, fmt.Errorf("no symbol found for node %d", nodeID)
}

// Shard calls cb on pairs of symbols and corresponding nodes. The same symbol(s) or node(s) may be passed to cb multiple times, so the client should dedupe.
// It returns an error only if something went very wrong; the client should probably abort on error.
func (i Compat) Shard(rm pythonresource.Manager, graph *pythonimports.Graph, cb func(sym pythonresource.Symbol, node *pythonimports.Node)) error {
	for _, dist := range rm.Distributions() {
		syms, err := rm.CanonicalSymbols(dist)
		if err != nil {
			return err
		}

		for _, sym := range syms {
			node, _ := graph.Navigate(sym.Path())
			if node == nil {
				continue
			}
			cb(sym, node)
		}
	}

	for j := range graph.Nodes {
		node := &graph.Nodes[j]

		sym, err := i.Lookup(rm, node.ID)
		if err != nil {
			continue
		}

		cb(sym, node)
	}

	return nil
}
