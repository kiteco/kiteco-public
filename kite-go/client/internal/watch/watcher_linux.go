package watch

import (
	"context"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

type inotifyFilesystem struct {
	debug bool

	w  *fsnotify.Watcher
	mu sync.Mutex
	// we're assuming that there will only be a few registered watches for the same path.
	// 65535 should be a safe upper limit.
	// uint8 might be good enough, but we don't have any numbers on usage yet
	//fixme we could store hashed paths to save a bit of memory
	watchCounts map[string]uint16
}

func (fs *inotifyFilesystem) WatchCount() int64 {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// each key represents an inotify watch
	return int64(len(fs.watchCounts))
}

func (fs *inotifyFilesystem) Watch(fileOrDir string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.watchCounts[fileOrDir] == 0 {
		if err := fs.w.Add(fileOrDir); err != nil {
			return err
		}

		if fs.debug {
			log.Printf("AddWatch: %s, watched paths: %d", fileOrDir, len(fs.watchCounts))
		}
	} else if fs.debug {
		log.Printf("AddWatch: skipping Watch() of %s for count %d", fileOrDir, fs.watchCounts[fileOrDir])
	}

	fs.watchCounts[fileOrDir]++
	return nil
}

func (fs *inotifyFilesystem) Unwatch(fileOrDir string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.watchCounts[fileOrDir] == 0 {
		return errors.Errorf("unmatched call of Unwatch() for path %s", fileOrDir)
	}

	fs.watchCounts[fileOrDir]--
	if fs.watchCounts[fileOrDir] > 0 {
		if fs.debug {
			log.Printf("RemoveWatch: skipping Unwatch of %s for count %d", fileOrDir, fs.watchCounts[fileOrDir])
		}
		return nil
	}

	delete(fs.watchCounts, fileOrDir)
	if fs.debug {
		log.Printf("RemoveWatch: %s, watched paths after removal: %d. Paths: %v", fileOrDir, len(fs.watchCounts), fs.watchCounts)
	}

	return fs.w.Remove(fileOrDir)
}

func filesystem(ctx context.Context, paths []string, ch chan []Event, readyChan chan<- bool, opts Options) (Filesystem, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = watcher.Close()
				return
			case ev := <-watcher.Events:
				log.Printf("watcher event: %s, %d", ev.Name, ev.Op)

				t := eventType(ev)
				if t == localfiles.UnrecognizedEvent {
					continue
				}

				msg := []Event{{
					Path:       ev.Name,
					Type:       t,
					underlying: ev,
				}}

				select {
				case ch <- msg:
				default:
					if opts.OnDrop != nil {
						opts.OnDrop()
					}
				}
			}
		}
	}()

	fs := &inotifyFilesystem{
		w:           watcher,
		watchCounts: make(map[string]uint16),
	}

	for _, d := range paths {
		err = fs.Watch(d)
		if err != nil {
			return fs, err
		}
	}

	// notify that the watcher is now operational
	readyChan <- true
	close(readyChan)

	return fs, nil
}

// Type returns the type of file event that has occurred.
func eventType(ev fsnotify.Event) localfiles.EventType {
	switch ev.Op {
	case fsnotify.Create, fsnotify.Write, fsnotify.Rename:
		return localfiles.ModifiedEvent
	case fsnotify.Remove:
		return localfiles.RemovedEvent
	default:
		return localfiles.UnrecognizedEvent
	}
}
