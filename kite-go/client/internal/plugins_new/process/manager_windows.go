package process

import (
	"context"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/mattn/go-zglob"
	"github.com/shirou/gopsutil/process"
	"github.com/shirou/w32"
	"github.com/winlabs/gowin32"
	"github.com/winlabs/gowin32/wrappers"
)

var (
	// Needed for shared.go's findBinary to compile properly.
	commonPaths = []string{}
	attributes  = &syscall.SysProcAttr{HideWindow: true, CreationFlags: wrappers.CREATE_NO_WINDOW}
)

// WindowsProcessManager handles process information on macOS
type WindowsProcessManager interface {
	IsProcessRunning(ctx context.Context, name string) (bool, error)
	List(ctx context.Context) (List, error)

	ExeProductVersion(exePath string) string
	FilterStartMenuTargets(p func(string) bool) []string
	Run(name string, arg ...string) ([]byte, error)
	RunWithEnv(name string, additionalEnv []string, arg ...string) ([]byte, error)
	FindBinary(name string) []string
	Home() (string, error)
	LocalAppData() (string, error)
}

// NewManager returns a new WindowsProcessManager
func NewManager() WindowsProcessManager {
	return &processManager{}
}

type processManager struct{}

// IsProcessRunning returns true if a process with the given executable name is running. It returns an error if the retrieval of the process list failed.
func (m *processManager) IsProcessRunning(ctx context.Context, name string) (bool, error) {
	processes, err := gowin32.GetProcesses()
	if err != nil {
		return false, err
	}
	for _, p := range processes {
		if p.ExeFile == name {
			return true, nil
		}
	}

	return false, nil
}

// List implements ProcessManager.
// It returns the list of currently running processes
func (m *processManager) List(ctx context.Context) (List, error) {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(snapshot)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := syscall.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}

	processList := make([]Process, 0, 50)
	for {
		// don't break on errors, a process may be short-lived or accessing it may need more permissions
		if exePath, err := readFullExePath(entry); err == nil {
			name := readTerminatedString(entry.ExeFile)
			processList = append(processList, winProcess{
				pid:  entry.ProcessID,
				name: name,
				exe:  exePath,
			})
		}

		// windows sends ERROR_NO_MORE_FILES on last process
		err = syscall.Process32Next(snapshot, &entry)
		if err == syscall.ERROR_NO_MORE_FILES {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return processList, nil
}

func readFullExePath(processEntry syscall.ProcessEntry32) (string, error) {
	// using gopsutil's w32 package because syscall isn't offering Module32First and related functionality
	handle := w32.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPMODULE, processEntry.ProcessID)
	defer w32.CloseHandle(handle)

	var module w32.MODULEENTRY32
	module.Size = uint32(unsafe.Sizeof(module))

	if ok := w32.Module32First(handle, &module); !ok {
		return "", errors.Errorf("Module32First failed")
	}
	return readTerminatedString(module.SzExePath), nil
}

func readTerminatedString(value [260]uint16) string {
	max := len(value) - 1
	end := 0
	for {
		if value[end] == 0 || end >= max {
			return syscall.UTF16ToString(value[:end])
		}
		end++
	}
}

// Run executes a command in a subprocess and returns its output:
//  - command is logged always
//  - stdout and stderr are logged on error
//  - returns stdout only
//  - returns a ProcessError which wraps stdout and stderr when the command failed
func (m *processManager) Run(name string, arg ...string) ([]byte, error) {
	return runProcess(name, nil, arg...)
}

// RunWithEnv executes a command in a subprocess with a modified environment and returns its output.
// The process is started with this process's env and additionalEnv appended to that.
//  - command is logged always
//  - stdout and stderr are logged on error
//  - returns stdout only
//  - returns a ProcessError which wraps stdout and stderr when the command failed
func (m *processManager) RunWithEnv(name string, additionalEnv []string, arg ...string) ([]byte, error) {
	return runProcess(name, additionalEnv, arg...)
}

// ExeProductVersion returns the product version string of the given exe file.
// It returns an empty string if the version isn't defined or when an error occurred.
func (m *processManager) ExeProductVersion(exePath string) string {
	buf, err := gowin32.GetFileVersion(exePath)
	if err != nil {
		return ""
	}
	fileInfo, err := buf.GetFixedFileInfo()
	if err != nil {
		return ""
	}
	return fileInfo.ProductVersion.String()
}

// FilterStartMenuTargets returns programs in the start menu satisfying predicate p.
func (m *processManager) FilterStartMenuTargets(p func(string) bool) []string {
	var targets []string
	for _, s := range allShortcutFiles() {
		t, err := readTarget(s)
		if err == nil && p(t) {
			targets = append(targets, t)
		}
	}
	return targets
}

// Finds any executables that lives in commonPaths, e.g. Vim, Neovim.
func (m *processManager) FindBinary(name string) []string {
	return findBinary(name)
}

// Home returns the current user's home directory, or an error if it can't be found.
func (m *processManager) Home() (string, error) {
	return homeDir()
}

// LocalAppData returns the current user's ~/AppData/Local directory.
func (m *processManager) LocalAppData() (string, error) {
	return gowin32.GetKnownFolderPath(gowin32.KnownFolderLocalAppData)
}

func allShortcutFiles() []string {
	var all []string
	all = append(all, shortcuts(gowin32.KnownFolderCommonStartMenu)...)
	all = append(all, shortcuts(gowin32.KnownFolderStartMenu)...)
	return all
}

func shortcuts(base gowin32.KnownFolder) []string {
	p, err := gowin32.GetKnownFolderPath(base)
	if err != nil {
		return []string{}
	}
	paths, _ := zglob.Glob(filepath.Join(p, "**", "*.lnk"))
	return paths
}

func readTarget(lnkFile string) (string, error) {
	// https://msdn.microsoft.com/en-us/subscriptions/xk6kst2k(v=vs.84).aspx
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)

	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return "", err
	}
	defer oleShellObject.Release()

	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return "", err
	}
	defer wshell.Release()

	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", lnkFile)
	if err != nil {
		return "", err
	}
	defer cs.Clear()

	target, err := oleutil.GetProperty(cs.ToIDispatch(), "TargetPath")
	if err != nil {
		return "", err
	}
	defer target.Clear()

	return target.ToString(), nil
}

type winProcess struct {
	pid  uint32
	name string
	exe  string
}

func (p winProcess) NameWithContext(ctx context.Context) (string, error) {
	return p.name, nil
}

func (p winProcess) CmdlineSliceWithContext(ctx context.Context) ([]string, error) {
	if info, err := process.GetWin32ProcWithContext(ctx, int32(p.pid)); err == nil && len(info) == 1 {
		return strings.Split(*info[0].CommandLine, " "), nil
	}
	// fallback to executable path
	return []string{p.exe}, nil
}

func (p winProcess) ExeWithContext(ctx context.Context) (string, error) {
	return p.exe, nil
}
