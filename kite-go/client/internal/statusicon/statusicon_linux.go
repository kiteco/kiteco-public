// +build linux

package statusicon

import (
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates"
)

// Manager is a dummy manager for linux, that disables the statusicon
type Manager struct{}

// NewManager returns a new Manager
func NewManager(updater updates.Manager) *Manager {
	return &Manager{}
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "statusicon"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
}

// SettingUpdated implements component.Settings
func (m *Manager) SettingUpdated(key, value string) {
}

// SettingDeleted implements component.Settings
func (m *Manager) SettingDeleted(key string) {
}

// LoggedIn implements component UserAuth
func (m *Manager) LoggedIn() {
}

// LoggedOut implements component UserAuth
func (m *Manager) LoggedOut() {
}
