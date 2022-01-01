package navigation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/internal/ws"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/navigation/codebase"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	projectRootKey = iota
	initRelatedRequestKey
)

const (
	defaultMaxFileRecs      = -1
	defaultMaxFileKeywords  = -1
	defaultMaxBlockRecs     = 5
	defaultMaxBlockKeywords = 10
	defaultUnloadInterval   = time.Hour
)

// Manager ...
type Manager struct {
	ws       *ws.Manager
	settings component.SettingsManager
	cohort   component.FeatureEnabledWrapper
	store    recommendStore
	bufcache bufferCache

	m              sync.Mutex
	activePath     string
	navigator      codebase.Navigator
	unloadInterval time.Duration
}

// NewManager ...
func NewManager(storagePath string) (*Manager, error) {
	opts := codebase.Options{
		ComputedCommitsLimit: git.DefaultComputedCommitsLimit,
		GitStorageOpts: git.StorageOptions{
			UseDisk: true,
			Path:    storagePath,
		},
	}
	return newManager(opts)
}

func newManager(opts codebase.Options) (*Manager, error) {
	navigator, err := codebase.NewNavigator(opts)
	if err != nil {
		return nil, err
	}
	return &Manager{
		navigator:      navigator,
		ws:             ws.NewManager(),
		unloadInterval: defaultUnloadInterval,
	}, nil
}

// Initialize is called by the kitelocal.Manager component
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.settings = opts.Settings
	m.cohort = opts.Cohort

	// Disable unloading
	if m.unloadInterval <= 0 {
		return
	}

	kitectx.Go(func() error {
		// Checking if we want to unload in a reasonable frequency, similar to lexicalcomplete/api/api.go
		ticker := time.NewTicker(m.unloadInterval / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if m.navigator.WasTerminated() {
					return nil
				}
				m.navigator.MaybeUnload(m.unloadInterval)
			}
		}
	})
}

// Name implements component Core. It returns the name of the component
func (m *Manager) Name() string {
	return "navigation"
}

// Terminate ...
func (m *Manager) Terminate() {
	m.navigator.Terminate()
	m.ws.CloseConnections()
}

// ProcessedEvent loads projects when they are edited.
func (m *Manager) ProcessedEvent(event *event.Event, editorEvent *component.EditorEvent) {
	if editorEvent == nil {
		return
	}

	m.m.Lock()
	defer m.m.Unlock()

	// Always update the buffer cache to the most recent event data
	defer m.bufcache.update(editorEvent.Filename, editorEvent.Text)

	if editorEvent.Filename == m.activePath {
		// If MaybeLoad is called repeatedly with the same path,
		// it recomputes the associated project root of a path each time.
		// If the associated project root changes, it will load the new project.
		// But the project associated with a path rarely changes,
		// so we only recompute when the path is not equal to the active path.
		return
	}
	m.activePath = editorEvent.Filename

	kitectx.Go(func() error {
		m.navigator.MaybeLoad(editorEvent.Filename, int64(m.settings.GetMaxFileSizeBytes()), m.getMaxFiles())
		return nil
	})
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/codenav/editor/related", m.cohort.WrapFeatureEnabled(m.handleEditorRelated)).Methods("POST")
	mux.HandleFunc("/codenav/decoration/line", m.cohort.WrapFeatureEnabled(m.handleDecorationLine)).Methods("POST")
	mux.HandleFunc("/codenav/related", m.cohort.WrapFeatureEnabled(m.handleFetchRelated)).Methods("POST")
	mux.Handle("/codenav/subscribe", websocket.Handler(m.ws.HandleEventsWS))
}

type editorRelatedRequest struct {
	Location   recommend.Location `json:"location"`
	Editor     data.Editor        `json:"editor"`
	EditorPath string             `json:"editor_install_path"`
}

// codenavPushMsg echos the initial request with some rendering
type codenavPushMsg struct {
	Editor     data.Editor        `json:"editor"`
	EditorPath string             `json:"editor_install_path"`
	Filename   string             `json:"filename"`
	Location   recommend.Location `json:"location"`
	ProjTag    string             `json:"project_tag"`
	RelPath    string             `json:"relative_path"`
}

