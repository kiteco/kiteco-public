package editorapi

import (
	"context"
	"net/http"
)

func (m *Manager) handleCuration(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), responsiveRemoteTimeout)
	defer cancel()

	m.docs.ServeHTTP(w, r.WithContext(ctx))
	return
}
