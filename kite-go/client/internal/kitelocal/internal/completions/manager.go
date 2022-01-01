package completions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"

	version "github.com/hashicorp/go-version"
	lru "github.com/hashicorp/golang-lru"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/rollbar"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/conversion/listener"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	complmetrics "github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
	lexicalapi "github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	pythonapi "github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/userids"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Options contains options
type Options struct {
	DevMode              bool
	Metrics              *complmetrics.MetricsByLang
	SmartSelectedMetrics *metrics.SmartSelectedMetrics
	PythonOptions        pythonapi.Options
	LexicalOptions       lexicalapi.Options
	ModelOptions         lexicalmodels.ModelOptions
}

// Manager ...
type Manager struct {
	cancel   func()
	opts     Options
	provider driver.Provider

	pythonapi  pythonapi.API
	lexicalapi lexicalapi.API

	settings    component.SettingsManager
	cohort      component.CohortManager
	permissions component.PermissionsManager
	userIDs     userids.IDs

	listener *listener.EventListener

	// A cache that indexes recently returned completions by snippet text.
	recentlyReturnedCompletions *lru.Cache

	// represents whether the last edit event was the typing of the first character of a token
	typedStartOfToken bool

	isUnitTestMode bool
}

// NewManager creates a new Manager
func NewManager(provider driver.Provider, opts Options) *Manager {
	croListener := listener.New()
	croListener.RegisterListener(opts.SmartSelectedMetrics)
	recentlyReturnedCompletions, _ := lru.New(100)
	return &Manager{
		opts:                        opts,
		provider:                    provider,
		listener:                    croListener,
		recentlyReturnedCompletions: recentlyReturnedCompletions,
	}
}

// Initialize is called by the kitelocal.Manager component
func (m *Manager) Initialize(opts component.InitializerOptions) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	m.initializeWithoutAPIs(opts)

	cohorts := m.cohort.Cohorts()
	for _, c := range cohorts.OnComplSelecters() {
		m.listener.RegisterListener(c)
	}
	m.pythonapi = pythonapi.New(ctx, m.opts.PythonOptions, cohorts)
	if m.opts.LexicalOptions.Models != nil {
		m.lexicalapi = lexicalapi.New(ctx, m.opts.LexicalOptions, cohorts)
	}
}

// Allows testing without resourcemanager, eg CRO related tests
func (m *Manager) initializeWithoutAPIs(opts component.InitializerOptions) {
	m.userIDs = opts.UserIDs
	m.settings = opts.Settings
	m.cohort = opts.Cohort
	m.permissions = opts.Permissions
	m.isUnitTestMode = opts.Platform.IsUnitTestMode
}

// Terminate is called by the kitelocal.Manager component
func (m *Manager) Terminate() {
	m.cancel()
}

// RegisterHandlers is called by the kitelocal.Manager component
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/editor/complete", m.permissions.WrapAuthorizedFile(m.cohort.WrapFeatureEnabled(m.handleComplete)))

	// This endpoint allows the editor plugins to report a selected completion
	mux.HandleFunc("/clientapi/metrics/completions/selected", m.handleCompletionSelected).Methods("POST")
}

// Reset clears completion engine drivers and models
func (m *Manager) Reset() {
	m.pythonapi.Reset()
	m.opts.PythonOptions.Models.Reset()
	if m.opts.LexicalOptions.Models != nil {
		m.lexicalapi.Reset()
		m.opts.LexicalOptions.Models.Reset()
	}
}

// ProcessedEvent is called by the kitelocal.Manager component
func (m *Manager) ProcessedEvent(e *event.Event, editorEvent *component.EditorEvent) {
	editor := data.Editor(editorEvent.Source)
	if m.supportsCompletionSelected(editor, editorEvent.EditorVersion, editorEvent.PluginVersion) {
		m.listener.RegisterReportingEditor(editor)
	}
	m.listener.OnEdit(e.GetText(), editor)
	if m.opts.LexicalOptions.Models != nil {
		m.lexicalapi.PushEditorEvent(editorEvent)
	}

	m.typedStartOfToken = isNewToken(e)

	l := lang.FromFilename(e.GetFilename())
	switch l {
	case lang.Python:
		m.handlePythonEvents(e)
	default:
		// Anything thats not python but supported via the permission manager
		// will be assumed to be lexical.
		m.handleLexicalEvents(e)
	}
}

// isNewToken returns true if an edit event is a single-character insertion that is the start of a new token
func isNewToken(e *event.Event) bool {
	newToken := false
	l := lang.FromFilename(e.GetFilename())
	diffs := e.GetDiffs()
	// If this is an ordinary typed character, there will be 1 diff
	if diffs != nil && len(diffs) == 1 {
		diff := diffs[0]
		// The diff must be an insertion with 1 character
		if *diff.Type == event.DiffType_INSERT && len(*diff.Text) == 1 {
			langLexer, err := lexicalv0.NewLexerForMetrics(l)
			if err != nil {
				log.Printf("could not lex edit event: %s\n", err.Error())
				return false
			}
			pretext := e.GetText()[0:*diff.Offset]
			text := e.GetText()[0 : *diff.Offset+1]

			pretoks, err := langLexer.Lex([]byte(pretext))
			if err != nil {
				log.Printf("could not lex edit event: %s\n", err.Error())
				return false
			}
			toks, err := langLexer.Lex([]byte(text))
			if err != nil {
				log.Printf("could not lex edit event: %s\n", err.Error())
				return false
			}

			// If there are more tokens after the insertion than before, then the insertion is the start of a new token
			if len(toks) > len(pretoks) {
				newToken = true
			}
		}
	}
	return newToken
}

