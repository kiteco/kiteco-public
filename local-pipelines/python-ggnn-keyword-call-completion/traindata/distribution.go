package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

const minScore = 10

type scores map[string]int

type scoresRequest struct {
	Symbols []string                 `json:"symbols"`
	Context pythoncode.SymbolContext `json:"context"`
}

type scoresResponse struct {
	Scores map[string]int    `json:"scores"`
	Errors map[string]string `json:"errors"`
}

func getScores(funcs []pythonresource.Symbol, endpoint string) (scores, error) {
	req := scoresRequest{
		Symbols: make([]string, 0, len(funcs)),
		Context: pythoncode.SymbolContextCallFunc,
	}

	for _, f := range funcs {
		req.Symbols = append(req.Symbols, f.PathString())
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, fmt.Errorf("error encoding request: %v", err)
	}

	resp, err := http.Post(endpoint, "application/json", &buf)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("got code %d: %v", resp.StatusCode, string(buf))
	}

	var sr scoresResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	if len(sr.Errors) > 0 {
		log.Printf("got %d errors\n", len(sr.Errors))

		var parts []string
		for s, e := range sr.Errors {
			parts = append(parts, fmt.Sprintf("sym: %s with err: %s", s, e))
		}

		log.Println(strings.Join(parts, ", "))
	}

	for sym, score := range sr.Scores {
		if score < minScore {
			delete(sr.Scores, sym)
		}
	}

	return scores(sr.Scores), nil
}

type walker struct {
	rm    pythonresource.Manager
	funcs []pythonresource.Symbol
	seen  map[pythonimports.Hash]bool
}

func (w *walker) walk(tl string, sym pythonresource.Symbol) error {
	children, err := w.rm.Children(sym)
	if err != nil {
		return fmt.Errorf("error getting children for %s: %v", sym.PathString(), err)
	}
	sort.Strings(children)

	for _, child := range children {
		cs, err := w.rm.ChildSymbol(sym, child)
		if err != nil {
			// happens for symbols that are not walkable
			// like __bases__[%d] or non walkable class members
			continue
		}
		cs = cs.Canonical()

		if w.seen[cs.PathHash()] || cs.Path().Head() != tl {
			continue
		}
		w.seen[cs.PathHash()] = true

		switch w.rm.Kind(cs) {
		case keytypes.ModuleKind, keytypes.TypeKind:
			if err := w.walk(tl, cs); err != nil {
				return err
			}
		case keytypes.FunctionKind:
			w.funcs = append(w.funcs, cs)
		}
	}

	return nil
}

func (w *walker) Walk(tl string) error {
	sym, err := w.rm.PathSymbol(pythonimports.NewDottedPath(tl))
	if err != nil {
		return fmt.Errorf("unable to find toplevel %s: %v", tl, err)
	}
	sym = sym.Canonical()

	w.seen[sym.PathHash()] = true

	return w.walk(tl, sym)
}
