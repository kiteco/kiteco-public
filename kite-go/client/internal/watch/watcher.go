package watch

import (
	"context"
	"time"

	"github.com/kiteco/kiteco/kite-go/localfiles"
)

// Event represents a filesystem change event.
type Event struct {
	Path       string
	Type       localfiles.EventType
	underlying interface{} // platform-specific underlying event
}

// Options contains options for watching filesystems
type Options struct {
	OnDrop  func()        // function to be called if an event is dropped
	Latency time.Duration // interval over which events are batched together
}

// Filesystem allows to manages watches of files
type Filesystem interface {
	// Watch registers a new watch for the given file or directory.
	// If Watch was called n times, then Unwatch must also be called n times to
	// remove the registered watch.
	// If fileOrDir is a directory, then the watcher will send events for changes to the directory itself
	// and to files inside of that directory. Subdirectories or files inside of subdirectories of fileOrDir
	// are not watched, i.e. there's no recursive watching.
	Watch(fileOrDir string) error
	// Unwatch removes a watch.
	// If WatchFile was called n times, then UnwatchFile must also be called n times to
	// remove the registered watch.
	Unwatch(fileOrDir string) error
	// WatchCount returns the number of active watches
	WatchCount() int64
}

// noopFilesystem provides no-op implementations of AddWatch and RemoveWatch
// it's used on Windows and macOS
type noopFilesystem struct{}

// Watch of noopFilesystem does nothing
func (fs *noopFilesystem) Watch(filePath string) error {
	return nil
}

// Unwatch of noopFilesystem does nothing
func (fs *noopFilesystem) Unwatch(filePath string) error {
	return nil
}

// WatchCount of noopFilesystem always returns 1, because macOS and Windows are just watching $HOME
func (fs *noopFilesystem) WatchCount() int64 {
	return 1
}

// NewFilesystem listens for filesystem change events within the given
// paths. Each entry on the channel is a slice of events because the
// underlying OS APIs sometimes may batch events together for efficiency, so
// we do the same w.r.t. channels.
func NewFilesystem(ctx context.Context, paths []string, ch chan []Event, readyChan chan bool, opts Options) (Filesystem, error) {
	return filesystem(ctx, paths, ch, readyChan, opts)
}