type fetchRelatedRequest struct {
	Location    recommend.Location `json:"location"`
	NFiles      int                `json:"num_files"`
	NBlockRecs  int                `json:"num_blocks,omitempty"`
	NBlockWords int                `json:"num_keywords,omitempty"`
	Offset      int                `json:"offset,omitempty"`

	// for telemetry
	Editor data.Editor `json:"editor,omitempty"`
}

type fetchRelatedResponse struct {
	Filename     string        `json:"filename"`
	RelPath      string        `json:"relative_path"`
	ProjRoot     string        `json:"project_root"`
	RelatedFiles []fileWithRel `json:"related_files"`
	RemoteRepo   string        `json:"remote_repo"`
}

type fileWithRel struct {
	File     recommend.File `json:"file"`
	Filename string         `json:"filename"`
	RelPath  string         `json:"relative_path"`
}

func (m *Manager) handleFetchRelated(w http.ResponseWriter, r *http.Request) {
	var req fetchRelatedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Could not decode request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := req.Location.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	proot, err := m.validate(req.Location.CurrentPath)
	switch err {
	case nil:
		// continue
	case codebase.ErrProjectNotLoaded, codebase.ErrProjectStillIndexing:
		if req.Offset == 0 {
			// For initial requests where the project hasn't loaded
			// or is still indexing block until loaded or errored
			m.navigator.MaybeLoad(req.Location.CurrentPath, int64(m.settings.GetMaxFileSizeBytes()), m.getMaxFiles())
			newproot, er := m.validate(req.Location.CurrentPath)
			if er != nil {
				http.Error(w, er.Error(), http.StatusMethodNotAllowed)
				return
			}
			// On successful load, continue and return results
			proot = newproot
		} else {
			// Model was unloaded but subsequent results were requested
			// Load the project in the background and notify client of the error
			kitectx.Go(func() error {
				m.navigator.MaybeLoad(req.Location.CurrentPath, int64(m.settings.GetMaxFileSizeBytes()), m.getMaxFiles())
				return nil
			})
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}
	default:
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}
	resp, err := m.loadRecs(&req, proot)
	if err != nil {
		http.Error(w, "Failed to load recommendations: "+err.Error(), http.StatusInternalServerError)
		return
	}
	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(b)

	if req.Offset == 0 {
		// Initial results have been sent, and assumed displayed by the Copilot
		clienttelemetry.EventWithKiteTelemetry("code_finder_results_displayed", map[string]interface{}{
			"editor": req.Editor,
		})
	}
}

func (m *Manager) loadRecs(req *fetchRelatedRequest, projroot string) (*fetchRelatedResponse, error) {
	if err := m.store.load(m.getFileIterator, req, projroot); err != nil {
		// Fallthrough. The store will return any requested in the range, which may be empty.
		if err != codebase.ErrEmptyIterator {
			return nil, err
		}
	}
	relfn, err := filepath.Rel(projroot, req.Location.CurrentPath)
	if err != nil {
		return nil, err
	}
	relPath, filename := filepath.Split(relfn)

	return &fetchRelatedResponse{
		ProjRoot: projroot,
		// RemoteRepo: TODO
		RelatedFiles: m.store.viewOver(req.Offset, req.NFiles),
		Filename:     filename,
		RelPath:      relPath,
	}, nil
}

var msgSomethingWentWrong = "Oops! Something went wrong with Code Finder. Please try again later."
var msgUnavailable = "Code Finder is not available. Please try again later."
var msgStillIndexing = "Kite is not done indexing your project yet. Please wait for the status icon to switch to ready before using Code Finder."

