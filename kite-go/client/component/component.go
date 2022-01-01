package component

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// Manager manages a list of components, components which implement Handlers are added to "handlers"
type Manager struct {
	components   sync.Map
	unitTestMode bool
}

// NewManager returns a new manager with an empty list of components
func NewManager() *Manager {
	return &Manager{}
}

// NewTestManager returns a manager suitable for test cases
func NewTestManager() *Manager {
	return &Manager{
		unitTestMode: true,
	}
}

// Add adds a component to the list of managed components.
// If the name was registered before an error is returned
// If the name is empty an error is returned
func (m *Manager) Add(component Core) error {
	name := component.Name()
	if name == "" {
		return fmt.Errorf("component must have Name() return non-empty value")
	}

	if _, ok := m.components.Load(name); ok {
		return fmt.Errorf("component with Name() %s already added", name)
	}

	m.components.LoadOrStore(name, component)
	return nil
}

// Components returns all registered components
func (m *Manager) Components() []Core {
	var result []Core
	m.components.Range(func(key, value interface{}) bool {
		if c, ok := value.(Core); ok {
			result = append(result, c)
		}
		return true
	})

	return result
}

// RegisterHandlers attaches all routes which are provided by components to the router.
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	m.components.Range(func(name, component interface{}) bool {
		if handler, ok := component.(Handlers); ok {
			func() {
				defer m.panicRecovery("RegisterHandlers", component.(Core))
				handler.RegisterHandlers(mux)
			}()
		}
		return true
	})
}

// --

// Initialize delegates the call to all components which implement the interface Initializer
func (m *Manager) Initialize(opts InitializerOptions) {
	m.components.Range(func(name, component interface{}) bool {
		if init, ok := component.(Initializer); ok {
			func() {
				defer m.panicRecovery("Initialize", component.(Core))
				init.Initialize(opts)
			}()
		}
		return true
	})
}

// Terminate delegates the call to all components which implement the interface Terminater
func (m *Manager) Terminate() {
	m.components.Range(func(name, component interface{}) bool {
		if term, ok := component.(Terminater); ok {
			func() {
				defer m.panicRecovery("Terminate", component.(Core))
				term.Terminate()
			}()
		}
		return true
	})
}

// NetworkOnline delegates the call to all components which implement the interface NetworkEventer
func (m *Manager) NetworkOnline() {
	m.components.Range(func(name, component interface{}) bool {
		if eventer, ok := component.(NetworkEventer); ok {
			func() {
				defer m.panicRecovery("NetworkOnline", component.(Core))
				eventer.NetworkOnline()
			}()
		}
		return true
	})
}

// NetworkOffline delegates the call to all components which implement the interface NetworkEventer
func (m *Manager) NetworkOffline() {
	m.components.Range(func(name, component interface{}) bool {
		if eventer, ok := component.(NetworkEventer); ok {
			func() {
				defer m.panicRecovery("NetworkOffline", component.(Core))
				eventer.NetworkOffline()
			}()
		}
		return true
	})
}

// KitedInitialized delegates the call to all components which implement the interface KitedEventer
func (m *Manager) KitedInitialized() {
	m.components.Range(func(name, component interface{}) bool {
		if eventer, ok := component.(KitedEventer); ok {
			func() {
				defer m.panicRecovery("KitedInitialized", component.(Core))
				eventer.KitedInitialized()
			}()
		}
		return true
	})
}

// KitedUninitialized delegates the call to all components which implement the interface KitedEventer
func (m *Manager) KitedUninitialized() {
	m.components.Range(func(name, component interface{}) bool {
		if eventer, ok := component.(KitedEventer); ok {
			func() {
				defer m.panicRecovery("KitedUninitialized", component.(Core))
				eventer.KitedUninitialized()
			}()
		}
		return true
	})
}

