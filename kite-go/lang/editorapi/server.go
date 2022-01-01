package editorapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/gziphttp"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	defaultOffset = 0
	defaultLimit  = 10
)

// Server handles serving the editor endpoints.
type Server struct {
	endpoints map[lang.Language]Endpoint
	router    *mux.Router
}

// NewServer returns a new server using the provided endpoints.
func NewServer(endpoints ...Endpoint) *Server {
	server := &Server{
		endpoints: make(map[lang.Language]Endpoint, len(endpoints)),
		router:    mux.NewRouter(),
	}

	for _, endpoint := range endpoints {
		server.endpoints[endpoint.Language()] = endpoint
	}

	editor := server.router.PathPrefix("/api/editor/").Subrouter()
	editor.HandleFunc("/value/{id}", gziphttp.Wrap(server.HandleValueReport))
	editor.HandleFunc("/value/{id}/members", gziphttp.Wrap(server.HandleValueMembers))
	editor.HandleFunc("/value/{id}/usages", gziphttp.Wrap(server.HandleValueUsages)) // TODO(naman) deprecated; rm
	editor.HandleFunc("/value/{id}/links", gziphttp.Wrap(server.HandleValueLinks))
	editor.HandleFunc("/value/{id}/definition-source", gziphttp.Wrap(server.HandleValueDefinition)) // TODO(naman) deprecated; rm
	editor.HandleFunc("/symbol/{id}", gziphttp.Wrap(server.HandleSymbolReport))
	editor.HandleFunc("/search", gziphttp.Wrap(server.HandleSearch))

	return server
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// HandleValueReport handles serving the value report endpoint.
// URL:
//   GET /api/editor/value/{id}
// Response Codes
//   * 200 -- Success
//   * 400 -- Malformed request
//   * 404 -- No results found
//   * 500 -- Internal error (usually JSON related)
//   * 501 -- Feature not implemented
//   * 503 -- Backend has not analyzed code yet (temporary condition)
func (s *Server) HandleValueReport(w http.ResponseWriter, r *http.Request) {
	id, code, err := s.id(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	ep := s.endpoints[id.lang]
	if ep == nil {
		http.Error(w, fmt.Sprintf("no endpoint for id %v", id), http.StatusNotFound)
		return
	}

	var resp *ReportResponse
	code = http.StatusInternalServerError
	err = kitectx.FromContext(r.Context(), func(ctx kitectx.Context) (err error) {
		resp, code, err = ep.ValueReport(ctx, id.id)
		return
	})
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	s.write(w, resp)
}

// HandleSymbolReport handles serving the symbol report endpoint.
// URL:
//   GET /api/editor/symbol/{id}
// Response Codes
//   * 200 -- Success
//   * 400 -- Malformed request
//   * 404 -- No results found
//   * 500 -- Internal error (usually JSON related)
//   * 501 -- Feature not implemented
//   * 503 -- Backend has not analyzed code yet (temporary condition)
func (s *Server) HandleSymbolReport(w http.ResponseWriter, r *http.Request) {
	id, code, err := s.id(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	ep := s.endpoints[id.lang]
	if ep == nil {
		http.Error(w, fmt.Sprintf("no endpoint for id %v", id), http.StatusNotFound)
		return
	}

	var resp *ReportResponse
	code = http.StatusInternalServerError
	err = kitectx.FromContext(r.Context(), func(ctx kitectx.Context) (err error) {
		resp, code, err = ep.SymbolReport(ctx, id.id)
		return
	})
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	s.write(w, resp)
}

// HandleValueMembers handles serving the value members endpoint.
// URL:
//   GET /api/editor/value/{id}/members?offset={offset}&limit={limit}
// Response Codes
//   * 200 -- Success
//   * 400 -- Malformed request
//   * 404 -- No results found
//   * 500 -- Internal error (usually JSON related)
//   * 501 -- Feature not implemented
//   * 503 -- Backend has not analyzed code yet (temporary condition)
func (s *Server) HandleValueMembers(w http.ResponseWriter, r *http.Request) {
	id, code, err := s.id(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	ep := s.endpoints[id.lang]
	if ep == nil {
		http.Error(w, fmt.Sprintf("no endpoint for id %v", id), http.StatusNotFound)
		return
	}

	offset, limit, code, err := s.offsetAndLimit(r, defaultOffset, defaultLimit)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	//get desired response type
	switch r.URL.Query().Get("type") {
	case "symbolext", "":
		var resp *MembersExtResponse
		code := http.StatusInternalServerError
		err := kitectx.FromContext(r.Context(), func(ctx kitectx.Context) (err error) {
			resp, code, err = ep.ValueMembersExt(ctx, id.id, offset, limit)
			return
		})
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		s.write(w, resp)
	case "symbol":
		var resp *MembersResponse
		code := http.StatusInternalServerError
		err := kitectx.FromContext(r.Context(), func(ctx kitectx.Context) (err error) {
			resp, code, err = ep.ValueMembers(ctx, id.id, offset, limit)
			return
		})
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		s.write(w, resp)
	default:
		http.Error(w, fmt.Sprintf("invalid type: %s", r.URL.Query().Get("type")), http.StatusBadRequest)
		return
	}
}

// HandleValueUsages handles serving the value usages endpoint.
// URL:
//   GET /api/editor/value/{id}/usages?offset={offset}&limit={limit}
// Response Codes
//   * 404 -- DEPRECATED
// TODO(naman) deprecated; rm
func (s *Server) HandleValueUsages(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "deprecated", http.StatusNotFound)
}

// HandleValueLinks handles serving the value links endpoint.
// URL:
//   GET /api/editor/value/{id}/links?offset={offset}&limit={limit}
// Response Codes
//   * 410 -- DEPRECATED
func (s *Server) HandleValueLinks(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "deprecated", http.StatusGone)
}

// HandleValueDefinition handles serving the definition source endpoint.
// URL:
//   GET /api/editor/value/{id}/definition-source
// Response Codes
//   * 404 -- DEPRECATED
// TODO(naman) deprecated; rm
func (s *Server) HandleValueDefinition(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "deprecated", http.StatusNotFound)
}

// HandleSearch handles serving the search endpoint.
// URL:
//   GET /api/editor/search?q={query}&offset={offset}&limit={limit}
// Response Codes
//   * 200 -- Success
//   * 400 -- Malformed request
//   * 404 -- No results found
//   * 500 -- Internal error (usually JSON related)
//   * 501 -- Feature not implemented
func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "no query", http.StatusBadRequest)
		return
	}

	offset, limit, code, err := s.offsetAndLimit(r, defaultOffset, defaultLimit)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	// for now we assume that search queries are for python
	ep := s.endpoints[lang.Python]
	if ep == nil {
		http.Error(w, "no python endpoint", http.StatusNotFound)
		return
	}

	var resp *SearchResults
	code = http.StatusInternalServerError
	err = kitectx.FromContext(r.Context(), func(ctx kitectx.Context) (err error) {
		resp, code, err = ep.Search(ctx, query, offset, limit)
		return
	})
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	s.write(w, resp)
}