func (m *Manager) handleEditorRelated(w http.ResponseWriter, r *http.Request) {
	msg, status := m.triggerEditorRelated(r)
	if status >= 400 && msg == "" {
		msg = msgSomethingWentWrong
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func (m *Manager) triggerEditorRelated(r *http.Request) (string, int) {
	var req editorRelatedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("error decoding /codenav/editor/related request.", err)
		return msgSomethingWentWrong, http.StatusBadRequest
	}

	if err := req.Location.Validate(); err != nil {
		log.Println("error validating /codenav/editor/related request.", err)
		return msgSomethingWentWrong, http.StatusBadRequest
	}

	proot, err := m.validate(req.Location.CurrentPath)
	errorShown := func(err error) {
		clienttelemetry.EventWithKiteTelemetry("code_finder_query_error_shown", map[string]interface{}{
			"editor": req.Editor,
			"error":  codebase.ErrorString(err),
		})
	}
	switch err {
	case nil:
		// continue
	case codebase.ErrProjectNotLoaded:
		// If the user sends a request before an editor event,
		// load the project and return that the project is indexing.
		kitectx.Go(func() error {
			m.navigator.MaybeLoad(req.Location.CurrentPath, int64(m.settings.GetMaxFileSizeBytes()), m.getMaxFiles())
			return nil
		})
		return msgStillIndexing, http.StatusServiceUnavailable
	case codebase.ErrProjectStillIndexing:
		errorShown(err)
		return msgStillIndexing, http.StatusServiceUnavailable
	case codebase.ErrPathHasUnsupportedExtension:
		ext := filepath.Ext(req.Location.CurrentPath)
		errorShown(err)
		return fmt.Sprintf("Code Finder does not yet support the %s file extension.", ext), http.StatusMethodNotAllowed
	case codebase.ErrPathInFilteredDirectory:
		errorShown(err)
		fname := filepath.Base(req.Location.CurrentPath)
		return fmt.Sprintf("Code Finder cannot operate on %s, as it is in a private directory ignored by Kite.", fname), http.StatusMethodNotAllowed
	case codebase.ErrPathNotInSupportedProject:
		fname := filepath.Base(req.Location.CurrentPath)
		errorShown(err)
		return fmt.Sprintf("The file %s is not in any Git project. Code Finder only works within Git projects.", fname), http.StatusMethodNotAllowed
	default:
		return msgSomethingWentWrong, http.StatusInternalServerError
	}

	projTag := filepath.Base(proot)
	rel, err := filepath.Rel(proot, req.Location.CurrentPath)
	if err != nil {
		log.Println("error computing relative path in /codenav/editor/related.", err)
		return msgSomethingWentWrong, http.StatusInternalServerError
	}
	relPath, filename := filepath.Split(rel)

	timeout := 10 * time.Second
	if err = sidebar.Start(); err == nil {
		if len(m.ws.ActiveConnections()) == 0 {
			select {
			case <-m.ws.ConnectionAdded():
				// block until Copilot establishes WebSocket connection
			case <-time.After(timeout):
				log.Println("Sidebar WebSocket initialization took longer than", timeout)
				return msgSomethingWentWrong, http.StatusInternalServerError
			}
		}
	}

	m.ws.BroadcastJSON(&codenavPushMsg{
		Location:   req.Location,
		Editor:     req.Editor,
		EditorPath: req.EditorPath,
		Filename:   filename,
		ProjTag:    projTag,
		RelPath:    relPath,
	})

	// Editor request was successfully initiated and processed
	reqType := "file"
	if req.Location.CurrentLine > 0 {
		reqType = "line"
	}
	clienttelemetry.EventWithKiteTelemetry("code_finder_query_sent", map[string]interface{}{
		"editor": req.Editor,
		"type":   reqType,
	})

	return "", http.StatusOK
}

func (m *Manager) handleDecorationLine(w http.ResponseWriter, r *http.Request) {
	type response struct {
		InlineMessage string `json:"inline_message"`
		HoverMessage  string `json:"hover_message"`
		ProjectReady  *bool  `json:"project_ready,omitempty"`
	}

	var loc recommend.Location
	if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
		http.Error(w, "Could not decode request", http.StatusBadRequest)
		return
	}
	if err := loc.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	root, vErr := m.validate(loc.CurrentPath)
	var resp response
	switch vErr {
	case nil:
		resp = response{
			InlineMessage: fmt.Sprintf("Find related code in %s", filepath.Base(root)),
			HoverMessage:  fmt.Sprintf("Search for code in %s which may be related to this line", filepath.Base(root)),
			ProjectReady:  proto.Bool(true),
		}
	case codebase.ErrPathNotInSupportedProject:
		// resp.ProjectReady == nil
	case codebase.ErrPathHasUnsupportedExtension:
		// resp.ProjectReady == nil
	default:
		resp = response{
			ProjectReady: proto.Bool(false),
		}
	}
	b, mErr := json.Marshal(resp)
	if mErr != nil {
		http.Error(w, mErr.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

// Validate whether the filepath can be used for codenav
// It's exported for the statusbar to check the project status
func (m *Manager) Validate(path string) error {
	_, err := m.validate(path)
	return err
}

func (m *Manager) validate(path string) (root string, err error) {
	status, root, err := m.navigator.ProjectInfo(path)
	if err != nil {
		return root, err
	}
	mapStatusToErr := func(status codebase.ProjectStatus) error {
		switch status {
		case codebase.Inactive:
			return codebase.ErrProjectNotLoaded
		case codebase.InProgress:
			return codebase.ErrProjectStillIndexing
		case codebase.Failed, codebase.IgnorerFailed:
			return codebase.ErrProjectBuildFailed
		}
		return nil
	}
	return root, mapStatusToErr(status)
}

func (m *Manager) getMaxFiles() int {
	maxFiles, err := m.settings.GetInt(settings.PredictiveNavMaxFilesKey)
	if err != nil {
		return 1e5
	}
	return maxFiles
}

func (m *Manager) toRecommendRequest(req *fetchRelatedRequest) recommend.Request {
	return recommend.Request{
		Location:         req.Location,
		MaxFileRecs:      defaultMaxFileRecs,
		MaxBlockRecs:     chooseDefaultIfZero(req.NBlockRecs, defaultMaxBlockRecs),
		MaxFileKeywords:  defaultMaxFileKeywords,
		MaxBlockKeywords: chooseDefaultIfZero(req.NBlockWords, defaultMaxBlockKeywords),
		BufferContents:   m.bufcache.bytes(req.Location.CurrentPath),
	}
}

func (m *Manager) getFileIterator(req *fetchRelatedRequest) (codebase.FileIterator, error) {
	iter, err := m.navigator.Navigate(m.toRecommendRequest(req))
	if err == codebase.ErrShouldLoad {
		kitectx.Go(func() error {
			m.navigator.MaybeLoad(req.Location.CurrentPath, int64(m.settings.GetMaxFileSizeBytes()), m.getMaxFiles())
			return nil
		})
	}
	return iter, err
}

type recommendStore struct {
	m    sync.Mutex
	loc  recommend.Location
	iter codebase.FileIterator
	recs []fileWithRel
}

func (s *recommendStore) load(getFileIterator func(req *fetchRelatedRequest) (codebase.FileIterator, error), req *fetchRelatedRequest, proot string) error {
	s.m.Lock()
	defer s.m.Unlock()

	// An offset of 0 indicates a new request, even if it's the same file location
	if s.iter.Next == nil || req.Location != s.loc || req.Offset == 0 {
		iter, err := getFileIterator(req)
		if err != nil {
			return err
		}
		s.iter = iter
		s.loc = req.Location
		s.recs = []fileWithRel{}
	}

	// Load any missing recommendations into the store
	deficit := (req.Offset + req.NFiles) - len(s.recs)
	if deficit > 0 {
		newRecs, err := s.iter.Next(deficit)
		if err != nil {
			return err
		}
		newRecsWithRel, err := mapWithRelative(newRecs, proot)
		if err != nil {
			return err
		}
		for _, rec := range newRecsWithRel {
			s.recs = append(s.recs, rec)
		}
	}
	return nil
}

func (s *recommendStore) viewOver(start, n int) []fileWithRel {
	s.m.Lock()
	defer s.m.Unlock()

	return s.recs[min(start, len(s.recs)):min(start+n, len(s.recs))]
}

func mapWithRelative(recs []recommend.File, proot string) ([]fileWithRel, error) {
	wr := make([]fileWithRel, len(recs))
	for i, f := range recs {
		relfn, err := filepath.Rel(proot, f.Path)
		if err != nil {
			return nil, err
		}
		relPath, filename := filepath.Split(relfn)
		wr[i] = fileWithRel{
			RelPath:  relPath,
			Filename: filename,
			File:     f,
		}
	}
	return wr, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func chooseDefaultIfZero(val, def int) int {
	if val == 0 {
		return def
	}
	return val
}

func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
