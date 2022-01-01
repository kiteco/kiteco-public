package process

import (
	"context"
	"fmt"
)

// MockManager is a mock implementation of LinuxProcessManager
type MockManager struct {
	// RunningProcesses maps a command name to true/false. A process not contained here will be reported as not running.
	IsProcessRunningData func(name string) (bool, error)
	// RunResults maps a command string to the result. Supported result types are []byte, string, and error
	RunResult      func(name string, arg ...string) ([]byte, error)
	BinaryLocation func(id string) []string
	CustomDir      func() (string, error)
	ListData       func() (List, error)
}

// Run implements LinuxProcessManager
func (m *MockManager) Run(name string, arg ...string) ([]byte, error) {
	if m.RunResult == nil {
		return nil, fmt.Errorf("mock: unable to execute %s", name)
	}
	return m.RunResult(name, arg...)
}

// IsProcessRunning implements LinuxProcessManager
func (m *MockManager) IsProcessRunning(ctx context.Context, name string) (bool, error) {
	if m.IsProcessRunningData == nil {
		return false, nil
	}
	return m.IsProcessRunningData(name)
}

// FindBinary implements LinuxProcessManager.
func (m *MockManager) FindBinary(name string) []string {
	return m.BinaryLocation(name)
}

// Home implements LinuxProcessManager.
func (m *MockManager) Home() (string, error) {
	if m.CustomDir == nil {
		return homeDir()
	}
	return m.CustomDir()
}

// List implements LinuxProcessManager.
func (m *MockManager) List(ctx context.Context) (List, error) {
	if m.ListData != nil {
		return m.ListData()
	}
	return []Process{}, nil
}
