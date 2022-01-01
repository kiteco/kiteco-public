package pythongraph

import (
	"fmt"
	"go/token"
	"math/rand"
	"sort"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

func assertTrue(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

func filterScopeByKinds(ctx kitectx.Context, a *analysis, s scope, invalidKinds ...pythontype.Kind) []*variable {
	ctx.CheckAbort()
	var reduced scope
	for _, v := range s {
		val := a.Resolve(ctx, v.Origin)
		if val == nil {
			// make sure to include unresolved names
			reduced = append(reduced, v)
			continue
		}

		gvs := a.ResolveToGlobals(ctx, v.Origin)
		if len(gvs) == 0 {
			// do not include names that resolved to source values
			continue
		}

		ok := true
	valCheck:
		for _, val := range gvs {
			kind := val.Kind()
			for _, invalid := range invalidKinds {
				if invalid == kind {
					ok = false
					break valCheck
				}
			}
		}
		if ok {
			reduced = append(reduced, v)
		}
	}
	return reduced
}

func trimEndLineOrStmt(pos token.Pos, n pythonast.Expr, lm *linenumber.Map, parents map[pythonast.Expr]pythonast.Stmt) int {
	_, lePos := lm.LineBounds(lm.Line(int(pos)))
	stmt := parents[n]
	if lePos < int(stmt.End()) {
		return int(stmt.End())
	}
	return lePos
}

func asciiOnly(s string) string {
	rs := make([]rune, 0, len(s))
	for _, c := range s {
		if c > unicode.MaxASCII {
			rs = append(rs, '#')
		} else {
			rs = append(rs, c)
		}
	}

	return string(rs)
}

func matchesAny(s pythonresource.Symbol, any []pythonresource.Symbol, canonicalize bool) bool {
	if canonicalize {
		s = s.Canonical()
	}
	for _, ss := range any {
		if canonicalize {
			ss = ss.Canonical()
		}

		if s.Equals(ss) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func newCorruptedSegmented(rand *rand.Rand, label, maxLabel, numCorrupted int) traindata.SegmentedIndicesFeed {
	var corrupted []int32
	for _, i := range rand.Perm(maxLabel) {
		if i == label {
			continue
		}
		corrupted = append(corrupted, int32(i))
	}

	if len(corrupted) > numCorrupted {
		corrupted = corrupted[:numCorrupted]
	}

	return traindata.NewSegmentedIndicesFeed(corrupted...)
}

func newNodeIDFeed(nodes []*Node, nodeIDs nodeIDFn) traindata.SegmentedIndicesFeed {
	sif := traindata.SegmentedIndicesFeed{
		Indices:   make([]int32, 0, len(nodes)),
		SampleIDs: make([]int32, len(nodes)),
	}
	for _, n := range nodes {
		nid := n.ID
		if nodeIDs != nil {
			nid = nodeIDs(n)
		}
		sif.Indices = append(sif.Indices, int32(nid))
	}
	return sif
}

func joinNodes(nss ...[]*Node) []*Node {
	var base []*Node
	for _, ns := range nss {
		base = append(base, ns...)
	}
	return base
}

func max(as ...int32) int32 {
	if len(as) == 0 {
		return 0
	}
	m := as[0]
	for _, a := range as {
		if a > m {
			m = a
		}
	}
	return m
}

type nodeIDFn func(*Node) NodeID

func mustNodeIDFunc(nodeIDs map[*Node]NodeID) nodeIDFn {
	return func(n *Node) NodeID {
		if nid, ok := nodeIDs[n]; ok {
			return nid
		}
		panic(fmt.Sprintf("no id found for node %v", n))
	}
}

func sortSymbolByPopularity(syms []pythonresource.Symbol, rm pythonresource.Manager) []pythonresource.Symbol {
	if len(syms) == 1 {
		return syms
	}
	m := make(map[pythonimports.Hash]int)
	for _, s := range syms {
		ss := rm.SigStats(s)
		pop := 0
		if ss != nil {
			pop = ss.Count
		}
		m[s.Hash()] = pop
	}
	sort.Slice(syms, func(i, j int) bool {
		pi := m[syms[i].Hash()]
		pj := m[syms[j].Hash()]
		if pi == pj {
			return syms[i].Less(syms[j])
		}
		return pi > pj
	})
	return syms
}

func mostPopularSymbol(syms []pythonresource.Symbol, rm pythonresource.Manager) pythonresource.Symbol {
	syms = sortSymbolByPopularity(syms, rm)
	return syms[0]
}
