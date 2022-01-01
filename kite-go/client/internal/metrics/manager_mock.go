package metrics

import (
	"sync"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	"github.com/kiteco/kiteco/kite-golib/mixpanel"
)

// NewMockManager returns a mocked metrics manager implementation
func NewMockManager() component.MetricsManager {
	return &mockManager{
		mu:              sync.Mutex{},
		completionsUsed: completions.NewMetrics(),
		mixpanel:        &mixpanel.Metrics{},
		traits:          make(map[string]interface{}),
	}
}

type mockManager struct {
	mu              sync.Mutex
	menubarVisible  bool
	region          string
	completionsUsed *completions.MetricsByLang
	mixpanel        *mixpanel.Metrics
	traits          map[string]interface{}
}

func (m *mockManager) Name() string {
	return "metrics-mock"
}

func (m *mockManager) SetMenubarVisible(v bool) {
	m.menubarVisible = v
}

func (m *mockManager) IsMenubarVisible() bool {
	return m.menubarVisible
}

func (m *mockManager) GetRegion() string {
	return m.region
}

func (m *mockManager) SetRegion(region string) {
	m.region = region
}

func (m *mockManager) Completions() *completions.MetricsByLang {
	return m.completionsUsed
}

func (m *mockManager) SendPartialStatusMetrics() {
	// no-op
}

func (m *mockManager) Identify() {
	// no-op
}

func (m *mockManager) GitFound() bool {
	return false
}

func (m *mockManager) UpdateUser(traits map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, v := range traits {
		m.traits[k] = v
	}
}

// Traits returns the traits which were added to this user
func (m *mockManager) Traits() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.traits
}
