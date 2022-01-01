package pythonexpr

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonattribute"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncall"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// AttrBaseInfo contains information about the attribute base stuff
type AttrBaseInfo struct {
	// - training only
	Dist traindata.SymbolDist `json:"dist"`
}

// ForInference deletes data that is not required for inference
func (s AttrBaseInfo) ForInference() AttrBaseInfo {
	return AttrBaseInfo{}
}

// ComputeMetaInfo for the model
func ComputeMetaInfo(rm pythonresource.Manager, endpoint string, pkgs []string, minScore, maxDepth, minKwargFreq int) (MetaInfo, error) {
	pib := traindata.NewProductionIndexBuilder()

	fi, err := pythoncall.ComputeFuncInfos(rm, pib, endpoint, pkgs, minScore, pythoncode.KeywordCountsStats, minKwargFreq)
	if err != nil {
		return MetaInfo{}, err
	}

	si, err := pythonattribute.ComputeSymbolInfo(rm, pib, endpoint, pkgs, maxDepth, minScore)
	if err != nil {
		return MetaInfo{}, err
	}

	ai, err := computeAttrBaseInfo(rm, endpoint, pkgs, minScore)
	if err != nil {
		return MetaInfo{}, err
	}

	_, syms, err := traindata.ComputeSymbolIndex(rm, minScore)
	if err != nil {
		return MetaInfo{}, err
	}

	sti, err := traindata.NewSubtokenIndex(traindata.NewSubtokenIndexPath)
	if err != nil {
		return MetaInfo{}, err
	}

	tsti := computeTypeSubtokenIndex(syms)

	var children []pythonimports.Hash
	for _, v := range pythongraph.ExpansionTaskIDs() {
		children = append(children, v)
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i] < children[j]
	})

	// we just add a flat version of the grammar here
	// and just use the production index as a simple lookup map
	err = pib.Add(traindata.Production{
		ID:       pythonimports.PathHash([]byte(pythongraph.ExpansionTaskRoot)),
		Children: children,
	}, true)
	if err != nil {
		return MetaInfo{}, err
	}

	return MetaInfo{
		Attr:              si,
		Call:              fi,
		AttrBase:          ai,
		NameSubtokenIndex: sti,
		ProductionIndex:   pib.Finalize(),
		TypeSubtokenIndex: tsti,
	}, nil
}

// Valid returns nil if the meta info is valid
func (m MetaInfo) Valid() error {
	if err := m.Attr.Valid(); err != nil {
		return fmt.Errorf("attr data is invalid: %v", err)
	}
	return nil
}

func computeAttrBaseInfo(rm pythonresource.Manager, endpoint string, pkgs []string, minScore int) (AttrBaseInfo, error) {
	symbols, err := attrBaseSymbols(rm, pkgs)
	if err != nil {
		return AttrBaseInfo{}, err
	}

	scores, err := scores(endpoint, symbols, minScore)
	if err != nil {
		return AttrBaseInfo{}, err
	}

	return AttrBaseInfo{
		Dist: scores,
	}, nil
}

func attrBaseSymbols(rm pythonresource.Manager, topLevels []string) ([]pythonresource.Symbol, error) {
	skip := func(s pythonresource.Symbol) bool {
		return strings.HasPrefix(s.Path().Last(), "_")
	}

	var symbols []pythonresource.Symbol
	walk := func(tl string, parent, child pythonresource.Symbol, isTopLevel bool) error {
		switch rm.Kind(child) {
		case keytypes.ModuleKind, keytypes.TypeKind:
			// TODO: descriptor kind? object kind?
			symbols = append(symbols, child)
		}
		return nil
	}

	walker := traindata.NewWalker(rm, true, walk, skip)

	for _, tl := range topLevels {
		if err := walker.Walk(tl); err != nil {
			return nil, err
		}
	}

	return symbols, nil
}

func scores(endpoint string, symbols []pythonresource.Symbol, minScore int) (traindata.SymbolDist, error) {
	scores, _, err := traindata.GetScores(endpoint, symbols, true, pythoncode.SymbolContextName)
	if err != nil {
		return nil, err
	}

	for sym, sde := range scores {
		if sde.Weight < float64(minScore) {
			delete(scores, sym)
		}
	}

	return scores, nil
}

func computeTypeSubtokenIndex(symbols map[string]struct{}) traindata.SubtokenIndex {
	idx := make(traindata.SubtokenIndex)

	var ks []string
	for s := range symbols {
		ks = append(ks, s)
	}
	sort.Strings(ks)

	for _, sym := range ks {
		for _, tok := range pythongraph.TypeToSubtokens(sym) {
			if _, found := idx[tok]; !found {
				idx[tok] = len(idx)
			}
		}
	}

	idx.AddSpecialSubtokens()

	return idx
}
