package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

const (
	epsilon        = 1e-9
	maxFailedSeeds = 1000
)

type app struct {
	res       *resources
	sessions  *sessionMgr
	templates *templateset.Set
}

func newApp(cache string) (*app, error) {
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		return nil, err
	}

	cs, err := newCodeStore(cache)
	if err != nil {
		return nil, err
	}

	res := &resources{
		store: cs,
		rm:    rm,
	}
	sessions := newSessionMgr(res)

	return &app{
		res:      res,
		sessions: sessions,
	}, nil
}

type interval struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

func (i interval) Covers(val float64) bool {
	return val > i.Low && val < i.High
}

func (i interval) Valid() error {
	if i.Low < 0 {
		return fmt.Errorf("partition.Low needs to be >= 0, got %f", i.Low)
	}

	if i.High > 1 {
		return fmt.Errorf("partition.High needs to be <= 1, got %f", i.High)
	}

	if math.Abs(i.High-i.Low) < epsilon || i.Low > i.High {
		return fmt.Errorf("need parition.Low < partition.High, got low %f >= high %f", i.Low, i.High)
	}

	return nil
}

type sessionResponse struct {
	Session int       `json:"session"`
	Samples []*sample `json:"samples"`
}

func (a *app) handleSessionsInfo(w http.ResponseWriter, r *http.Request) {
	stats := a.sessions.Stats()
	a.encode(w, stats)
}

type sessionPing struct {
	Session int `json:"session"`
}

