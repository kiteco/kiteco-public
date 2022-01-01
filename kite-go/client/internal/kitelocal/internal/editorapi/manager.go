package editorapi

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/localcode"
)

// --

const responsiveRemoteTimeout = time.Second * 2

// Manager ...
type Manager struct {
	provider    driver.Provider
	permissions component.PermissionsManager
	cohort      component.CohortManager
	docs        http.Handler
	network     component.NetworkManager

	services         *python.Services
	editorAPIHandler http.Handler
}

// NewManager creates a new Manager
func NewManager(provider driver.Provider, services *python.Services, local localcode.Context) *Manager {
	return &Manager{
		services:         services,
		provider:         provider,
		editorAPIHandler: editorapi.NewServer(python.NewEditorEndpoint(services, local)),
	}
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "editorapi"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.permissions = opts.Permissions
	m.docs = opts.DocsClient
	m.network = opts.Network
	m.cohort = opts.Cohort
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.PathPrefix("/api/python/curation").HandlerFunc(m.handleCuration)

	mux.PathPrefix("/api/editor/search").HandlerFunc(m.handleSearch)
	mux.PathPrefix("/api/editor/value/{id}").HandlerFunc(m.permissions.WrapAuthorizedFile(m.cohort.WrapFeatureEnabled(m.handleEditor)))
	mux.PathPrefix("/api/editor/symbol/{id}").HandlerFunc(m.permissions.WrapAuthorizedFile(m.cohort.WrapFeatureEnabled(m.handleEditor)))

	mux.PathPrefix("/api/buffer/{editor}/{filename}/{state}/{reqType}").HandlerFunc(m.cohort.WrapFeatureEnabled(m.permissions.WrapAuthorizedFile(m.handleBuffer)))
}
