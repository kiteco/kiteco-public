package process

import (
	"context"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// MockManager is a mock implementation of WindowsProcessManager
type MockManager struct {
	// RunningProcesses maps a command name to true/false. A process not contained here will be reported as not running.
	IsProcessRunningData func(name string) (bool, error)
	VersionData          func(exePath string) string
	ListData             func() (List, error)
	// StartMenuData returns a set of unfiltered data
	StartMenuData func() []string
	// RunResult maps a command string to the result. Supported result types are []byte, string, and error
	RunResult func(name string, arg ...string) ([]byte, error)
	// RunResultWithEnv maps a command string to the result. Supported result types are []byte, string, and error
	RunResultWithEnv func(name string, additionalEnv []string, arg ...string) ([]byte, error)
	BinaryLocation   func(id string) []string
	CustomDir        func() (string, error)
}

// IsProcessRunning implements WindowsProcessManager
func (m *MockManager) IsProcessRunning(ctx context.Context, name string) (bool, error) {
	if m.IsProcessRunningData == nil {
		return false, errors.Errorf("mock: no data available for %s", name)
	}
	return m.IsProcessRunningData(name)
}

// ExeProductVersion implements WindowsProcessManager
func (m *MockManager) ExeProductVersion(exePath string) string {
	if m.VersionData == nil {
		return ""
	}
	return m.VersionData(exePath)
}

// Run implements WindowsProcessManager
func (m *MockManager) Run(name string, arg ...string) ([]byte, error) {
	if m.RunResult == nil {
		return nil, errors.Errorf("mock: unable to execute %s", name)
	}
	return m.RunResult(name, arg...)
}

// RunWithEnv implements WindowsProcessManager
func (m *MockManager) RunWithEnv(name string, additionalEnv []string, arg ...string) ([]byte, error) {
	if m.RunResultWithEnv == nil {
		return nil, errors.Errorf("mock: unable to execute %s", name)
	}
	return m.RunResultWithEnv(name, additionalEnv, arg...)
}

// FilterStartMenuTargets returns matching entries of the start menu
func (m *MockManager) FilterStartMenuTargets(p func(string) bool) []string {
	if m.StartMenuData == nil {
		return []string{}
	}

	data := m.StartMenuData()
	var result []string
	for _, d := range data {
		if p(d) {
			result = append(result, d)
		}
	}
	return result
}

// FindBinary implements WindowsProcessManager
func (m *MockManager) FindBinary(name string) []string {
	return m.BinaryLocation(name)
}

// Home implements WindowsProcessManager
func (m *MockManager) Home() (string, error) {
	if m.CustomDir == nil {
		return homeDir()
	}
	return m.CustomDir()
}

// LocalAppData implements WindowsProcessManager
func (m *MockManager) LocalAppData() (string, error) {
	return m.CustomDir()
}

// List implements WindowsProcessManager.
func (m *MockManager) List(ctx context.Context) (List, error) {
	if m.ListData != nil {
		return m.ListData()
	}
	return []Process{}, nil
}
