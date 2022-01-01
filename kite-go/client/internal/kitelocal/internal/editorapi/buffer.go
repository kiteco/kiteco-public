package editorapi

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

type requestBuffer struct {
	Buffer string `json:"buffer"`
}

func getCursorCallee(params url.Values, bufferBytes []byte) int {
	// ignore error here; it'll get picked up by the driver handler, where tracking/logging happens
	cursor, _ := webutils.ParseByteOrRuneOffset(bufferBytes, params.Get("offset_bytes"), params.Get("offset_runes"))
	return cursor
}

func getCursorHover(params url.Values, bufferBytes []byte) int {
	var cursor int
	// ignore error here; it'll get picked up by the driver handler, where tracking/logging happens
	if params.Get("cursor_bytes") == "" && params.Get("cursor_runes") == "" {
		// old style selection query; use begin as cursor position
		cursor, _ = webutils.ParseByteOrRuneOffset(bufferBytes, params.Get("selection_begin_bytes"), params.Get("selection_begin_runes"))
	} else if params.Get("offset_encoding") != "" {
		// Use ParseOffsetToUTF8 if offset_encoding param is present
		cursor, _ = webutils.ParseOffsetToUTF8(bufferBytes, params.Get("cursor_runes"), params.Get("offset_encoding"))
	} else {
		cursor, _ = webutils.ParseByteOrRuneOffset(bufferBytes, params.Get("cursor_bytes"), params.Get("cursor_runes"))
	}
	return cursor
}

// HandleBuffer routes a request to a running driver that is buffer specific
func (m *Manager) handleBuffer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	editor := vars["editor"]
	if editor == "" {
		http.Error(w, "editor was empty", http.StatusBadRequest)
		return
	}

	filename := vars["filename"]
	if filename == "" {
		http.Error(w, "filename was empty", http.StatusBadRequest)
		return
	}

	state := vars["state"]
	if state == "" {
		http.Error(w, "state was empty", http.StatusBadRequest)
		return
	}

	reqType := vars["reqType"]
	switch reqType {
	case "hover", "callee":
	default:
		http.Error(w, "unhandled request type", http.StatusNotFound)
		return
	}

	// look up the driver for this path
	filename = strings.Replace(filename, ":", "/", -1)

	// This is also the only place the driver.Provider API is called with an unix
	// path. This breaks windows because driver.Provider attempts to convert to a
	// unix path, and unix path conversion is not idempotent. So we convert it back to a regular path first.
	var err error
	filename, err = localpath.FromUnix(filename)
	if err != nil {
		http.Error(w, "path error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fix for windows case sensitivity issue.
	if strings.HasPrefix(filename, "/windows/") {
		filename = strings.ToLower(filename)
	}

	// Consume body if exists, faily gracefully (i.e don't expect a body to exist)
	var reader io.Reader
	if r.Header.Get("Content-Encoding") == "gzip" {
		// NOTE: we don't want to log the error, so we're just gonna swallow it here...
		reader, _ = gzip.NewReader(r.Body)
	} else {
		reader = r.Body
	}

	var req requestBuffer
	if reader != nil {
		// NOTE: we don't want to log the error, so we're just gonna swallow it here...
		if err := json.NewDecoder(reader).Decode(&req); err != nil {
			req.Buffer = ""
		}
	}
	bufferBytes := []byte(req.Buffer)

	var cursor int
	switch reqType {
	case "hover":
		cursor = getCursorHover(r.URL.Query(), bufferBytes)
	case "callee":
		cursor = getCursorCallee(r.URL.Query(), bufferBytes)
	default:
		// rollbar, since this case was checked above
		rollbar.Error(errors.New("unhandled request type"), reqType)
		http.Error(w, "unhandled request type", http.StatusNotFound)
		return
	}

	var f *driver.State
	var exists bool
	err = kitectx.FromContext(r.Context(), func(ctx kitectx.Context) error {
		f, exists = m.provider.Driver(ctx, filename, editor, state)
		if !exists {
			if len(req.Buffer) > 0 {
				// Create file driver from content if it doesn't exist
				f = m.provider.DriverFromContent(ctx, filename, editor, req.Buffer, cursor)
			} else {
				return errors.New("file not found: " + filename)
			}
		}
		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if f.BufferHandler == nil {
		http.Error(w, fmt.Sprintf("%T does not handle http requests", f.FileDriver), http.StatusNotFound)
		return
	}

	// send the HTTP request to the driver
	f.BufferHandler.ServeHTTP(w, r)
}
