package pythonattribute

import (
	"fmt"
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// SymbolInfo contains information about the symbols used to train the model.
type SymbolInfo struct {
	// CanonToSyms maps canonical symbols to aliases that point to it,
	// we need this to support ggnn attribute completions on instances
	// since we cannot guarantee that static analysis will not canonicalize the symbol
	// for return values
	CanonToSyms map[string][]string

	// - training only
	// Dist contains the distribution of symbols that the training data is sampled from.
	Dist traindata.SymbolDist `json:"dist"`
	// Parents map each symbol to its parent symbol
	Parents map[string]string `json:"parents"`
}

// ForInference deletes data that is not required for inference
func (s SymbolInfo) ForInference() SymbolInfo {
	return SymbolInfo{
		CanonToSyms: s.CanonToSyms,
	}
}

// Valid symbol info
func (s SymbolInfo) Valid() error {
	for sym := range s.Dist {
		if _, ok := s.Parents[sym]; !ok {
			return fmt.Errorf("no parent for %s", sym)
		}
	}
	if len(s.Dist) != len(s.Parents) {
		return fmt.Errorf("len dist %d != len parents %d", len(s.Dist), len(s.Parents))
	}
	return nil
}

// ComputeSymbolInfo for ggnn attribute training
func ComputeSymbolInfo(rm pythonresource.Manager, pib traindata.ProductionIndexBuilder, endpoint string, pkgs []string, maxDepth, minScore int) (SymbolInfo, error) {
	symbols, typeAttrs, err := symbols(rm, pkgs, maxDepth)
	if err != nil {
		return SymbolInfo{}, err
	}

	scores, err := scores(endpoint, symbols, typeAttrs, minScore)
	if err != nil {
		return SymbolInfo{}, err
	}

	cands := candidateMap(scores)

	calcDist(scores, cands)

	// filter out any cands that are not in the final dist
	for s, children := range cands {
		var newChildren []string
		for _, child := range children {
			if _, ok := scores[child]; ok {
				newChildren = append(newChildren, child)
			}
		}

		if len(newChildren) > 0 {
			cands[s] = newChildren
		} else {
			delete(cands, s)
		}
	}

	// TODO: add a prefix to make ids unique?
	parents := make(map[string]string, len(scores))
	for parent, children := range cands {
		var childIDs []pythonimports.Hash
		for _, child := range children {
			childIDs = append(childIDs, pythonimports.PathHash([]byte(child)))
		}
		pib.Add(traindata.Production{
			ID:       pythonimports.PathHash([]byte(parent)),
			Children: childIDs,
		}, false)
		for _, child := range children {
			if p, ok := parents[child]; ok {
				return SymbolInfo{}, fmt.Errorf("child %s has multiple parents %s and %s", child, parent, p)
			}
			parents[child] = parent
		}
	}

	canonToSyms := make(map[string][]string)
	for s := range scores {
		sym, err := rm.PathSymbol(pythonimports.NewDottedPath(s))
		if err != nil {
			return SymbolInfo{}, fmt.Errorf("unable to map %s to a symbol: %v", s, err)
		}
		p := sym.Canonical().PathString()
		canonToSyms[p] = append(canonToSyms[p], s)
	}

	si := SymbolInfo{
		Dist:        scores,
		Parents:     parents,
		CanonToSyms: canonToSyms,
	}
	if err := si.Valid(); err != nil {
		return SymbolInfo{}, err
	}
	return si, nil
}

func symbols(rm pythonresource.Manager, topLevels []string, maxDepth int) ([]pythonresource.Symbol, []pythonresource.Symbol, error) {
	skip := func(s pythonresource.Symbol) bool {
		depth := maxDepth
		// TODO: hacky, if we expand everything to depth 5 the model becomes massive
		// and alot of our infra breaks down. For now we do this hack to make
		// sure google is included
		if s.Path().Head() == "google" {
			depth = 5
		}
		return len(s.Path().Parts) > depth || strings.HasPrefix(s.Path().Last(), "_")
	}

	var symbols []pythonresource.Symbol
	var typeAttrs []pythonresource.Symbol
	walk := func(tl string, parent, child pythonresource.Symbol, isTopLevel bool) error {
		if isTopLevel {
			// we don't add the top-level symbol to the list because we're only interested in symbols
			// that are attributes of their parents
			return nil
		}

		if rm.Kind(parent) == keytypes.TypeKind {
			typeAttrs = append(typeAttrs, child)
		} else {
			symbols = append(symbols, child)
		}
		return nil
	}

	walker := traindata.NewWalker(rm, false, walk, skip)

	for _, tl := range topLevels {
		if err := walker.Walk(tl); err != nil {
			return nil, nil, err
		}
	}

	return symbols, typeAttrs, nil
}

func scores(endpoint string, symbols, typeAttrs []pythonresource.Symbol, minScore int) (traindata.SymbolDist, error) {
	scores, errs, err := traindata.GetScores(endpoint, symbols, false, pythoncode.SymbolContextAttribute)
	if err != nil {
		return nil, err
	}

	fmt.Printf("got %d errors getting non canonical attr symbol scores\n", len(errs))

	// we need to get the scores for the canonicalized symbols that correspond to
	// attributes of a type because return types of function calls (including __init__) are always canonicalized
	// which means that anytime we are trying to suggest attributes of an instance
	// we cannot rely on the noncanonical score since it is usually 0
	scoresAttrs, errs, err := traindata.GetScores(endpoint, typeAttrs, true, pythoncode.SymbolContextAttribute)
	if err != nil {
		return nil, err
	}

	fmt.Printf("got %d errors getting canonical type attr symbol scores\n", len(errs))

	for sym, sde := range scoresAttrs {
		// should not need to check but just to be safe
		if _, ok := scores[sym]; !ok {
			scores[sym] = sde
		}
	}

	for sym, sde := range scores {
		ms := float64(minScore)
		// hack to include enough members of the google packge
		// into the model
		if strings.HasPrefix(sym, "google") {
			ms = 50
		}
		if sde.Weight < ms {
			delete(scores, sym)
		}
	}

	return scores, nil
}

// calcDist finds the distribution with which the symbols should be sampled.
// Formula used is:
//   p(symbol) ~ sqrt(sum(sibling_scores)) * sibling_entropy * symbol_score / sum(sibling_scores)
// where:
//   sum(sibling_scores) is the sum of all the scores of the symbols that share the same parent as the given symbol
//   sibling_entropy = -sum(p_i * log(p_i)) where p_i = score of each sibling symbol / sum(sibling_scores)
//   symbol_score = score of the given symbol
//
func calcDist(scores traindata.SymbolDist, byParent map[string][]string) {
	dist := make(map[string]float64)

	// for each parent, calculate the sums of the child scores and entropies, and then calculate the probabilities
	// of all the children
	for _, children := range byParent {
		var childSum float64
		for _, child := range children {
			childSum += scores[child].Weight
		}

		var childEntropy float64
		for _, child := range children {
			childProb := scores[child].Weight / childSum
			childEntropy += -childProb * math.Log(childProb)
		}

		for _, child := range children {
			prob := math.Sqrt(childSum) * childEntropy * scores[child].Weight / childSum
			// The calculated probability can be zero sometimes, e.g. there is only one child for a given
			// parent, so we filter those cases out
			if prob > 0 {
				dist[child] = prob
			}
		}
	}

	for sym, sde := range scores {
		score, ok := dist[sym]
		if !ok {
			delete(scores, sym)
		} else {
			sde.Weight = score
		}
	}
}

func candidateMap(scores traindata.SymbolDist) map[string][]string {
	cands := make(map[string][]string)
	for sym := range scores {
		// TODO: this is super sketchy!!!! we cannot guarantee that
		// this will be right right parent...
		parent := pythonimports.NewDottedPath(sym).Predecessor().String()
		cands[parent] = append(cands[parent], sym)
	}
	return cands
}

func unique(ss []string) {
	seen := make(map[string]bool)
	for _, s := range ss {
		if seen[s] {
			panic(fmt.Sprintf("duplicated %s", s))
		}
		seen[s] = true
	}
}
