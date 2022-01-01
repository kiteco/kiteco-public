package event

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	idleFlushTimeout = 15 * time.Minute
)

// FilterFunc defines a function that applies a filter to
// the input event. It returns true if the event passes the
// filter, otherwise false.
type FilterFunc func(ev *Event) bool

// Driver wraps the streams and block writers for a particular
// user, as well as the block file system and metadata manager.
type Driver struct {
	uid     int64
	machine string

	blockSize int
	fs        blockFileSystem
	manager   *MetadataManager

	mutex     sync.Mutex
	streams   map[string]FilterFunc
	writers   map[string]*blockWriter
	lastEvent map[string]time.Time
}

func newDriver(uid int64, machine string, blockSize int, fs blockFileSystem, manager *MetadataManager) *Driver {
	d := &Driver{
		uid:       uid,
		machine:   machine,
		blockSize: blockSize,
		fs:        fs,
		manager:   manager,
		streams:   make(map[string]FilterFunc),
		writers:   make(map[string]*blockWriter),
		lastEvent: make(map[string]time.Time),
	}
	d.printf("created")
	return d
}

// HandleEvent handles writing incoming events.
//
// It applies each stream's filter to the input event
// and writes the event if the filter passes.
func (d *Driver) HandleEvent(ev *Event) string {
	start := time.Now()
	defer func() {
		totalDuration.RecordDuration(time.Since(start))
	}()

	// Ignore terminal events that are not complete commands
	if IsTerminal(ev) && !IsTerminalCommand(ev) {
		return ""
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	for stream, filter := range d.streams {
		if filter(ev) {
			writer, exists := d.writers[stream]
			if !exists {
				writer = newBlockWriter(d.uid, d.machine, d.blockSize, stream, d.fs, d.manager)
				d.writers[stream] = writer
			}
			d.lastEvent[stream] = time.Now()
			err := writer.writeEvent(ev)
			if err != nil {
				d.printf("error writing event for user %d: %s", d.uid, err)
			}
		}
	}
	return ""
}

// AddStream adds a new stream defined by the filter function fn
// to the driver's stream registry.
func (d *Driver) AddStream(name string, fn FilterFunc) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.streams[name] = fn
}

// Iterator returns a new event Iterator.
func (d *Driver) Iterator(name string) (*Iterator, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if writer, exists := d.writers[name]; exists {
		buf := &bytes.Buffer{}
		err := writer.block.writeTo(buf)
		if err != nil {
			return nil, err
		}
		return newIteratorWithReader(d.uid, name, d.fs, d.manager, buf)
	}

	return newIterator(d.uid, name, d.fs, d.manager)
}

// Flush flushes all the driver's writers.
func (d *Driver) Flush() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for _, writer := range d.writers {
		writer.flush()
	}
}

// Wait blocks until all the driver's writers have finished writing.
func (d *Driver) Wait() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for _, writer := range d.writers {
		writer.wait()
	}
}

// --

func (d *Driver) flushTicker() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for stream, writer := range d.writers {
		if lastWrite, ok := d.lastEvent[stream]; ok {
			if time.Since(lastWrite) > idleFlushTimeout {
				writer.flush()
			}
		}
	}
}

// --

func (d *Driver) printf(msg string, vars ...interface{}) {
	log.Printf("event.Driver (%d, %s) %s", d.uid, d.machine, fmt.Sprintf(msg, vars...))
}
