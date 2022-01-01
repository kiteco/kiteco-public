package traindata

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

const (
	// UnknownType is a special marker for expression nodes that we were unable to infer
	// the type of
	UnknownType = "UNKNOWN_TYPE"
	// UnknownTypeIndex is the index for unknown types in the graph nodes
	UnknownTypeIndex = 0

	// NAType is a special marker for nodes that it does not make sense to have a type for
	NAType = "NA"
	// NATypeIndex is the index for NA types in graph nodes
	NATypeIndex = 1

	// ReturnValueTail is the tail of a pythonimports.DottedPath that is used
	// to mark the return value of global functions that we do not know the return value of
	ReturnValueTail = "RETURN"
	// InstanceTail is the tail of a pythonimports.DottedPath that is used to differentiate
	// an instance of a type from the type itself
	InstanceTail = "INSTANCE"
)

// SymbolIndex maps a symbol ("type") from to a row in a type embedding matric
type SymbolIndex map[pythonimports.Hash]int

// Index of the specified type
func (si SymbolIndex) Index(t string) int {
	hash := pythonimports.NewDottedPath(t).Hash
	idx, ok := si[hash]
	if !ok {
		return UnknownTypeIndex
	}
	return idx
}

// ComputeSymbolIndex mapping hashes of canonical symbols to fixed indices.
// Also returns the set of all symbols that are present in the index.
func ComputeSymbolIndex(rm pythonresource.Manager, minScore int) (SymbolIndex, map[string]struct{}, error) {
	symToHashes, err := pythoncode.NewSymbolToHashesIndex(pythoncode.CanonicalSymbolToHashesIndexPath, "/data")
	if err != nil {
		return nil, nil, err
	}
	dists := rm.Distributions()

	var count int
	syms := make(map[keytypes.Distribution][]pythonresource.Symbol, len(dists))
	for _, d := range dists {
		cs, err := rm.CanonicalSymbols(d)
		if err != nil {
			return nil, nil, err
		}
		count += len(cs)
		syms[d] = cs
	}

	idxs := make(map[pythonimports.Hash]int)

	// add special tokens since these are often used
	// as part of the type embedding in certain cases
	hash := pythonimports.NewDottedPath(UnknownType).Hash
	idxs[hash] = UnknownTypeIndex
	hash = pythonimports.NewDottedPath(NAType).Hash
	idxs[hash] = NATypeIndex

	allSyms := make(map[string]struct{})
	allSyms[UnknownType] = struct{}{}
	allSyms[NAType] = struct{}{}

	addPath := func(p pythonimports.DottedPath) {
		allSyms[p.String()] = struct{}{}
		hash := p.Hash
		if _, ok := idxs[hash]; !ok {
			idxs[hash] = len(idxs)
		}
	}

	for _, s := range specialSubtokens {
		addPath(pythonimports.NewDottedPath(s))
	}

	for _, d := range dists {
		for _, s := range syms[d] {
			// always include builtins
			if !isBuiltin(s) && minScore > 0 {
				count := rm.SymbolCounts(s)
				if count == nil || count.Sum() < minScore {
					// these counts are based on the non-canonical scores,
					// lets try the canonical ones...
					if countFor(s, symToHashes) < minScore {
						continue
					}
				}
			}

			// GGNN models currently only operate on paths so we do the same here
			addPath(s.Path())

			switch rm.Kind(s) {
			case keytypes.FunctionKind:
				// return value for external functions that have an unknown return value
				if rets := rm.ReturnTypes(s); len(rets) > 0 {
					break
				}

				addPath(s.Path().WithTail(ReturnValueTail))
			case keytypes.TypeKind:
				// add instance for type
				addPath(s.Path().WithTail(InstanceTail))
			}
		}
	}

	return idxs, allSyms, nil
}

func countFor(sym pythonresource.Symbol, idx *pythoncode.SymbolToHashesIndex) int {
	hs, _ := idx.HashesFor(sym)
	var count int
	for _, h := range hs {
		count += int(h.Counts.CountFor(pythoncode.SymbolContextAll))
	}
	return count
}

func isBuiltin(s pythonresource.Symbol) bool {
	head := s.Canonical().Path().Head()
	return head == "builtins" || s.Dist().Name == keytypes.BuiltinDistributionName
}
