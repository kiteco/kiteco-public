package watch

import (
	"context"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/readdirchanges"
)

const (
	// latency is the time in milliseconds to wait between sending file
	// events. If multiple events occur in the meantime, they will be batched.
	latency = 500 * time.Millisecond
)

// Type returns the type of file event that has occurred.
func eventType(ev readdirchanges.Event) localfiles.EventType {
	switch ev.Action {
	case readdirchanges.Created, readdirchanges.Modified, readdirchanges.RenamedTo:
		return localfiles.ModifiedEvent
	case readdirchanges.Removed, readdirchanges.RenamedFrom:
		return localfiles.RemovedEvent
	default:
		return localfiles.UnrecognizedEvent
	}
}

// printErrors prints errors from a channel
func printErrors(ch <-chan error) {
	for err := range ch {
		log.Println(err)
	}
}

func filesystem(ctx context.Context, paths []string, ch chan []Event, readyChan chan bool, opts Options) (Filesystem, error) {
	for _, path := range paths {
		watchPath(ctx, path, ch, opts)
	}

	// notify that the watcher is now operational
	readyChan <- true
	close(readyChan)

	return &noopFilesystem{}, nil
}

func watchPath(ctx context.Context, path string, ch chan []Event, opts Options) (Filesystem, error) {
	stream, err := readdirchanges.New(ctx, path)
	if err != nil {
		return nil, errors.Errorf("error creating monitor for %s: %v", path, err)
	}

	go printErrors(stream.Errors)

	err = stream.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		for ev := range stream.Events {
			t := eventType(ev)
			if t == localfiles.UnrecognizedEvent {
				continue
			}

			msg := []Event{Event{
				Path: ev.Path,
				Type: t,
			}}

			// do not check ctx.Done here because we should read to end of
			// channel or else the underlying stream will be unable to clean
			// up.
			select {
			case ch <- msg:
			default:
				if opts.OnDrop != nil {
					opts.OnDrop()
				}
			}
		}
	}()
	return &noopFilesystem{}, nil
}
