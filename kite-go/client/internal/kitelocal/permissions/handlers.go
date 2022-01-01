package permissions

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
)

// HandleLanguages handles returning the set of supported languages
func (m *Manager) HandleLanguages(w http.ResponseWriter, r *http.Request) {
	var resp []string
	for lang := range m.langs {
		resp = append(resp, lang.Name())
	}

	sort.Strings(resp)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleSupportStatus returns whether or not the given file extension is supported
func (m *Manager) HandleSupportStatus(w http.ResponseWriter, r *http.Request) {
	fn := filename(r)
	if fn == "" {
		http.Error(w, "must include native absolute filepath", http.StatusBadRequest)
		return
	}
	e := filepath.Ext(fn)
	res := supportMap[e]

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleAuthorized handles checking if kite is authorized for a file/directory,
// i.e. the file/directory is whitelisted and not ignored
func (m *Manager) HandleAuthorized(w http.ResponseWriter, r *http.Request) {
	fn := filename(r)
	if fn == "" {
		http.Error(w, "must include native absolute filepath", http.StatusBadRequest)
		return
	}

	if m.gotFilename != nil {
		m.gotFilename(fn)
	}

	reason, _, err := m.Authorized(fn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if reason == ReasonUnsupportedLang {
		http.Error(w, reason.String(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// WrapAuthorizedFile returns a new http HandlerFunc which first checks for authorized
// before passing control to the original handler
func (m *Manager) WrapAuthorizedFile(handler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fn := filename(r); fn != "" {
			if m.gotFilename != nil {
				m.gotFilename(fn)
			}
			reason, ok, err := m.Authorized(fn)
			switch {
			case err != nil:
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			case !ok:
				http.Error(w, reason.String(), http.StatusForbidden)
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}
