package process

import (
	"context"
	"fmt"
)

// MockManager allows one to mock methods defined in MacProcessManager.
type MockManager struct {
	// set an empty []string array if you want it to return an empty list instead of an error
	BundleLocationsData func(id string) ([]string, error)
	// this defaults to false, i.e. it's only necessary to add values for running bundles
	IsBundleRunningData func(id string) bool
	// allows mocking of process running state
	IsProcessRunningData func(id string) (bool, error)
	// maps appPath to version info, missing entries result in errors
	VersionData func(appPath string) (string, error)
	// RunResults maps a command string to the result. Supported result types are []byte, string, and error
	RunResult                  func(name string, arg ...string) ([]byte, error)
	BinaryLocation             func(id string) []string
	CustomDir                  func() (string, error)
	RunningApplicationsData    func() (List, error)
	AllRunningApplicationsData func() (SimpleList, error)
}

// BundleLocations implements MacProcessManager.
func (m *MockManager) BundleLocations(ctx context.Context, id string) ([]string, error) {
	if m.BundleLocationsData == nil {
		return nil, fmt.Errorf("no bundle location defined in mock manager for id %s", id)
	}

	return m.BundleLocationsData(id)
}

// IsBundleRunning implements MacProcessManager.
func (m *MockManager) IsBundleRunning(id string) bool {
	if m.IsBundleRunningData == nil {
		return false
	}
	return m.IsBundleRunningData(id)
}

// IsProcessRunning implements MacProcessManager.
func (m *MockManager) IsProcessRunning(exeName string) (bool, error) {
	if m.IsProcessRunningData == nil {
		return false, nil
	}
	return m.IsProcessRunningData(exeName)
}

// AppVersion implements MacProcessManager.
func (m *MockManager) AppVersion(appPath string) (string, error) {
	if m.VersionData == nil {
		return "", fmt.Errorf("no version defined in mock manager for path %s", appPath)
	}
	return m.VersionData(appPath)
}

// Run implements MacProcessManager.
func (m *MockManager) Run(name string, arg ...string) ([]byte, error) {
	if m.RunResult == nil {
		return nil, fmt.Errorf("mock: unable to execute %s", name)
	}
	return m.RunResult(name, arg...)
}

// FindBinary implements MacProcessManager.
func (m *MockManager) FindBinary(name string) []string {
	return m.BinaryLocation(name)
}

// Home implements MacProcessManager.
func (m *MockManager) Home() (string, error) {
	if m.CustomDir == nil {
		return homeDir()
	}
	return m.CustomDir()
}

// RunningApplications implements MacProcessManager
func (m *MockManager) RunningApplications() (List, error) {
	if m.RunningApplicationsData != nil {
		return m.RunningApplicationsData()
	}
	return []Process{}, nil
}

// AllRunningApplications implements MacProcessManager
func (m *MockManager) AllRunningApplications() (SimpleList, error) {
	if m.AllRunningApplicationsData != nil {
		return m.AllRunningApplicationsData()
	}
	return []SimpleProcess{}, nil
}