// Returned is called when a completions request is successfully responded to.
func (m *Manager) Returned(resp data.APIResponse) {
	compls := resp.Completions
	for _, compl := range compls {
		m.recentlyReturnedCompletions.Add(compl.Completion.Snippet.Text, compl.RCompletion)
	}
}

// ToggleRemote swaps out the current models based on changes to the Kite Server setting.
func (m *Manager) ToggleRemote(host string) {
	opts := m.opts.ModelOptions
	if host != "" {
		opts = m.opts.ModelOptions.WithRemoteModels(host)
	} else {
		opts = m.opts.ModelOptions.ClearRemoteModels()
	}
	m.opts.PythonOptions.LexicalModels.UpdateRemote(opts)
	m.opts.LexicalOptions.Models.UpdateRemote(opts)
}

// --

type completionsFunc func(ctx kitectx.Context, req data.APIRequest, metricFn data.EngineMetricsCallback) data.APIResponse

func (m *Manager) handleComplete(w http.ResponseWriter, r *http.Request) {
	fn := m.permissions.Filename(r)
	l := lang.FromFilename(fn)

	metric := m.opts.Metrics.Get(l)
	start := time.Now()
	metric.Requested()

	var resp data.APIResponse
	defer func() {
		resBuf, _ := json.Marshal(resp)

		if resp.HTTPStatus == http.StatusMethodNotAllowed {
			// don't record returned or error if completions are disabled
		} else if resp.HTTPStatus < 200 || resp.HTTPStatus > 299 {
			metric.Errored()
		} else {
			m.Returned(resp)
			metric.Returned(resp, start)
			m.listener.OnReturned(resp)
		}
		if resp.HTTPStatus == 0 {
			resp.HTTPStatus = 500
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.HTTPStatus)
		w.Write(resBuf)
	}()

	complFunc := m.lexicalCompletions
	if l == lang.Python {
		complFunc = m.idccCompletions
	}

	m.processRequest(r, complFunc, metric, &resp)
}

func (m *Manager) processRequest(r *http.Request, complFunc completionsFunc, metric *completions.Metrics, resp *data.APIResponse) {
	var req data.APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.HTTPStatus = http.StatusBadRequest
		resp.Error = "could not decode request"
		return
	}

	if skip, _ := m.settings.GetBool(settings.CompletionsDisabledKey); skip {
		resp.HTTPStatus = http.StatusMethodNotAllowed
		resp.Error = "completions disabled"
		return
	}

	if len(req.Text()) > m.settings.GetMaxFileSizeBytes() {
		resp.HTTPStatus = http.StatusBadRequest
		resp.Error = "file too large"
		return
	}

	err := kitectx.FromContext(r.Context(), func(ctx kitectx.Context) error {
		*resp = complFunc(ctx, req, metric.GetEngineMetricsCallback(m.typedStartOfToken))
		return nil
	})
	if err != nil {
		resp.Error = err.Error()
		resp.HTTPStatus = http.StatusInternalServerError
		return
	}
}

func (m *Manager) handleCompletionSelected(w http.ResponseWriter, r *http.Request) {
	var cs complmetrics.CompletionSelectedEvent

	if err := json.NewDecoder(r.Body).Decode(&cs); err != nil {
		http.Error(w, errors.Errorf("error unmarshalling request: %v", err).Error(), http.StatusBadRequest)
		return
	}
	// Populate the event with a completion from the cache
	match, ok := m.recentlyReturnedCompletions.Get(cs.Completion.Snippet.Text)
	if !ok {
		http.Error(w, errors.Errorf("could not lookup completion").Error(), http.StatusBadRequest)
		return
	}
	compl, ok := match.(data.RCompletion)
	if !ok {
		rollbar.Error(fmt.Errorf("object retrieved from cache was not an RCompletion"))
		http.Error(w, errors.Errorf("error retrieving cached completion").Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("%s completion selection reported by editor", cs.Language)
	cs.Completion = compl
	if m.listener.EditorReports(cs.Editor) {
		m.listener.SendComplSelect(cs.Completion, cs.Editor)
	}
	langMetrics := m.opts.Metrics.Get(lang.FromName(cs.Language))
	if langMetrics == nil {
		http.Error(w, errors.Errorf("error finding metrics for language: %v", cs.Language).Error(), http.StatusBadRequest)
		return
	}
	langMetrics.CompletionSelected(cs)
}

// supportsCompletionSelected returns true if a plugin has implemented completion selected events
func (m *Manager) supportsCompletionSelected(editor data.Editor, editorVersion, pluginVersion string) bool {
	if plugVer, err := version.NewVersion(pluginVersion); err == nil {
		switch editor {
		case data.AtomEditor:
		case data.IntelliJEditor:
			// https://github.com/kiteco/intellij-plugin-private/commit/a74e30d8ecb96b3449e08e1f5b1b81ae75694cd1
			required, _ := version.NewConstraint(">= 1.7.7")
			return required.Check(plugVer)
		case data.JupyterEditor:
		case data.SpyderEditor:
		case data.SublimeEditor:
		case data.VSCodeEditor:
			// https://github.com/kiteco/vscode-plugin/commit/07194a483ac81791ced2a74b66e79c188c01210d
			required, _ := version.NewConstraint(">= 0.124.0")
			return required.Check(plugVer)
		case data.VimEditor:
		}
	}
	return false
}
