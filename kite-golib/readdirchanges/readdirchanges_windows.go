package readdirchanges

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Action is the enum for event types
type Action uint32

const (
	// Created is the action for file creation events
	Created Action = syscall.FILE_ACTION_ADDED
	// Removed is the action for file deletion events
	Removed Action = syscall.FILE_ACTION_REMOVED
	// Modified is the action for file modification events
	Modified Action = syscall.FILE_ACTION_MODIFIED
	// RenamedFrom is the action for rename events corresponding to the old name
	RenamedFrom Action = syscall.FILE_ACTION_RENAMED_OLD_NAME
	// RenamedTo is the action for rename events corresponding to the new name
	RenamedTo Action = syscall.FILE_ACTION_RENAMED_NEW_NAME
)

// String gets a string representation of an action.
func (a Action) String() string {
	switch a {
	case Created:
		return "Created"
	case Removed:
		return "Removed"
	case Modified:
		return "Modified"
	case RenamedFrom:
		return "RenamedFrom"
	case RenamedTo:
		return "RenamedTo"
	default:
		return fmt.Sprintf("Unknown(%d)", a)
	}
}

// Event contains information about a filesystem change.
type Event struct {
	Path   string // Path is always absolute and symlink-free
	Action Action
}

const bufferSize = 4096

var errAlreadyStarted = errors.New("the monitor is already running")
var errAlreadyStopped = errors.New("the monitor is already stopped")

// Monitor listens for windows filesystem events.
type Monitor struct {
	// these are used internally by the read loop:
	handle     syscall.Handle
	cph        syscall.Handle
	buffer     [bufferSize]byte
	overlapped *syscall.Overlapped

	// running is used to cancel the read loop
	running bool

	// cancel is used to cancel the read loop
	cancel func()

	// these are initialized in the constructor and then remain fixed:
	dir    string
	Events chan Event
	Errors chan error
}

// New creates a monitor that watches for the filesystem events in the tree
// rooted at the given directory.
func New(ctx context.Context, dir string) (*Monitor, error) {
	ctx, cancel := context.WithCancel(ctx) // so that we can cancel ourselves on error

	absdir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	rdir, err := filepath.EvalSymlinks(absdir)
	if err != nil {
		return nil, err
	}

	m := &Monitor{
		cancel: cancel,
		dir:    rdir,
		Events: make(chan Event, 100),
		Errors: make(chan error, 100),
	}
	go m.watchCancellation(ctx)
	return m, nil
}

// watchCancellation waits for a context to be cancelled, then stops the monitor. The underlying windows
// API is fundamentally a blocking API, so this is unavoidable.
func (m *Monitor) watchCancellation(ctx context.Context) error {
	// block until context is cancelled
	<-ctx.Done()

	err1 := syscall.CloseHandle(m.handle)
	err2 := syscall.CloseHandle(m.cph)
	m.running = false

	// close everything before returning any errors
	if err1 != nil {
		return os.NewSyscallError("CloseHandle", err1)
	}
	if err2 != nil {
		return os.NewSyscallError("CloseHandle", err2)
	}
	return nil
}

// close is called after the read loop terminates
func (m *Monitor) close() {
	close(m.Events)
	close(m.Errors)
}

// Start starts listening for events
func (m *Monitor) Start() error {
	if m.running {
		return errAlreadyStarted
	}

	m.running = true

	pdir, err := syscall.UTF16PtrFromString(m.dir)
	if err != nil {
		return os.NewSyscallError("UTF16PtrFromString", err)
	}

	m.handle, err = syscall.CreateFile(
		pdir,
		syscall.FILE_LIST_DIRECTORY,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OVERLAPPED,
		0,
	)

	if err != nil {
		return os.NewSyscallError("CreateFile", err)
	}

	m.cph, err = syscall.CreateIoCompletionPort(m.handle, 0, 0, 0)
	if err != nil {
		err2 := syscall.CloseHandle(m.handle)
		if err2 != nil {
			return os.NewSyscallError("CloseHandle", err2)
		}

		return os.NewSyscallError("CreateIoCompletionPort", err)
	}

	m.overlapped = &syscall.Overlapped{}

	// start reading events
	go m.loop()

	// fire off the first read
	err = m.readDirChanges(m.handle, &m.buffer[0], m.overlapped)
	if err != nil {
		return os.NewSyscallError("ReadDirectoryChanges", err)
	}

	return nil
}

func (m *Monitor) loop() {
	defer m.close()

	var n, key uint32
	var ov *syscall.Overlapped

	for {
		if !m.running {
			return
		}

		err := syscall.GetQueuedCompletionStatus(m.cph, &n, &key, &ov, syscall.INFINITE)
		if !m.running {
			return
		}

		switch err {
		case syscall.ERROR_MORE_DATA:
			if ov == nil {
				m.Errors <- fmt.Errorf("ERROR_MORE_DATA has unexpectedly null lpOverlapped buffer")
			} else {
				n = uint32(unsafe.Sizeof(m.buffer))
			}
		case syscall.ERROR_ACCESS_DENIED:
			// @todo, handle watched dir is removed
			continue
		case syscall.ERROR_OPERATION_ABORTED:
			continue
		default:
			m.Errors <- os.NewSyscallError("GetQueuedCompletionPort", err)
			continue
		case nil:
		}

		var offset uint32
		for {
			if n == 0 {
				// short read in readEvents()
				// continue with a new call to readDirChanges to retrieve a new set of changes
				break
			}

			raw := (*syscall.FileNotifyInformation)(unsafe.Pointer(&m.buffer[offset]))
			buf := (*[syscall.MAX_LONG_PATH]uint16)(unsafe.Pointer(&raw.FileName))
			if uint32(len(buf)) < raw.FileNameLength/2 {
				m.Errors <- fmt.Errorf("filename length exceeds buffer size")
				break
			}
			name := syscall.UTF16ToString(buf[:raw.FileNameLength/2])
			fullname := filepath.Clean(filepath.Join(m.dir, name))

			m.Events <- Event{
				Path:   fullname,
				Action: Action(raw.Action),
			}

			if raw.NextEntryOffset == 0 {
				break
			}

			offset += raw.NextEntryOffset
			if offset >= n {
				m.Errors <- fmt.Errorf("buffer exhausted, events have likely been missed")
				break
			}
		}

		// schedule new read if we didn't stop in the meantime
		if m.running {
			err = m.readDirChanges(m.handle, &m.buffer[0], m.overlapped)
			if err != nil {
				if err == syscall.ERROR_ACCESS_DENIED {
					m.cancel()
					continue
				}

				m.Errors <- os.NewSyscallError("ReadDirectoryChanges", err)
			}
		}
	}
}

func (m *Monitor) readDirChanges(h syscall.Handle, pBuff *byte, ov *syscall.Overlapped) error {
	return syscall.ReadDirectoryChanges(
		h,
		pBuff,
		uint32(bufferSize),
		true,
		syscall.FILE_NOTIFY_CHANGE_SIZE|syscall.FILE_NOTIFY_CHANGE_FILE_NAME|syscall.FILE_NOTIFY_CHANGE_DIR_NAME,
		nil,
		(*syscall.Overlapped)(unsafe.Pointer(ov)),
		0,
	)
}
