package status

import (
	"net/http"
	"strconv"
)

// RecordStatusCode wraps an HTTP handler and records the status code
func RecordStatusCode(wrapped http.HandlerFunc, code *Breakdown) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ww := newStatusCodeResponseWriter(w, code)
		wrapped.ServeHTTP(ww, r)
	}
}

type statusCodeResponseWriter struct {
	http.ResponseWriter
	code   *Breakdown
	header bool
}

func newStatusCodeResponseWriter(w http.ResponseWriter, code *Breakdown) *statusCodeResponseWriter {
	return &statusCodeResponseWriter{
		ResponseWriter: w,
		code:           code,
	}
}

// Header implements the http.ResponseWriter interface
func (w *statusCodeResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Write implements the http.ResponseWriter interface
func (w *statusCodeResponseWriter) Write(body []byte) (int, error) {
	if !w.header {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// WriteHeader implements the http.ResponseWriter interface
func (w *statusCodeResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	if w.header {
		return
	}
	w.code.HitAndAdd(strconv.Itoa(code))
	w.header = true
}