func (a *app) handleSessionPing(w http.ResponseWriter, r *http.Request) {
	var req sessionPing
	if err := a.decode(r.Body, &req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	session := a.sessions.GetSession(sessionID(req.Session))
	if session == nil {
		http.Error(w, fmt.Sprintf("session not found for ID: %d", req.Session), http.StatusNotFound)
		return
	}

	session.Ping()

	w.WriteHeader(http.StatusOK)
}

func (a *app) handleSessionKill(w http.ResponseWriter, r *http.Request) {
	var req sessionRequest
	if err := a.decode(r.Body, &req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	if err := a.sessions.Kill(sessionID(req.Session)); err != nil {
		http.Error(w, fmt.Sprintf("error killing session: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type symbolRequest struct {
	Symbol       string                   `json:"symbol"`
	Offset       int                      `json:"offset"`
	Limit        int                      `json:"limit"`
	Context      pythoncode.SymbolContext `json:"context"`
	Canonicalize bool                     `json:"canonicalize"`
}

type symbolMetaInfo struct {
	TotalSources int `json:"total_sources"`
}

func (a *app) handleSymbolMetaInfo(w http.ResponseWriter, r *http.Request) {
	sr, ok := a.symbolRequestOrErr(w, r)
	if !ok {
		return
	}

	_, total, code, err := a.hashesFor(sr)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	a.encode(w, symbolMetaInfo{
		TotalSources: total,
	})
}

type symbolMember struct {
	Member string `json:"member"`
	Score  int    `json:"score"`
}

type symbolMembers struct {
	Members []symbolMember `json:"members"`
}

func (a *app) handleSymbolMembers(w http.ResponseWriter, r *http.Request) {
	sr, ok := a.symbolRequestOrErr(w, r)
	if !ok {
		return
	}

	symb, err := getSymbol(sr.Symbol, a.res.rm)
	if err != nil {
		http.Error(w, fmt.Sprintf("errro getting symbol %s: %v", sr.Symbol, err), http.StatusBadRequest)
		return
	}

	children, err := a.res.rm.Children(symb)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting children for %s: %v", symb.String(), err), http.StatusInternalServerError)
		return
	}

	members := make([]symbolMember, 0, len(children))
	for _, child := range children {
		cs, err := a.res.rm.ChildSymbol(symb, child)
		if err != nil {
			continue
		}

		score, err := a.symbolScore(cs, sr.Context, sr.Canonicalize)
		if err != nil {
			http.Error(w, fmt.Sprintf("error getting score for symbol %s: %v", child, err), http.StatusInternalServerError)
			return
		}

		members = append(members, symbolMember{
			Member: child,
			Score:  score,
		})
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].Score > members[j].Score
	})

	a.encode(w, symbolMembers{
		Members: members,
	})
}

type symbolSources struct {
	Sources []string `json:"sources"`
	Total   int      `json:"total"`
}

func (a *app) handleSymbolSources(w http.ResponseWriter, r *http.Request) {
	sr, ok := a.symbolRequestOrErr(w, r)
	if !ok {
		return
	}

	hashes, total, code, err := a.hashesFor(sr)
	if err != nil {
		http.Error(w, err.Error(), code)
	}

	srcs := make([]string, 0, len(hashes))

	for _, hash := range hashes {
		src, err := a.res.store.SourceFor(hash)
		if err != nil {
			log.Printf("unable to found source code for hash %s: %v\n", hash, err)
			src = []byte{}
		}

		srcs = append(srcs, string(src))
	}

	a.encode(w, symbolSources{
		Sources: srcs,
		Total:   total,
	})
}

type hashSourceRequest struct {
	Hash string `json:"hash"`
}

type hashSourceResponse struct {
	Source string `json:"source"`
}

func (a *app) handleHashSource(w http.ResponseWriter, r *http.Request) {
	var req hashSourceRequest
	if err := a.decode(r.Body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	src, err := a.res.store.SourceFor(req.Hash)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not get source for hash %s: %v", req.Hash, err), http.StatusNotFound)
		return
	}

	a.encode(w, hashSourceResponse{Source: string(src)})
}

type symbolScoresRequest struct {
	Symbols      []string                 `json:"symbols"`
	Context      pythoncode.SymbolContext `json:"context"`
	Canonicalize bool                     `json:"canonicalize"`
}

type symbolScoresResponse struct {
	Scores map[string]int    `json:"scores"`
	Errors map[string]string `json:"errors"`
}

func (a *app) handleSymbolScores(w http.ResponseWriter, r *http.Request) {
	var sr symbolScoresRequest
	if err := a.decode(r.Body, &sr); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	if sr.Context == "" {
		sr.Context = pythoncode.SymbolContextAttribute
	}

	if !sr.Context.Valid() {
		http.Error(w, fmt.Sprintf("invalid symbol context %s", sr.Context), http.StatusBadRequest)
		return
	}

	scores := make(map[string]int, len(sr.Symbols))
	errors := make(map[string]string)
	for _, sym := range sr.Symbols {
		s, err := a.res.rm.PathSymbol(pythonimports.NewDottedPath(sym))
		if err != nil {
			errors[sym] = fmt.Sprintf("error getting validated symbol for %s: %v", sym, err)
			continue
		}

		score, err := a.symbolScore(s, sr.Context, sr.Canonicalize)
		if err != nil {
			errors[sym] = fmt.Sprintf("error getting score for %s (%v): %v", sym, s, err)
			continue
		}

		scores[sym] = score
	}

	a.encode(w, symbolScoresResponse{
		Scores: scores,
		Errors: errors,
	})
}

type packageAndScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

func (a *app) handleSymbolPackages(w http.ResponseWriter, r *http.Request) {
	pkgs := a.res.rm.Pkgs()

	scores := make(map[string]int)
	for _, pkg := range pkgs {
		syms, err := a.res.rm.PathSymbols(kitectx.Background(), pythonimports.NewDottedPath(pkg))
		if err != nil {
			continue
		}

		for _, sym := range syms {
			// TODO: need to use the symbol counts since the symbol -> hash index only records direct
			// hits to the exact symbol so e.g import abc.foo does not count as a hit for abc
			score := a.res.rm.SymbolCounts(sym)
			if score == nil {
				continue
			}

			if score.Import > scores[pkg] {
				scores[pkg] = score.Import
			}
		}
	}

	var sorted []packageAndScore
	for sym, score := range scores {
		sorted = append(sorted, packageAndScore{
			Name:  sym,
			Score: score,
		})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	a.encode(w, sorted)
}

type symbolImportsResp struct {
	Imports [][]string `json:"imports"`
	Total   int        `json:"total"`
}

func (a *app) handleSymbolImports(w http.ResponseWriter, r *http.Request) {
	sr, ok := a.symbolRequestOrErr(w, r)
	if !ok {
		return
	}

	getSyms := func(val pythontype.Value) []pythonresource.Symbol {
		syms := a.getExternalSymbols(val)

		if !sr.Canonicalize {
			return syms
		}

		for i := range syms {
			syms[i] = syms[i].Canonical()
		}
		return syms
	}

	hashes, total, code, err := a.hashesFor(sr)
	if err != nil {
		http.Error(w, err.Error(), code)
	}

	jobs := make([]workerpool.Job, 0, len(hashes))
	completed := make([][]string, len(hashes))
	for i, h := range hashes {
		hc := h
		ic := i
		jobs = append(jobs, func() error {
			src, err := a.res.store.SourceFor(hc)
			if err != nil {
				return nil
			}

			if len(src) > maxFileBytes {
				return nil
			}

			in, err := getInputs(src, a.res)
			if err != nil {
				return nil
			}

			seen := make(map[string]struct{})
			pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
				if pythonast.IsNil(n) {
					return false
				}

				switch stmt := n.(type) {
				case *pythonast.ImportFromStmt:
					if stmt.Package == nil || len(stmt.Dots) > 0 {
						return false
					}

					pkg := in.RAST.References[stmt.Package]
					for _, sym := range getSyms(pkg) {
						seen[sym.PathString()] = struct{}{}
					}

					for _, name := range stmt.Names {
						val := in.RAST.References[name.External]
						for _, sym := range getSyms(val) {
							seen[sym.PathString()] = struct{}{}
						}
					}
					return false
				case *pythonast.ImportNameStmt:
					for _, clause := range stmt.Names {
						if clause == nil || len(clause.External.Names) == 0 {
							continue
						}

						val := in.RAST.References[clause.External]
						for _, sym := range getSyms(val) {
							seen[sym.PathString()] = struct{}{}
						}
					}

					return false
				case pythonast.Expr:
					return false
				default:
					return true
				}
			})

			imports := make([]string, 0, len(seen))
			for imp := range seen {
				imports = append(imports, imp)
			}

			sort.Strings(imports)

			completed[ic] = imports
			return nil
		})
	}

	pool := workerpool.New(5)
	pool.Add(jobs)
	defer pool.Stop() // make sure to always clean up the pool so that we do not leak go routines

	pool.Wait()

	var imports [][]string
	for _, imps := range completed {
		if len(imps) > 0 {
			imports = append(imports, imps)
		}
	}

	a.encode(w, symbolImportsResp{
		Total:   total,
		Imports: imports,
	})
}

//
// -- helpers
//

func (a *app) symbolRequestOrErr(w http.ResponseWriter, r *http.Request) (symbolRequest, bool) {
	var sr symbolRequest
	if err := a.decode(r.Body, &sr); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return symbolRequest{}, false
	}

	if sr.Limit == 0 {
		sr.Limit = 10
	}

	if sr.Context == "" {
		// backwards compatibility
		sr.Context = pythoncode.SymbolContextAttribute
	}

	if !sr.Context.Valid() {
		http.Error(w, fmt.Sprintf("invalid symbol context %s", sr.Context), http.StatusBadRequest)
		return symbolRequest{}, false
	}

	return sr, true
}

