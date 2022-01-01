package autocorrect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lang"
)

var (
	defaultAutocorrectTimeout = time.Second
	defaultHTTPTimeout        = 10 * time.Second
)

// Manager for autocorrect.
type Manager struct {
	proxy              component.AuthClient
	permissions        component.PermissionsManager
	maxFileSizeBytes   int
	enabled            bool
	autocorrectTimeout time.Duration
	defaultTimeout     time.Duration
}

// NewManager for autocorrect.
func NewManager(maxFileSizeBytes int, enabled bool) *Manager {
	return &Manager{
		maxFileSizeBytes:   maxFileSizeBytes,
		enabled:            enabled,
		autocorrectTimeout: defaultAutocorrectTimeout,
		defaultTimeout:     defaultHTTPTimeout,
	}
}

func newTestManager(maxFileSizeBytes int, enabled bool, defaultTimeout, autocorrectTimeout time.Duration) *Manager {
	m := NewManager(maxFileSizeBytes, enabled)
	m.defaultTimeout = defaultTimeout
	m.autocorrectTimeout = autocorrectTimeout
	return m
}

// Name implements component Core. It returns the name of the component
func (m *Manager) Name() string {
	return "autocorrect"
}

// Initialize implements component Initializer. It's called to setup the component
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.proxy = opts.AuthClient
	m.permissions = opts.Permissions
}

// RegisterHandlers implemented component Handlers. It's called to setup http routes.
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/editor/autocorrect/metrics", m.HandleMetrics).Methods("POST")
	mux.HandleFunc("/clientapi/editor/autocorrect/feedback", m.HandleFeedback).Methods("POST")
	mux.HandleFunc("/clientapi/editor/autocorrect/validation/on-save", m.HandleOnSave).Methods("POST")
	mux.HandleFunc("/clientapi/editor/autocorrect/validation/on-run", m.HandleOnRunGet).Methods("GET")
	mux.HandleFunc("/clientapi/editor/autocorrect/validation/on-run", m.HandleOnRun).Methods("POST")
	mux.HandleFunc("/clientapi/editor/autocorrect", m.HandleAutocorrect).Methods("POST")
}

// HandleOnSave requests.
func (m *Manager) HandleOnSave(w http.ResponseWriter, r *http.Request) {
	m.handleOnX(w, r, "/api/editor/autocorrect/validation/on-save")
}

// HandleOnRunGet requests
func (m *Manager) HandleOnRunGet(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// HandleOnRun requests.
func (m *Manager) HandleOnRun(w http.ResponseWriter, r *http.Request) {
	m.handleOnX(w, r, "/api/editor/autocorrect/validation/on-run")
}

func (m *Manager) handleOnX(w http.ResponseWriter, r *http.Request, endpoint string) {
	// do whitelist and language checks manually because the other flows are kind of messy
	var req struct {
		Filename string `json:"filename"`
		Buffer   string `json:"buffer"`
	}

	var buf bytes.Buffer
	body := io.TeeReader(r.Body, &buf)

	if err := json.NewDecoder(body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error unmarshalling json: %v", err), http.StatusBadRequest)
		return
	}

	code, err := m.validate(req.Filename, req.Buffer)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), m.defaultTimeout)
	defer cancel()

	m.postAndRespond(ctx, endpoint, "application/json", &buf, w)
}

// HandleAutocorrect requests
func (m *Manager) HandleAutocorrect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metadata struct {
			Source string `json:"source"`
		} `json:"metadata"`
		Filename string `json:"filename"`
		Buffer   string `json:"buffer"`
	}

	var buf bytes.Buffer
	reqBody := io.TeeReader(r.Body, &buf)

	if err := json.NewDecoder(reqBody).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error unmarshalling json: %v", err), http.StatusBadRequest)
		responseCodes.HitAndAdd(fmt.Sprintf("%d", http.StatusBadRequest))
		return
	}

	switch {
	case m.enabled:
	case req.Metadata.Source == "atom":
	case req.Metadata.Source == "vscode":
	default:
		http.Error(w, "not implemented", http.StatusNotImplemented)
		responseCodes.HitAndAdd(fmt.Sprintf("%d", http.StatusNotImplemented))
		return
	}

	code, err := m.validate(req.Filename, req.Buffer)
	if err != nil {
		responseCodes.HitAndAdd(fmt.Sprintf("%d", code))
		http.Error(w, err.Error(), code)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), m.autocorrectTimeout)
	defer cancel()

	code = m.postAndRespond(ctx, "/api/editor/autocorrect", "application/json", &buf, w)

	responseCodes.HitAndAdd(fmt.Sprintf("%d", code))
}

// HandleMetrics requests.
func (m *Manager) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), m.defaultTimeout)
	defer cancel()

	m.postAndRespond(ctx, "/api/editor/autocorrect/metrics", "application/json", r.Body, w)
}

// HandleFeedback requests.
func (m *Manager) HandleFeedback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), m.defaultTimeout)
	defer cancel()

	m.postAndRespond(ctx, "/api/editor/autocorrect/feedback", "application/json", r.Body, w)
}

func (m *Manager) validate(filename, buffer string) (int, error) {
	// check file size
	if len(buffer) > m.maxFileSizeBytes {
		//drop event
		return http.StatusRequestEntityTooLarge, fmt.Errorf("file too large %d > %d", len(buffer), m.maxFileSizeBytes)
	}

	// check language
	ok, err := m.permissions.IsSupportedLangExtension(filename, map[lang.Language]struct{}{lang.Python: {}})
	if !ok || err != nil {
		return http.StatusForbidden, fmt.Errorf("non python file")
	}

	return http.StatusOK, nil
}

func (m *Manager) postAndRespond(ctx context.Context, url, contentType string, body io.Reader, w http.ResponseWriter) int {
	resp, err := m.proxy.Post(ctx, url, contentType, body)

	var isTimeout bool
	if err, ok := err.(net.Error); ok {
		isTimeout = err.Timeout()
	}

	switch {
	case err == nil:
		defer resp.Body.Close()
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return resp.StatusCode
	case isTimeout || err == context.DeadlineExceeded:
		http.Error(w, "timeout", http.StatusRequestTimeout)
		return http.StatusRequestTimeout
	default:
		err = fmt.Errorf("error posting to %s: %v", url, err)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return http.StatusInternalServerError
	}
}
