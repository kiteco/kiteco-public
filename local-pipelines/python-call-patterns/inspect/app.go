package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/templateset"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/diskmapindex"

	"github.com/kiteco/kiteco/local-pipelines/python-call-patterns/internal/data"
)

type app struct {
	rm        pythonresource.Manager
	idx       *diskmapindex.Index
	templates *templateset.Set
	patterns  data.PatternsByHash
	sources   *pythoncode.HashToSourceIndex
}

func (a *app) HandleHome(w http.ResponseWriter, r *http.Request) {
	err := a.templates.Render(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type result struct {
	Symbol     string            `json:"symbol"`
	Calls      string            `json:"calls"`
	Patterns   []renderedPattern `json:"patterns"`
	Signatures string            `json:"signatures"`
	Argspec    string            `json:"argspec"`
}

func (a *app) HandleSearch(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	query := params.Get("q")
	ls := params.Get("limit")
	if ls == "" {
		ls = "100"
	}

	limit, err := strconv.ParseInt(ls, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing %s to int: %v", ls, err), http.StatusBadRequest)
		return
	}

	syms, err := a.rm.PathSymbols(kitectx.Background(), pythonimports.NewDottedPath(query))
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to find symbols for path %s: %v", query, err), http.StatusNotFound)
		return
	}

	sym := syms[0]
	ps := pythonpatterns.Symbol{
		Dist: sym.Canonical().Dist(),
		Path: sym.Canonical().Path(),
	}

	calls, total, cerr := a.getCalls(ps, int(limit))
	pats, perr := a.getPatterns(sym.Canonical())
	res := result{
		Symbol:     sym.String(),
		Calls:      renderCalls(calls, total, cerr),
		Patterns:   renderPatterns(pats, perr),
		Signatures: renderPopularSignatures(a.rm.PopularSignatures(sym)),
		Argspec:    renderArgspec(a.rm.ArgSpec(sym)),
	}

	buf, err := json.Marshal(res)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) HandleSource(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	hash := params.Get("hash")

	var src string
	if a.sources == nil {
		src = "hash to source index not loaded"
	} else {
		buf, err := a.sources.SourceFor(hash)
		if err != nil {
			src = err.Error()
		} else {
			src = string(buf)
		}
	}

	err := a.templates.Render(w, "source.html", map[string]interface{}{
		"Hash":   hash,
		"Source": src,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *app) getCalls(sym pythonpatterns.Symbol, limit int) (data.Calls, int, error) {
	if a.idx == nil {
		return nil, 0, fmt.Errorf("calls not loaded")
	}

	bufs, err := a.idx.Get(sym.Hash().String())
	if err != nil {
		return nil, 0, err
	}

	var calls data.Calls
	var total int
	for _, buf := range bufs {
		c := new(data.Calls)
		fail(c.Decode(buf))

		cc := *c
		for _, call := range cc {
			if total >= limit {
				// Use ResevoirSampling to get an unbiased sampling of calls
				// https://en.wikipedia.org/wiki/Reservoir_sampling
				if j := rand.Intn(total + 1); j < limit {
					calls[j] = call
				}
			} else {
				calls = append(calls, call)
			}
			total++
		}
	}

	return calls, total, nil
}

func (a *app) getPatterns(sym pythonresource.Symbol) ([]pythonpatterns.Call, error) {
	if a.patterns == nil {
		return nil, fmt.Errorf("patterns not loaded")
	}

	p, ok := a.patterns[sym.Hash()]
	if !ok {
		return nil, fmt.Errorf("unable to find patterns for %v", sym)
	}

	return p.Patterns.Calls, nil
}
