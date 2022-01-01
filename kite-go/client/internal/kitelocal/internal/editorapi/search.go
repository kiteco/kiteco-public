package editorapi

import (
	"context"
	"net/http"
)

func (m *Manager) handleSearch(w http.ResponseWriter, r *http.Request) {
	// If we're online, forward the request to the backend
	// NB: we want a pretty quick timeout for this, due to the expected responsiveness of search
	ctx, cancel := context.WithTimeout(r.Context(), responsiveRemoteTimeout)
	defer cancel()

	// we also want a quick timeout here, in case m.network.Online()
	// and it hasn't yet been updated
	m.docs.ServeHTTP(w, r.WithContext(ctx))
	return
}
