package signatures

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	pythonsig "github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/signatures/internal/python"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

// Options contains options
type Options struct {
	Metric  *metrics.SignaturesMetric
	DevMode bool
}

type calleeRequest struct {
	Editor         string                     `json:"editor"`
	Filename       string                     `json:"filename"`
	Text           string                     `json:"text"`
	CursorRunes    int64                      `json:"cursor_runes"`
	OffsetEncoding stringindex.OffsetEncoding `json:"offset_encoding"`
}

// validate returns an error if the request is malformed
func (c calleeRequest) validate(m *Manager) error {
	_, err := webutils.OffsetToUTF8([]byte(c.Text), int(c.CursorRunes), c.OffsetEncoding)
	if err != nil {
		return err
	}

	if len(c.Text) > m.settings.GetMaxFileSizeBytes() {
		return errors.New("file too large")
	}

	return nil
}

// cursor returns the cursor offset in bytes, as inferred from CursorRunes.
// It is assumed to be called on valid requests only.
func (c calleeRequest) cursor() int64 {
	cursor, _ := webutils.OffsetToUTF8([]byte(c.Text), int(c.CursorRunes), c.OffsetEncoding)
	return int64(cursor)
}

// --

// Provider defines an interface used to retrieve callee from a driver
type Provider interface {
	Callee(ctx kitectx.Context, cursor int64) (python.CalleeResult, int, error)
}

// --

// Manager ...
type Manager struct {
	opts        Options
	provider    driver.Provider
	permissions component.PermissionsManager
	settings    component.SettingsManager
	cohort      component.FeatureEnabledWrapper
}

// NewManager creates a new Manager
func NewManager(provider driver.Provider, opts Options) *Manager {
	return &Manager{
		opts:     opts,
		provider: provider,
	}
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "signatures"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.permissions = opts.Permissions
	m.settings = opts.Settings
	m.cohort = opts.Cohort
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/editor/signatures", m.permissions.WrapAuthorizedFile(m.cohort.WrapFeatureEnabled(m.handleSignatures)))
}

// --

func (m *Manager) handleSignatures(w http.ResponseWriter, r *http.Request) {
	signaturesCount.Add(1)
	signaturesCountDist.Add(1)
	start := time.Now()
	defer signaturesDuration.DeferRecord(start)

	// default to UTF32 for backwards-compatibility
	req := calleeRequest{OffsetEncoding: stringindex.UTF32}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = req.validate(m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resp *editorapi.CalleeResponse
	err = kitectx.FromContext(r.Context(), func(ctx kitectx.Context) (err error) {
		resp, err = m.callee(ctx, req)
		return
	})
	if err != nil {
		aggregateHitRate.Miss()
		aggregateHitRateDist.Miss()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch {
	case resp == nil:
		http.Error(w, "resp was nil", http.StatusInternalServerError)
		return
	case resp.Callee == nil:
		http.Error(w, "resp.Callee was nil", http.StatusInternalServerError)
		return
	}

	manager, err := pythonsig.NewManager(
		resp.Callee,
		resp.Signatures,
		resp.FuncName,
		req.Filename,
		req.Text,
		req.cursor(),
		m.opts.DevMode,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	signatureResp := manager.Handle(req.Text, req.cursor())

	buf, err := json.Marshal(signatureResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(buf)

	signaturesReturnedCount.Add(1)
	signaturesReturnedCountDist.Add(1)
	aggregateHitRate.Hit()
	aggregateHitRateDist.Hit()
}

// Callee implements the signatures.CalleeProvider interface
func (m *Manager) callee(ctx kitectx.Context, r calleeRequest) (*editorapi.CalleeResponse, error) {
	ctx.CheckAbort()

	state := fmt.Sprintf("%x", md5.Sum([]byte(r.Text)))

	var f *driver.State
	var exists bool
	f, exists = m.provider.Driver(ctx, r.Filename, r.Editor, state)
	if !exists {
		if len(r.Text) > 0 {
			// Create file driver from content if it doesn't exist
			f = m.provider.DriverFromContent(ctx, r.Filename, r.Editor, r.Text, int(r.cursor()))
		} else {
			m.opts.Metric.SignatureRequested(false)
			return nil, fmt.Errorf("filename/editor/state not found")
		}
	}

	if provider, ok := f.FileDriver.(Provider); ok {
		result, _, err := provider.Callee(ctx, r.cursor())
		if err != nil {
			m.opts.Metric.SignatureRequested(false)
			return nil, err
		}

		m.opts.Metric.SignatureRequested(true)
		return result.Response, nil
	}

	m.opts.Metric.SignatureRequested(false)
	return nil, fmt.Errorf("unsupported driver")
}
