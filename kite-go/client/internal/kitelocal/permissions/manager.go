package permissions

import (
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lang"
)

// Manager implements PermissionsManager for kite local
type Manager struct {
	langs       map[lang.Language]struct{}
	gotFilename func(string)
	m           sync.RWMutex
}

// NewManager returns a new manager which whitelists everything.
func NewManager(langs []lang.Language, gotFilename func(string)) *Manager {
	langMap := make(map[lang.Language]struct{}, len(langs))
	for _, l := range langs {
		langMap[l] = struct{}{}
	}

	return &Manager{
		langs:       langMap,
		gotFilename: gotFilename,
	}
}

// NewTestManager returns a permissions object that can be used for testing (won't persist state on disk, etc)
// you can pass nil to wrapperAuthorized to get a default wrapper which returns the input handlerFunc
func NewTestManager(langs ...lang.Language) *Manager {
	m := NewManager(langs, nil)
	return m
}

// Name implements component Core
func (m *Manager) Name() string {
	return "permissions"
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/languages", m.HandleLanguages).Methods("GET")
	mux.HandleFunc("/clientapi/support-status", m.HandleSupportStatus).Methods("GET")
	// check if file/directory is whitelisted and not ignored
	mux.HandleFunc("/clientapi/permissions/authorized", m.HandleAuthorized).Methods("GET")
}

// Filename will extract a filename from the request
func (m *Manager) Filename(r *http.Request) string {
	return filename(r)
}

// IsSupportedExtension returns whether the path is a suported extension
func (m *Manager) IsSupportedExtension(path string) component.SupportStatus {
	e := filepath.Ext(path)
	return supportMap[e]
}

// IsSupportedLangExtension returns whether the path is a suported extension of the languages
func (m *Manager) IsSupportedLangExtension(path string, langs map[lang.Language]struct{}) (bool, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return IsSupportedLangExtension(path, langs)
}

// AuthorizedReason values
const (
	ReasonOK              = "authorized"
	ReasonError           = "error"
	ReasonUnsupportedLang = "language not supported"
	ReasonUnsupportedFile = "file type not supported"
)

// AuthorizedReason returns why a file was or was not authorized
type AuthorizedReason string

// String implements Stringer
func (a AuthorizedReason) String() string {
	return string(a)
}

// Authorized is the top-level method to determine whether a file is allowed to be
// accessed based on the user's permissions. It returns a reason object for all paths,
// and returns a bool indicating whether the path is authorized.
func (m *Manager) Authorized(filename string) (AuthorizedReason, bool, error) {
	if filename == "" {
		return ReasonUnsupportedLang, false, nil
	}

	if _, err := sanitizePath(filename); err != nil {
		return ReasonError, false, err
	}

	if supported, err := m.IsSupportedLangExtension(filename, m.langs); err != nil {
		return ReasonError, false, err
	} else if !supported {
		return ReasonUnsupportedLang, false, nil
	}

	return ReasonOK, true, nil
}