// LoggedIn delegates the call to all components which implement the interface UserAuth
func (m *Manager) LoggedIn() {
	m.components.Range(func(name, component interface{}) bool {
		if auth, ok := component.(UserAuth); ok {
			func() {
				defer m.panicRecovery("LoggedIn", component.(Core))
				auth.LoggedIn()
			}()
		}
		return true
	})
}

// LoggedOut delegates the call to all components which implement the interface UserAuth
func (m *Manager) LoggedOut() {
	m.components.Range(func(name, component interface{}) bool {
		if auth, ok := component.(UserAuth); ok {
			func() {
				defer m.panicRecovery("LoggedOut", component.(Core))
				auth.LoggedOut()
			}()
		}
		return true
	})
}

// PluginEvent delegates the call to all components which implement the interface PluginEventer
func (m *Manager) PluginEvent(evt *EditorEvent) {
	m.components.Range(func(name, component interface{}) bool {
		if eventer, ok := component.(PluginEventer); ok {
			func() {
				defer m.panicRecovery("PluginEvent", component.(Core))
				eventer.PluginEvent(evt)
			}()
		}
		return true
	})
}

// ProcessedEvent delegates the call to all components which implement the interface ProcessedEventer
func (m *Manager) ProcessedEvent(evt *event.Event, editorEvt *EditorEvent) {
	m.components.Range(func(name, component interface{}) bool {
		if eventer, ok := component.(ProcessedEventer); ok {
			func() {
				defer m.panicRecovery("ProcessedEvent", component.(Core))
				eventer.ProcessedEvent(evt, editorEvt)
			}()
		}
		return true
	})
}

// EventResponse delegates the call to all components which implement the interface EventResponser
func (m *Manager) EventResponse(resp *response.Root) {
	m.components.Range(func(name, component interface{}) bool {
		if responser, ok := component.(EventResponser); ok {
			func() {
				defer m.panicRecovery("EventResponse", component.(Core))
				responser.EventResponse(resp)
			}()
		}
		return true
	})
}

// Updated delegates the call to all components which implement the interface Settings
func (m *Manager) Updated(key, value string) {
	m.components.Range(func(name, component interface{}) bool {
		if s, ok := component.(Settings); ok {
			func() {
				defer m.panicRecovery("SettingUpdated", component.(Core))
				s.SettingUpdated(key, value)
			}()
		}
		return true
	})
}

// Deleted delegates the call to all components which implement the interface Settings
func (m *Manager) Deleted(key string) {
	m.components.Range(func(name, component interface{}) bool {
		if s, ok := component.(Settings); ok {
			func() {
				defer m.panicRecovery("SettingDeleted", component.(Core))
				s.SettingDeleted(key)
			}()
		}
		return true
	})
}

// GoTick delegates the call to all components which implement the interface Ticker
func (m *Manager) GoTick(ctx context.Context) {
	m.components.Range(func(name, component interface{}) bool {
		if s, ok := component.(Ticker); ok {
			// start GoTick in a goroutine to let all components work in parallel while sharing the same context
			go func() {
				defer m.panicRecovery("GoTick", component.(Core))
				s.GoTick(ctx)
			}()
		}
		return true
	})
}

// TestFlush delegates the call to all components which implement the interface TestFlusher
func (m *Manager) TestFlush(ctx context.Context) {
	m.components.Range(func(name, component interface{}) bool {
		if s, ok := component.(TestFlusher); ok {
			s.TestFlush(ctx)
		}
		return true
	})
}

func (m *Manager) panicRecovery(action string, comp Core) {
	if r := recover(); r != nil {
		var msg string
		if err, ok := r.(error); ok {
			msg = fmt.Sprintf("error in componentmanager: %s of %s failed with panic: %s", action, comp.Name(), err.Error())
		} else {
			msg = fmt.Sprintf("error in componentmanager: %s of %s failed with panic", action, comp.Name())
		}

		log.Println(msg)
		if !m.unitTestMode {
			// send rollbar reports only in production mode
			// stack dumps are printed to the test log and mess things up
			rollbar.PanicRecovery(msg)
		}
	}
}
