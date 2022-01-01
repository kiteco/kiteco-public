package python

import (
	"encoding/json"
	"fmt"
	"go/token"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/gziphttp"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/status"
)

func wrapCtx(f func(kitectx.Context, http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// ignore any resulting error; if there is one, the request has timed out, so there's nothing to do
		_ = kitectx.FromContext(r.Context(), func(ctx kitectx.Context) error {
			f(ctx, w, r)
			return nil
		})
	}
}

// DriverEndpoint provides access to the state of a UnifiedDriver via HTTP.
type DriverEndpoint struct {
	d      *UnifiedDriver
	editor *editorServices
	router *mux.Router
}

// NewDriverEndpoint creates a new driver for the given endpoint
func NewDriverEndpoint(d *UnifiedDriver) *DriverEndpoint {
	s := DriverEndpoint{
		d:      d,
		editor: newEditorServices(d.python),
		router: mux.NewRouter(),
	}
	r := s.router.PathPrefix("/api/buffer/{editor}/{filename}/{state}/").Subrouter()
	r.HandleFunc("/hover", gziphttp.Wrap(status.RecordStatusCode(wrapCtx(s.handleHover), hoverStatusCode)))
	r.HandleFunc("/callee", gziphttp.Wrap(status.RecordStatusCode(wrapCtx(s.handleCallee), calleeStatusCode)))

	return &s
}

// ServeHTTP implements http.Handler
func (s *DriverEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *DriverEndpoint) handleHover(ctx kitectx.Context, w http.ResponseWriter, r *http.Request) {
	ctx.CheckAbort()

	hoverCounter.Add(1)
	defer hoverDuration.DeferRecord(time.Now())

	// Note: this function holds a read-only lock, so it must not modify the driver state
	s.d.lock.RLock()
	defer s.d.lock.RUnlock()

	pyctx := s.d.lastContext
	if pyctx == nil {
		hoverFailReason.HitAndAdd("no context available")
		http.Error(w, "driver.lastContext is nil", http.StatusServiceUnavailable)
		return
	}

	var begin, end int64
	params := r.URL.Query()
	if params.Get("cursor_bytes") == "" && params.Get("cursor_runes") == "" {
		// old style selection query
		hoverSelectionCounter.Add(1)
		b, err := webutils.ParseByteOrRuneOffset(pyctx.ContextInputs.Buffer,
			params.Get("selection_begin_bytes"),
			params.Get("selection_begin_runes"))
		if err != nil {
			hoverFailReason.HitAndAdd("error parsing query param")
			http.Error(w, "error parsing selection_begin: "+err.Error(), http.StatusBadRequest)
			return
		}
		begin = int64(b)

		e, err := webutils.ParseByteOrRuneOffset(pyctx.ContextInputs.Buffer,
			params.Get("selection_end_bytes"),
			params.Get("selection_end_runes"))
		if err != nil {
			hoverFailReason.HitAndAdd("error parsing query param")
			http.Error(w, "error parsing selection_end: "+err.Error(), http.StatusBadRequest)
			return
		}
		end = int64(e)
	} else if params.Get("offset_encoding") != "" {
		// Use ParseOffsetToUTF8 if offset_encoding param is present
		c, err := webutils.ParseOffsetToUTF8(pyctx.ContextInputs.Buffer,
			params.Get("cursor_runes"),
			params.Get("offset_encoding"))
		if err != nil {
			hoverFailReason.HitAndAdd("error parsing query param")
			http.Error(w, "error parsing cursor: "+err.Error(), http.StatusBadRequest)
			return
		}
		begin = int64(c)
		end = begin
	} else {
		c, err := webutils.ParseByteOrRuneOffset(pyctx.ContextInputs.Buffer,
			params.Get("cursor_bytes"),
			params.Get("cursor_runes"))
		if err != nil {
			hoverFailReason.HitAndAdd("error parsing query param")
			http.Error(w, "error parsing cursor: "+err.Error(), http.StatusBadRequest)
			return
		}
		begin = int64(c)
		end = begin
	}

	if pyctx.LocalIndex != nil {
		hoverIndexAvailableRatio.Hit()
	} else {
		hoverIndexAvailableRatio.Miss()
	}

	resp := editorapi.HoverResponse{
		Language: lang.Python.Name(),
	}

	nodeType, sbs, err := resolveNode(ctx, pythonhelpers.DeepestContainingSelection(ctx, pyctx.AST, begin, end), resolveInputs{
		LocalIndex:  pyctx.LocalIndex,
		BufferIndex: pyctx.BufferIndex,
		Resolved:    pyctx.Resolved,
		Graph:       pyctx.Importer.Global,
	})
	if err != nil {
		var code int
		switch err.(type) {
		case unsupportedNodeError, nodeNotFoundError:
			code = http.StatusNotFound
		case resolutionError:
			if pyctx.LocalIndex == nil {
				code = http.StatusServiceUnavailable
			} else {
				code = http.StatusNotFound
			}
		default:
			panic("unhandled error type from resolveNode")
		}

		errS := err.Error()
		hoverFailReason.HitAndAdd(errS)
		http.Error(w, errS, code)
		return
	}

	resp.PartOfSyntax = nodeType

	for _, sb := range sbs {
		if resp.Report == nil {
			resp.Report = s.editor.renderSymbolReport(ctx, sb.chooseOne(ctx))
		}
		sym := s.editor.renderMemberSymbolExt(ctx, sb)
		resp.Symbol = append(resp.Symbol, sym)
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling json: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (s *DriverEndpoint) handleCallee(ctx kitectx.Context, w http.ResponseWriter, r *http.Request) {
	ctx.CheckAbort()

	calleeCounter.Add(1)
	defer calleeDuration.DeferRecord(time.Now())

	// Note: this function holds a read-only lock, so it must not modify the driver state
	s.d.lock.RLock()
	defer s.d.lock.RUnlock()

	params := r.URL.Query()

	pyctx := s.d.lastContext
	if pyctx == nil {
		http.Error(w, "driver.lastContext is nil", http.StatusServiceUnavailable)
		return
	}

	offset, err := webutils.ParseByteOrRuneOffset(pyctx.ContextInputs.Buffer,
		params.Get("offset_bytes"),
		params.Get("offset_runes"))
	if err != nil {
		http.Error(w, "error parsing offset: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, code, err := s.d.Callee(ctx, int64(offset))
	if err != nil || result.Failure != "" {
	}
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	buf, err := json.Marshal(result.Response)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling json: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func tokenBetween(t *pythonscanner.Word, begin, end token.Pos) bool {
	return begin <= t.Begin && t.End <= end
}

func tokenAt(t *pythonscanner.Word, begin, end token.Pos) bool {
	return t.Begin == begin && t.End == end
}
