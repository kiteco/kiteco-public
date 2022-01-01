package watch

import (
	"context"
	"os"

	"github.com/kiteco/fsevents"
	"github.com/kiteco/kiteco/kite-go/localfiles"
)

// Type returns the type of file event that has occurred.
func eventType(e fsevents.Event) localfiles.EventType {
	if e.Flags&fsevents.ItemRemoved == fsevents.ItemRemoved {
		return localfiles.RemovedEvent
	}
	if e.Flags&fsevents.ItemCreated == fsevents.ItemCreated ||
		e.Flags&fsevents.ItemModified == fsevents.ItemModified ||
		e.Flags&fsevents.ItemRenamed == fsevents.ItemRenamed ||
		e.Flags&fsevents.ItemInodeMetaMod == fsevents.ItemInodeMetaMod {
		return localfiles.ModifiedEvent
	}
	if _, err := os.Stat(e.Path); os.IsNotExist(err) {
		// If the file no longer exists, delete it.
		return localfiles.RemovedEvent
	}
	return localfiles.UnrecognizedEvent
}

func filesystem(ctx context.Context, paths []string, ch chan []Event, readyChan chan bool, opts Options) (Filesystem, error) {
	// start fs event stream
	events := make(chan []fsevents.Event, 100)
	stream := fsevents.EventStream{
		Paths:   paths,
		Latency: opts.Latency,
		Events:  events,
		Flags:   fsevents.FileEvents | fsevents.WatchRoot,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				stream.Stop()
				return
			case msg := <-events:
				var out []Event
				for _, e := range msg {
					t := eventType(e)
					if t == localfiles.UnrecognizedEvent {
						continue
					}
					out = append(out, Event{Path: e.Path, Type: t})
				}
				if len(out) > 0 {
					select {
					case ch <- out:
					default:
						if opts.OnDrop != nil {
							opts.OnDrop()
						}
					}
				}
			}
		}
	}()

	stream.Start()

	// notify that the watcher is now operational
	readyChan <- true
	close(readyChan)

	return &noopFilesystem{}, nil
}
