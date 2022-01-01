package component

import (
	"context"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/response"
)

// compile-time check that we implement the intended components
var (
	_ = UserAuth((*CountingComponent)(nil))
)

// NewCountingComponent returns a new counting component
func NewCountingComponent(name string) *CountingComponent {
	return &CountingComponent{
		name:   name,
		counts: make(map[string]int64),
	}
}

// CountingComponent is supposed to be used in unit tests.
// It implements every component interface and counts its method calls
type CountingComponent struct {
	name   string
	mu     sync.Mutex
	counts map[string]int64
}

// Reset resets all recorded counts to zero
func (f *CountingComponent) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counts = make(map[string]int64)
}

// Name implements interface Core
func (f *CountingComponent) Name() string {
	return f.name
}

// GoTick implements interace Ticker
func (f *CountingComponent) GoTick(ctx context.Context) {
	f.incCount("tick")
}

// GetTickCount returns the ticks
func (f *CountingComponent) GetTickCount() int64 {
	return f.getCount("tick")
}

// Initialize implements interface Initializer
func (f *CountingComponent) Initialize(opts InitializerOptions) {
	f.incCount("init")
}

// GetInitCount returns how many times Initialize() was called
func (f *CountingComponent) GetInitCount() int64 {
	return f.getCount("init")
}

// RegisterHandlers implements a component
func (f *CountingComponent) RegisterHandlers(mux *mux.Router) {
	f.incCount("handlers")
}

// GetRegisterHandlersCount returns how many times RegisterHandlers() was called
func (f *CountingComponent) GetRegisterHandlersCount() int64 {
	return f.getCount("handlers")
}

// EventResponse implements a component
func (f *CountingComponent) EventResponse(root *response.Root) {
	f.incCount("eventResponse")
}

// GetEventResponseCount returns how many times EventResponse() was called
func (f *CountingComponent) GetEventResponseCount() int64 {
	return f.getCount("eventResponse")
}

// SettingUpdated implements a component
func (f *CountingComponent) SettingUpdated(string, string) {
	f.incCount("settingsUpdated")
}

// GetSettingsUpdatedCount returns how many times SettingsUpdated() was called
func (f *CountingComponent) GetSettingsUpdatedCount() int64 {
	return f.getCount("settingsUpdated")
}

// SettingDeleted implements a component
func (f *CountingComponent) SettingDeleted(string) {
	f.incCount("settingsDeleted")
}

// GetSettingsDeletedCount returns how many times GetSettingsDeleted() was called
func (f *CountingComponent) GetSettingsDeletedCount() int64 {
	return f.getCount("settingsDeleted")
}

// PluginEvent implements a component
func (f *CountingComponent) PluginEvent(*EditorEvent) {
	f.incCount("pluginEvent")
}

// GetPluginEventCount returns how many times PluginEvent() was called
func (f *CountingComponent) GetPluginEventCount() int64 {
	return f.getCount("pluginEvent")
}

// ProcessedEvent implements a component
func (f *CountingComponent) ProcessedEvent(*event.Event, *EditorEvent) {
	f.incCount("processedEvent")
}

// GetProcessedEventsCount returns how many times ProcessedEvents() was called
func (f *CountingComponent) GetProcessedEventsCount() int64 {
	return f.getCount("processedEvent")
}

// LoggedIn implements a component
func (f *CountingComponent) LoggedIn() {
	f.incCount("loggedIn")
}

// GetLoggedInCount returns how many times LoggedIn() was called
func (f *CountingComponent) GetLoggedInCount() int64 {
	return f.getCount("loggedIn")
}

// LoggedOut implements a component
func (f *CountingComponent) LoggedOut() {
	f.incCount("loggedOut")
}

// GetLoggedOutCount returns how many times LoggedOut() was called
func (f *CountingComponent) GetLoggedOutCount() int64 {
	return f.getCount("loggedOut")
}

// Terminate implements a component
func (f *CountingComponent) Terminate() {
	f.incCount("terminate")
}

// GetTerminateCount returns how many times Terminate() was called
func (f *CountingComponent) GetTerminateCount() int64 {
	return f.getCount("terminate")
}

func (f *CountingComponent) incCount(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counts[key]++
}

func (f *CountingComponent) getCount(key string) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.counts[key]
}