func (a *app) decode(r io.Reader, v interface{}) error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("json decode error: %v", err)
	}
	return nil
}

func (a *app) encode(w http.ResponseWriter, v interface{}) {
	buf, err := json.Marshal(v)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) hashesFor(sr symbolRequest) ([]string, int, int, error) {
	symb, err := getSymbol(sr.Symbol, a.res.rm)
	if err != nil {
		return nil, 0, http.StatusBadRequest, fmt.Errorf("unable to find symbol %s: %v", sr.Symbol, err)
	}

	hashes, err := a.res.store.HashesFor(symb, sr.Canonicalize, true)
	if err != nil {
		return nil, 0, http.StatusInternalServerError, fmt.Errorf("error getting hashes for %s: %v", sr.Symbol, err)
	}

	if len(hashes) == 0 {
		return nil, 0, http.StatusOK, nil
	}

	var hashStrs []string
	for _, hash := range hashes {
		if ShouldUseHash(hash, sr.Context) {
			hashStrs = append(hashStrs, hash.Hash)
		}
	}

	if sr.Offset > len(hashStrs) {
		return nil, 0, http.StatusBadRequest, fmt.Errorf("invalid request offset %d >= %d", sr.Offset, len(hashes))
	}

	total := len(hashStrs)

	hashStrs = hashStrs[sr.Offset:]
	if sr.Limit > 0 && len(hashStrs) > sr.Limit {
		hashStrs = hashStrs[:sr.Limit]
	}

	return hashStrs, total, http.StatusOK, nil
}

// symbolScore returns the score for a given symbol, which is a function of the hashes/counts available for that symbol.
func (a *app) symbolScore(sym pythonresource.Symbol, sc pythoncode.SymbolContext, canonicalize bool) (int, error) {
	hashes, err := a.res.store.HashesFor(sym, canonicalize, false)
	if err != nil {
		return 0, err
	}

	var count int
	for _, hash := range hashes {
		count += int(hash.Counts.CountFor(sc))
	}

	return count, nil
}

func (a *app) getExternalSymbols(val pythontype.Value) []pythonresource.Symbol {
	val = pythontype.Translate(kitectx.Background(), val, a.res.rm)
	if val == nil {
		return nil
	}

	var syms []pythonresource.Symbol
	for _, val := range pythontype.DisjunctsNoCtx(val) {
		switch val := val.(type) {
		case pythontype.External:
			syms = append(syms, val.Symbol())
		case pythontype.ExternalInstance:
			syms = append(syms, val.TypeExternal.Symbol())
		}
	}

	return syms
}

// ShouldUseHash returns true if we want to use the specific file hash to generate training data
func ShouldUseHash(counts pythoncode.HashCounts, sc pythoncode.SymbolContext) bool {
	return counts.Counts.CountFor(sc) > 0
}