func (s *Server) write(w http.ResponseWriter, resp interface{}) {
	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (s *Server) id(r *http.Request) (ID, int, error) {
	sid := mux.Vars(r)["id"]
	id := ParseID(sid)
	if id.LanguageSpecific() == "" {
		// TODO(juan): Hack to fix issues with interlinking in code examples and documentation.
		// SEE: https://github.com/kiteco/kiteco/issues/4769
		if sid == "" {
			return ID{}, http.StatusBadRequest, fmt.Errorf("request with empty id `%s`", sid)
		}
		id = ID{
			lang: lang.Python,
			id:   sid,
		}
	}

	if id.lang == lang.Unknown {
		return ID{}, http.StatusBadRequest, fmt.Errorf("request with id %s contains an unknown language", sid)
	}

	return id, http.StatusOK, nil
}

func (s *Server) offsetAndLimit(r *http.Request, defaultOffset, defaultLimit int) (int, int, int, error) {
	q := r.URL.Query()

	var err error
	if offstr := q.Get("offset"); offstr != "" {
		if defaultOffset, err = strconv.Atoi(offstr); err != nil {
			return 0, 0, http.StatusBadRequest, fmt.Errorf("invalid offset: %v", err)
		}
		if defaultOffset < 0 {
			return 0, 0, http.StatusBadRequest, fmt.Errorf("negative offset")
		}
	}

	if numstr := q.Get("limit"); numstr != "" {
		if defaultLimit, err = strconv.Atoi(numstr); err != nil {
			return 0, 0, http.StatusBadRequest, fmt.Errorf("invalid limit: %v", err)
		}
		if defaultLimit < 0 {
			return 0, 0, http.StatusBadRequest, fmt.Errorf("negative limit")
		}
	}
	return defaultOffset, defaultLimit, http.StatusOK, nil
}
