package pythoncall

import (
	"encoding/gob"
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const (
	maxNumArgs = 3
)

// FuncInfo contains information about the score for a symbol along with
// its maximum pattern
type FuncInfo struct {
	// The possible keyword argument names for the given func
	KwargNames []string              `json:"kwarg_names"`
	Symbol     pythonresource.Symbol `json:"-"`
}

// FuncInfos contains all the information for the (function) symbols that the model trained on
type FuncInfos struct {
	Infos map[string]*FuncInfo `json:"infos"`

	// - training only
	Dist traindata.SymbolDist `json:"dist"`
}

// ForInference deletes data that is not required for inference
func (fis FuncInfos) ForInference() FuncInfos {
	return FuncInfos{
		Infos: fis.Infos,
	}
}

// SymbolForFunc returns the appropriate symbol to check for the provided func symbol
// returns an empty symbol if `fs` is not supported
func SymbolForFunc(rm pythonresource.Manager, fs pythonresource.Symbol) pythonresource.Symbol {
	// always canonicalize
	fs = fs.Canonical()

	switch rm.Kind(fs) {
	case keytypes.TypeKind:
		// see if we can find an __init__ for the type
		init, err := rm.ChildSymbol(fs, "__init__")
		if err != nil {
			return pythonresource.Symbol{}
		}
		fs = init.Canonical()
	case keytypes.FunctionKind:
	default:
		return pythonresource.Symbol{}
	}
	return fs
}

// GetKwargNames gets keyword argument names for funcs, extracted from github
func GetKwargNames(kwargInfoPath string, minKwargNameFreq int) (map[string][]string, error) {
	r, err := fileutil.NewCachedReader(kwargInfoPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var kwc pythoncode.KeywordCountsByFunc
	if err := gob.NewDecoder(r).Decode(&kwc); err != nil {
		return nil, err
	}

	counts := make(map[string][]string)
	for f := range kwc {
		var keys []string
		for key, count := range kwc[f] {
			if count > int32(minKwargNameFreq) {
				keys = append(keys, key)
			}
		}
		if len(keys) > 0 {
			counts[f] = keys
		}
	}
	return counts, nil
}

// ComputeFuncInfos for ggnn call training
func ComputeFuncInfos(rm pythonresource.Manager, pib traindata.ProductionIndexBuilder, endpoint string, topLevels []string, minScore int, kwargInfoPath string, minKwargNameFreq int) (FuncInfos, error) {
	funcs, err := funcs(rm, topLevels)
	if err != nil {
		return FuncInfos{}, err
	}

	scores, err := scores(endpoint, funcs, minScore)
	if err != nil {
		return FuncInfos{}, err
	}

	kwargs, err := GetKwargNames(kwargInfoPath, minKwargNameFreq)

	infos := make(map[string]*FuncInfo, len(scores))
	for _, f := range funcs {
		s := f.PathString()

		if _, ok := scores[s]; !ok {
			continue
		}

		keys := kwargs[s]

		infos[s] = &FuncInfo{
			KwargNames: append([]string{}, keys...),
			Symbol:     f,
		}
	}

	for sym, info := range infos {
		var chooseKwargIDs []pythonimports.Hash
		for _, kn := range info.KwargNames {
			chooseKwargIDs = append(chooseKwargIDs, traindata.IDForChooseKwarg(sym, kn))
		}

		sortHashes(chooseKwargIDs)

		err := pib.Add(traindata.Production{
			ID:       traindata.IDForChooseKwargParent(sym),
			Children: chooseKwargIDs,
		}, true)
		if err != nil {
			return FuncInfos{}, fmt.Errorf("error adding kw to production index: %v", err)
		}

		var argTypeIDs []pythonimports.Hash
		for _, argType := range traindata.ArgTypes {
			argTypeIDs = append(argTypeIDs, traindata.IDForChooseArgType(sym, argType))
		}

		sortHashes(argTypeIDs)
		err = pib.Add(traindata.Production{
			ID:       traindata.IDForChooseArgTypeParent(sym),
			Children: argTypeIDs,
		}, true)
		if err != nil {
			return FuncInfos{}, fmt.Errorf("error adding arg type to production index: %v", err)
		}

		sigStats := rm.SigStats(info.Symbol)

		var argPlaceholderIDs []pythonimports.Hash
		for name := range sigStats.ArgsByName {
			argPlaceholderIDs = append(argPlaceholderIDs,
				traindata.IDForChooseArgPlaceholder(sym, name, traindata.Placeholder),
				traindata.IDForChooseArgPlaceholder(sym, name, traindata.NoPlaceholder))
		}
		sort.Slice(argPlaceholderIDs, func(i, j int) bool {
			return argPlaceholderIDs[i] < argPlaceholderIDs[j]
		})

		err = pib.Add(traindata.Production{
			ID:       traindata.IDForChooseArgPlaceholderParent(sym),
			Children: argPlaceholderIDs,
		}, true)
		if err != nil {
			return FuncInfos{}, fmt.Errorf("error adding arg type to production index: %v", err)
		}
	}
	return FuncInfos{
		Dist:  scores,
		Infos: infos,
	}, nil
}

var targetFunctions []string

func funcs(rm pythonresource.Manager, topLevels []string) ([]pythonresource.Symbol, error) {

	// Set here the functions you want to limit your metainfo to
	// Let the variable empty to train over all the functions
	// targetFunctions = []string{"requests.api.get", "requests.api.put", "requests.api.post", "requests.api.patch"}
	var funcs []pythonresource.Symbol

	var skipped int
	walk := func(tl string, parent, child pythonresource.Symbol, isTopLevel bool) error {
		child = child.Canonical()

		if k := rm.Kind(child); k != keytypes.FunctionKind {
			return nil
		}

		if strings.Contains(strings.ToLower(child.PathString()), "tests") {
			skipped++
			return nil
		}

		if !isInTargetList(child.PathString()) {
			return nil
		}

		if traindata.NewCallPatterns(rm, child) != nil {
			funcs = append(funcs, child)
		}

		return nil
	}

	walker := traindata.NewWalker(rm, true, walk, nil)
	for _, tl := range topLevels {
		if err := walker.Walk(tl); err != nil {
			return nil, err
		}
	}

	fmt.Printf("skipped %d test functions\n", skipped)
	if len(targetFunctions) > 0 {
		fmt.Printf("Selected symbols : %v\n ", funcs)
	}
	return funcs, nil
}

func isInTargetList(s string) bool {
	if len(targetFunctions) == 0 {
		return true
	}
	for _, ss := range targetFunctions {
		if strings.Contains(ss, s) {
			return true
		}
	}
	return false
}

func sortHashes(hs []pythonimports.Hash) {
	sort.Slice(hs, func(i, j int) bool {
		return hs[i] < hs[j]
	})
}

func scores(endpoint string, syms []pythonresource.Symbol, minScore int) (traindata.SymbolDist, error) {
	scores, errs, err := traindata.GetScores(endpoint, syms, true, pythoncode.SymbolContextCallFunc)
	if err != nil {
		return nil, err
	}

	fmt.Printf("got %d errors getting call symbol scores\n", len(errs))

	for sym, sde := range scores {
		// TODO: hacky, make sure we include enough interesting google functions without
		// blowing up the size of our model
		ms := float64(minScore)
		if strings.HasPrefix(sym, "google") {
			ms = 50
		}
		if sde.Weight < ms {
			delete(scores, sym)
		}
	}

	return scores, nil
}
