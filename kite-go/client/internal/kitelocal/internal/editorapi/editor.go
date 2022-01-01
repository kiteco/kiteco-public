package editorapi

import (
	"net/http"

	"github.com/gorilla/mux"
	backend_editorapi "github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
)

func (m *Manager) handleEditor(w http.ResponseWriter, r *http.Request) {
	sid := mux.Vars(r)["id"]
	id := backend_editorapi.ParseID(sid)

	addr, _, err := pythonenv.ParseLocator(id.LanguageSpecific())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If the id is local, use the local handler
	isLocal := addr.User > 0 || len(addr.Machine) > 0 || len(addr.File) > 0
	if isLocal {
		m.editorAPIHandler.ServeHTTP(w, r)
		return
	}

	// If the global symbol is available locally, serve locally
	dp := addr.Path
	sym, err := m.services.ResourceManager.PathSymbol(dp)
	if err == nil && m.services.ResourceManager.DistLoaded(sym.Dist()) {
		m.editorAPIHandler.ServeHTTP(w, r)
		return
	}

	/* Removed for now due to https://github.com/kiteco/kiteco/issues/11796
	// Forward to the backend if we're online
	if m.network.Online() {
		m.docs.ServeHTTP(w, r)
		return
	}
	*/

	// Otherwise use local handler (which might 404)
	m.editorAPIHandler.ServeHTTP(w, r)
	return
}
