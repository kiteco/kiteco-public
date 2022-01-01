package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/data"
)

// AttrDist ...
type AttrDist map[string]float64

// Normalize d by the provided sum
func (d AttrDist) Normalize(sum float64) {
	invSum := 1. / sum
	for a := range d {
		d[a] *= invSum
	}
}

// Type ...
type Type struct {
	Pkg   string
	Dist  keytypes.Distribution
	Sym   pythonresource.Symbol
	Attrs AttrDist
	Prob  float64
}

// TypeAndAttrs ...
type TypeAndAttrs struct {
	Type  data.Symbol
	Attrs AttrDist
}

func loadTypeAndAttrs(rm pythonresource.Manager, pkgs map[string]bool, attrsDistPath string) map[pythonimports.Hash]AttrDist {
	pretrained := make(map[pythonimports.Hash]AttrDist)
	for p := range pkgs {
		depFileName := pathForPkg(attrsDistPath, p)
		if _, err := os.Stat(depFileName); err == nil {
			f, err := os.Open(depFileName)
			fail(err)

			var dist []TypeAndAttrs
			fail(json.NewDecoder(f).Decode(&dist))
			f.Close()

			fmt.Printf("Loading pretrained attribute distribution from %s\n", depFileName)
			for _, v := range dist {
				sym := mustSymbol(rm, v.Type)
				if old, ok := pretrained[sym.PathHash()]; ok {
					logf(logLevelSevere, "entry already exists in pretrained: symbol %s with attr dist %v\n", v.Type, old)
				}
				pretrained[sym.PathHash()] = v.Attrs
			}
		}
	}
	return pretrained
}

// String representation of t
func (t *Type) String() string {
	attrs := make([]string, 0, len(t.Attrs))
	for attr := range t.Attrs {
		attrs = append(attrs, attr)
	}
	sort.Strings(attrs)

	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		parts = append(parts, fmt.Sprintf("%s (%f)", attr, t.Attrs[attr]))
	}

	return fmt.Sprintf("{sym: %s (%f) attrs: [%s]}",
		t.Sym.Path().String(), t.Prob, strings.Join(parts, ","))
}

// TypeDist ...
type TypeDist map[*Type]float64

// SetAll values in the distribution
// to the provided one and return the sum
// the values
func (d TypeDist) SetAll(v float64) float64 {
	var sum float64
	for t := range d {
		sum += v
		d[t] = v
	}
	return sum
}

// Normalize the dist by the provided sum
func (d TypeDist) Normalize(sum float64) {
	invSum := 1. / sum
	for t := range d {
		d[t] *= invSum
	}
}

// Func ...
type Func struct {
	Sym    pythonresource.Symbol
	Dist   keytypes.Distribution
	Return TypeDist
	Prob   float64
	Usages []*Variable
}

// String representation of f
func (f *Func) String() string {
	return f.Sym.PathString()
}

// Variable ...
type Variable struct {
	Func  *Func
	Types TypeDist
	Attrs []string
}

// String representation of v
func (v *Variable) String() string {
	return fmt.Sprintf("{func: %s, attrs: [%s]}",
		v.Func.Sym.Canonical().String(), strings.Join(v.Attrs, ","),
	)
}

type model struct {
	funcs     []*Func
	types     []*Type
	variables []*Variable
}

type funcAndAttrs struct {
	Func  *Func
	Attrs map[string]int
}

