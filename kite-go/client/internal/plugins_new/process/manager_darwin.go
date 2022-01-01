package process

import (
	"context"
	"os/exec"
	"strings"
	"syscall"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system/nsbundle"
)

var (
	commonPaths = []string{
		// don't include /sbin/ and others because that would be a weird place to put an editor
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
	}
	attributes = &syscall.SysProcAttr{}
)

// MacProcessManager handles process information on macOS
type MacProcessManager interface {
	BundleLocations(ctx context.Context, id string) ([]string, error)
	// IsBundleRunning reports if at least one application with the given bundle identifier is running
	IsBundleRunning(id string) bool
	// IsProcessRunning reports if at least one process with the given executable name is running
	IsProcessRunning(exeName string) (bool, error)
	// RunningApplications returns a list of running applications. Pure terminal applications are not included.
	RunningApplications() (List, error)
	// RunningCliApplications returns a list of running terminal applications and bundles. RunningAppliations is to be
	// preferred if you only need information about an application bundle
	AllRunningApplications() (SimpleList, error)
	AppVersion(appPath string) (string, error)
	Run(name string, arg ...string) ([]byte, error)
	FindBinary(name string) []string
	Home() (string, error)
}

// Process provides basic information on a running process
type Process struct {
	Pid            int
	BundleID       string
	BundleLocation string
}

// SimpleProcess provides basic information on a running process
type SimpleProcess struct {
	Executable string
	Arguments  []string
}

// SimpleList is a list of cli processes
type SimpleList []SimpleProcess

// List is a list of processes
type List []Process

// Matching returns all non-empty returned by the filter function
// the filter function is invoked once for each of the processes
func (l List) Matching(filter func(process Process) string) []string {
	if len(l) == 0 {
		return nil
	}

	var matching []string
	for _, p := range l {
		if v := filter(p); v != "" {
			matching = append(matching, v)
		}
	}
	return matching
}

// NewManager returns a new MacProcessManager
func NewManager() MacProcessManager {
	return &macProcessManager{}
}

type macProcessManager struct{}

// BundleLocations returns the detected locations of bundles with the given CFBundleIdentifier property.
// This relies on spotlight indexing. If spotlight is disabled then no bundles will be detected.
// The corresponding API method is LSCopyApplicationURLsForBundleIdentifier, if mdfind turns out to be too slow
// then this is a possible fix.
func (m *macProcessManager) BundleLocations(ctx context.Context, id string) ([]string, error) {
	out, err := exec.CommandContext(ctx, "mdfind", "kMDItemCFBundleIdentifier", "==", id).Output()
	if err != nil {
		return nil, err
	}

	var valid []string
	for _, x := range strings.Split(string(out), "\n") {
		if x != "" {
			valid = append(valid, x)
		}
	}
	return valid, nil
}

// IsBundleRunning implements MacProcessManager.
// It returns true if an application with the given bundle identifier is running.
func (m *macProcessManager) IsBundleRunning(id string) bool {
	// fixme move AppRunning into this package
	running, err := nsbundle.AppRunning(id)
	return err == nil && running
}

// IsProcessRunning uses ps to get a list of running processes on the system and checks id
// against them.
func (m *macProcessManager) IsProcessRunning(exeName string) (bool, error) {
	list, err := m.AllRunningApplications()
	if err != nil {
		return false, err
	}

	for _, proc := range list {
		if proc.Executable == exeName {
			return true, nil
		}
	}
	return false, nil
}

// RunningApplications implements ProcessManager.
// It returns the list of currently running processes
func (m *macProcessManager) RunningApplications() (List, error) {
	return GetRunningApplications()
}

// RunningApplications implements ProcessManager.
// It returns the list of currently running processes
func (m *macProcessManager) AllRunningApplications() (SimpleList, error) {
	processes, err := exec.Command("ps", "ax", "-o", "command").Output()
	if err != nil {
		return nil, err
	}

	var list SimpleList
	for _, proc := range strings.Split(string(processes), "\n") {
		cmd := strings.Split(proc, " ")
		var args []string
		if len(cmd) >= 2 {
			args = cmd[1:]
		}
		list = append(list, SimpleProcess{
			Executable: cmd[0],
			Arguments:  args,
		})
	}
	return list, nil
}

func (m *macProcessManager) AppVersion(appPath string) (string, error) {
	return nsbundle.AppVersion(appPath)
}

// Run executes a command in a subprocess and returns its output:
//  - command is logged always
//  - stdout and stderr are logged on error
//  - returns stdout only
//  - returns a ProcessError which wraps stdout and stderr when the command failed
func (m *macProcessManager) Run(name string, arg ...string) ([]byte, error) {
	return runProcess(name, nil, arg...)
}

// Finds any executables that lives in commonPaths, e.g. Vim, Neovim.
func (m *macProcessManager) FindBinary(name string) []string {
	return findBinary(name)
}

// Home returns the current user's home directory, or an error if it can't be found.
func (m *macProcessManager) Home() (string, error) {
	return homeDir()
}
