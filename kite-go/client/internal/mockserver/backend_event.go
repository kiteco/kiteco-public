package mockserver

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/response"
)

// handles /http/events
func (s *MockBackendServer) handleEditorEvent(w http.ResponseWriter, r *http.Request) {
	authorized := r.Header.Get("Kite-Token") != ""

	reader := r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		var err error
		if reader, err = gzip.NewReader(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	var evt event.Event
	if err := json.NewDecoder(reader).Decode(&evt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := response.Root{}
	resp.Hash = evt.GetTextMD5()
	for _, sel := range evt.GetSelections() {
		if sel.GetStart() == sel.GetEnd() {
			resp.Cursor = sel.GetStart()
		}
	}

	switch evt.GetAction() {
	case "skip", "ping", "surface":
		http.Error(w, fmt.Sprintf("got unexpected event action: %s", evt.GetAction()), 500)
	default:
		resp.Type = response.Passive
		resp.Filename = evt.GetFilename()
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
	}
	w.Write(buf)

	s.IncrementRequestCount("editorEvent")
}

// EditorEventRequestCount returns how many requests to /http/events were received
func (s *MockBackendServer) EditorEventRequestCount() int64 {
	return s.GetRequestCount("editorEvent")
}
