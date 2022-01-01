package main

import (
	"math"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

func buildTypes(pkgs map[string]bool, rm pythonresource.Manager, minCount int, pretrainedAttrs map[pythonimports.Hash]AttrDist) []*Type {
	children := func(p pythonresource.Symbol) []pythonresource.Symbol {
		children, err := rm.Children(p)
		if err != nil {
			return nil
		}

		var css []pythonresource.Symbol
		for _, child := range children {
			cs, err := rm.ChildSymbol(p, child)
			if err != nil {
				continue
			}
			css = append(css, cs)
		}
		return css
	}

	seen := make(map[pythonimports.Hash]bool)
	ts := make(map[pythonimports.Hash]*Type)
	for _, dist := range rm.Distributions() {
		tls, err := rm.TopLevels(dist)
		if err != nil {
			continue
		}

		// walk via the top levels so that we
		// we get more representative scores for aliases
		// of a canonical symbol
		for _, tl := range tls {
			if !pkgs[tl] {
				continue
			}

			sym, err := rm.NewSymbol(dist, pythonimports.NewDottedPath(tl))
			if err != nil {
				continue
			}

			q := []pythonresource.Symbol{sym}
			for len(q) > 0 {
				s := q[0]
				q = q[1:]
				canon := s.Canonical()
				if !pkgs[canon.Path().Head()] {
					continue
				}
				switch rm.Kind(canon) {
				case keytypes.TypeKind:
					// types are indexed by canonical symbol
					t := ts[canon.Hash()]
					if t == nil {
						t = &Type{
							Pkg:   canon.Path().Head(),
							Sym:   canon,
							Dist:  canon.Dist(),
							Attrs: make(AttrDist),
						}
						ts[canon.Hash()] = t
					}

					var tScore float64
					if count := rm.SymbolCounts(s); count != nil {
						tScore = float64(count.Sum())
					}

					if tScore > t.Prob {
						t.Prob = tScore
					}

					if attrs, ok := pretrainedAttrs[s.PathHash()]; ok {
						t.Attrs = attrs
					} else if attrs, ok := pretrainedAttrs[canon.PathHash()]; ok {
						t.Attrs = attrs
					} else {
						for _, cs := range children(s) {
							var aScore float64
							if count := rm.SymbolCounts(cs); count != nil {
								aScore = float64(count.Sum())
							}

							// log normalize to keep counts in roughly the same order of magnitude
							aScore = math.Log(1 + aScore)
							if aScore > t.Attrs[cs.Path().Last()] {
								t.Attrs[cs.Path().Last()] = aScore
							}
						}
						// NOTE: we could explore children of the type that are also types
						// but we ignore this for now
					}

				case keytypes.ModuleKind:
					// check to make sure we have not explored the module already
					if seen[canon.Hash()] {
						break
					}
					seen[canon.Hash()] = true

					for _, cs := range children(canon) {
						q = append(q, cs)
					}
				}
			}
		}
	}

	var filtered []*Type
	var tScores float64
	for _, t := range ts {
		if minCount > 0 && t.Prob < float64(minCount) {
			logf(logLevelWarn, "skipping type %s, count too low %f < %d\n", t.Sym.String(), t.Prob, minCount)
			continue
		}

		if len(t.Attrs) == 0 {
			logf(logLevelWarn, "skipping type %s, no attributes found\n", t.Sym.String())
			continue
		}

		// log normalize to keep counts in roughly the same order of magnitude,
		// make sure to do this after count check above to avoid confusion
		t.Prob = math.Log(1 + t.Prob)
		tScores += t.Prob

		filtered = append(filtered, t)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Sym.Less(filtered[j].Sym)
	})

	// normalize type prior and attribute dist for each type
	for _, t := range filtered {
		t.Prob /= tScores

		var aScores float64
		for _, aScore := range t.Attrs {
			aScores += aScore
		}

		t.Attrs.Normalize(aScores)
	}

	return filtered

}