func newModel(samples []data.Sample, rm pythonresource.Manager, pkgs map[string]bool, minCount int, pretrainedAttrs map[pythonimports.Hash]AttrDist, maxAttrs int) *model {
	m := &model{
		// initializes P(a|t) to the uniform distribution
		types: buildTypes(pkgs, rm, minCount, pretrainedAttrs),
	}

	fas := make(map[pythonimports.Hash]*funcAndAttrs)

	var skippedSamples, skippedFuncs int
	for _, s := range samples {

		// because of the way EM works, the initial variable distributions
		// do not actual matter, these are just auxiliary distributions which are
		// set on the first E step, SEE: model.EStep and model.EM.
		vts := make(TypeDist, len(m.types))
		fs := mustSymbol(rm, s.Func)

		for _, t := range m.types {
			// The type has to be from the given package or its dependencies, has to be consistent in terms of attrs
			if pkgs[t.Pkg] && consistentWithType(s.Attrs, t) {
				vts[t] = 1.
			}
		}

		if len(vts) == 0 {
			skippedSamples++
			logf(logLevelWarn, "skipping inconsistent sample for func: %s attrs: %s\n",
				s.Func.Path.String(), strings.Join(s.Attrs, ","),
			)
			continue
		}

		// Sample from Attrs to avoid P(a|t) being zero after multiplying by too many times.
		sampledAttrs := sampleAttrs(s.Attrs, maxAttrs)

		hash := fs.Hash()

		fa := fas[hash]
		if fa == nil {
			// NOTE: we do not append fa.func to m.funcs
			// here because we want to make sure that
			// the samples for the function are consistent (see below)
			fa = &funcAndAttrs{
				Func: &Func{
					Sym:  fs,
					Dist: fs.Dist(),
				},
				Attrs: make(map[string]int),
			}
			fas[hash] = fa
		}

		for _, attr := range sampledAttrs {
			fa.Attrs[attr]++
		}

		v := &Variable{
			Func:  fa.Func,
			Types: vts,
			Attrs: sampledAttrs,
		}

		fa.Func.Usages = append(fa.Func.Usages, v)
	}

	// initialize P(t|f)
	hashes := make([]pythonimports.Hash, 0, len(fas))
	for h := range fas {
		hashes = append(hashes, h)
	}

	sort.Slice(hashes, func(i, j int) bool {
		return hashes[i] < hashes[j]
	})

	for _, h := range hashes {
		fa := fas[h]
		f := fa.Func

		f.Return = make(TypeDist)

		truths := rm.TruthyReturnTypes(f.Sym)
		truthMap := make(map[pythonimports.Hash]bool)

		for _, truth := range truths {
			// Not include the ones that only come from EM model
			if truth.Truthiness.String() != "emmodel" {
				truthMap[truth.Symbol.PathHash()] = true
			}
		}

		var sum float64
		for _, t := range m.types {
			if s := returnTypeScore(fa, t); s > eps {
				if truthMap[t.Sym.PathHash()] {
					s *= 5
				}
				sum += s
				f.Return[t] = s
			}
		}

		if len(f.Return) == 0 {
			skippedFuncs++
			logf(logLevelWarn, "skipping func %s the %d usages were not consistent\n", f.String(), len(f.Usages))
			continue
		}

		// normalize
		invSum := 1. / sum
		for t := range f.Return {
			f.Return[t] *= invSum
		}

		m.funcs = append(m.funcs, f)
		m.variables = append(m.variables, f.Usages...)
	}

	// Re-calculate P(f) after filtering the functions
	for _, f := range m.funcs {
		f.Prob = float64(len(f.Usages)) / float64(len(m.variables))
	}

	fmt.Printf("Skipped %d samples and %d funcs that were not consistent\n", skippedSamples, skippedFuncs)

	return m
}

func returnTypeScore(fa *funcAndAttrs, t *Type) float64 {
	var count int
	for _, usage := range fa.Func.Usages {
		if consistentWithType(usage.Attrs, t) {
			count++
		}
	}

	return float64(count) * t.Prob
}

func mustSymbol(rm pythonresource.Manager, s data.Symbol) pythonresource.Symbol {
	sym, err := rm.NewSymbol(s.Dist, s.Path)
	fail(err)
	return sym.Canonical()
}

func consistentWithType(attrs []string, t *Type) bool {
	for _, attr := range attrs {
		if _, ok := t.Attrs[attr]; !ok {
			return false
		}
	}
	return true
}

func sampleAttrs(attrs []string, maxAttrs int) []string {
	if len(attrs) > maxAttrs {
		attrCounts := make(map[string]int)
		for _, a := range attrs {
			attrCounts[a]++
		}

		var sampled []string
		// Make sure at least every unique attribute gets sampled
		// Since it would be more useful to determine the type
		for a := range attrCounts {
			sampled = append(sampled, a)
			attrCounts[a]--
		}

		rand.Shuffle(len(attrs), func(i, j int) {
			attrs[i], attrs[j] = attrs[j], attrs[i]
		})

		if len(sampled) < maxAttrs {
			sampled = append(sampled, attrs[:(maxAttrs-len(sampled))]...)
		}
		return sampled
	}
	return attrs
}
