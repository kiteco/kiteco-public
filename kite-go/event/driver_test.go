package event

import (
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/stretchr/testify/assert"
)

func Test_HandleEventEmpty(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	opts := BlockStoreOptions{
		Type:      InMemoryStore,
		BlockSize: 1000, // 1000 bytes, uncompressed
	}
	store := NewBlockStore(mm, opts)
	driver := store.DriverForUser(1, "")
	defer store.RemoveDriver(driver)

	driver.HandleEvent(&Event{})
}

func Test_NoEvents(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	opts := BlockStoreOptions{
		Type:      InMemoryStore,
		BlockSize: 1000, // 1000 bytes, uncompressed
	}
	store := NewBlockStore(mm, opts)
	driver := store.DriverForUser(0, "")
	defer store.RemoveDriver(driver)

	driver.AddStream("testStream", func(ev *Event) bool { return true })

	// --

	// Iterator creation and calling Next should work when no events
	// have been added.
	iter, err := driver.Iterator("testStream")
	assert.NotNil(t, iter, "expected iterator to succeed")
	assert.Nil(t, err, "expected iterator to have no errors")

	success := iter.Next()
	assert.Equal(t, success, false, "expected next on empty stream to return false")
	assert.Nil(t, iter.Err(), "expected err to be nil when reading from empty stream")
}

func Test_EventsInMemory(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	opts := BlockStoreOptions{
		Type:      InMemoryStore,
		BlockSize: 1000, // 1000 bytes, uncompressed
	}
	store := NewBlockStore(mm, opts)
	driver := store.DriverForUser(0, "")
	defer store.RemoveDriver(driver)

	driver.AddStream("testStream", func(ev *Event) bool { return true })

	// --

	var writeEvents []*Event
	for i := 0; i < 10; i++ {
		event := &Event{
			UserId:    proto.Int64(0),
			Source:    proto.String("sublime-text"),
			Timestamp: proto.Int64(time.Now().UnixNano()),
		}
		writeEvents = append(writeEvents, event)
		driver.HandleEvent(event)
	}

	driver.Wait()
	// Ensure no blocks were flushed to fs
	var numFiles int
	switch v := store.fs.(type) {
	case *inMemoryBlockFileSystem:
		numFiles = len(v.files)
	}
	assert.Equal(t, numFiles, 0, "number of flushed blocks should be 0")

	iter, err := driver.Iterator("testStream")
	assert.NotNil(t, iter, "expected iterator to succeed")
	assert.Nil(t, err, "expected iterator to have no errors")

	var readEvents []*Event
	for iter.Next() {
		ev := iter.Event()
		readEvents = append(readEvents, ev)
		index := len(writeEvents) - len(readEvents)
		assert.Equal(t, ev, writeEvents[index])
	}
	assert.Nil(t, iter.Err(), "expected err to be nil")
	assert.Len(t, readEvents, 10, "number of read events should be 10")
}

func Test_EventsInMemoryAndFs(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	opts := BlockStoreOptions{
		Type:      InMemoryStore,
		BlockSize: 1000, // 1000 bytes, uncompressed
	}
	store := NewBlockStore(mm, opts)
	driver := store.DriverForUser(0, "")
	defer store.RemoveDriver(driver)

	driver.AddStream("testStream", func(ev *Event) bool { return true })

	// --

	var writeEvents []*Event
	for i := 0; i < 10; i++ {
		event := &Event{
			UserId:    proto.Int64(0),
			Source:    proto.String("sublime-text"),
			Timestamp: proto.Int64(time.Now().UnixNano()),
		}
		writeEvents = append(writeEvents, event)
		driver.HandleEvent(event)
	}

	driver.Flush()
	driver.Wait()
	// Ensure blocks were flushed to fs
	var numFiles int
	switch v := store.fs.(type) {
	case *inMemoryBlockFileSystem:
		numFiles = len(v.files)
	}
	assert.True(t, numFiles > 0)

	// Add event and ensure it is in memory
	event := &Event{}
	driver.HandleEvent(event)
	assert.NotEqual(t, opts.BlockSize, driver.writers["testStream"].block.available())
	writeEvents = append(writeEvents, event)

	iter, err := driver.Iterator("testStream")
	assert.NotNil(t, iter, "expected iterator to succeed")
	assert.Nil(t, err, "expected iterator to have no errors")

	var readEvents []*Event
	for iter.Next() {
		ev := iter.Event()
		readEvents = append(readEvents, ev)
		index := len(writeEvents) - len(readEvents)
		assert.Equal(t, ev, writeEvents[index])
	}
	assert.Nil(t, iter.Err(), "expected err to be nil")
	assert.Len(t, readEvents, 11, "number of read events should be 11")
}

func Test_EventsOnFs(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	opts := BlockStoreOptions{
		Type:      InMemoryStore,
		BlockSize: 1000, // 1000 bytes, uncompressed
	}
	store := NewBlockStore(mm, opts)
	driver := store.DriverForUser(0, "")
	defer store.RemoveDriver(driver)

	driver.AddStream("testStream", func(ev *Event) bool { return true })

	// --

	var writeEvents []*Event
	for i := 0; i < 10; i++ {
		event := &Event{
			UserId:    proto.Int64(0),
			Source:    proto.String("sublime-text"),
			Timestamp: proto.Int64(time.Now().UnixNano()),
		}
		writeEvents = append(writeEvents, event)
		driver.HandleEvent(event)
	}

	driver.Flush()
	driver.Wait()
	// Ensure blocks were flushed to fs
	var numFiles int
	switch v := store.fs.(type) {
	case *inMemoryBlockFileSystem:
		numFiles = len(v.files)
	}
	assert.True(t, numFiles > 0)

	iter, err := driver.Iterator("testStream")
	assert.NotNil(t, iter, "expected iterator to succeed")
	assert.Nil(t, err, "expected iterator to have no errors")

	var readEvents []*Event
	for iter.Next() {
		ev := iter.Event()
		readEvents = append(readEvents, ev)
		index := len(writeEvents) - len(readEvents)
		assert.Equal(t, ev, writeEvents[index])
	}
	assert.Nil(t, iter.Err(), "expected err to be nil")
	assert.Len(t, readEvents, 10, "number of read events should be 10")
}

func Test_MultipleBlockBatches(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	opts := BlockStoreOptions{
		Type:      InMemoryStore,
		BlockSize: 1000, // 1000 bytes, uncompressed
	}
	store := NewBlockStore(mm, opts)
	driver := store.DriverForUser(0, "")
	defer store.RemoveDriver(driver)

	driver.AddStream("testStream", func(ev *Event) bool { return true })

	// --

	var writeEvents []*Event
	// Each block contains one event, so 21 events will create
	// 3 batches of blocks when iterating.
	for i := 0; i < 21; i++ {
		event := &Event{
			UserId:    proto.Int64(0),
			Filename:  proto.String(strings.Repeat("a", 32)),
			Source:    proto.String(strings.Repeat("a", 32)),
			Action:    proto.String(strings.Repeat("a", 32)),
			Text:      proto.String(strings.Repeat("a", 32)),
			Timestamp: proto.Int64(time.Now().UnixNano()),
		}
		writeEvents = append(writeEvents, event)
		driver.HandleEvent(event)
		// It is necessary to do this each iteration because sqlite3 cannot
		// handle multiple connections.
		driver.Flush()
		driver.Wait()
	}

	// Ensure blocks were flushed to fs
	var numFiles int
	switch v := store.fs.(type) {
	case *inMemoryBlockFileSystem:
		numFiles = len(v.files)
	}
	assert.True(t, numFiles > 0)

	iter, err := driver.Iterator("testStream")
	assert.NotNil(t, iter, "expected iterator to succeed")
	assert.Nil(t, err, "expected iterator to have no errors")

	var readEvents []*Event
	for iter.Next() {
		ev := iter.Event()
		readEvents = append(readEvents, ev)
		index := len(writeEvents) - len(readEvents)
		assert.Equal(t, ev, writeEvents[index])
	}
	assert.Nil(t, iter.Err(), "expected err to be nil")
	assert.Len(t, readEvents, 21, "number of read events should be 10")
}

func Test_TerminalEvents(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	opts := BlockStoreOptions{
		Type:      InMemoryStore,
		BlockSize: 1000, // 1000 bytes, uncompressed
	}
	store := NewBlockStore(mm, opts)
	driver := store.DriverForUser(0, "")
	defer store.RemoveDriver(driver)

	driver.AddStream("testStream", func(ev *Event) bool { return true })

	// --

	command := &Event{
		UserId:    proto.Int64(0),
		Source:    proto.String("terminal"),
		Action:    proto.String("command"),
		Timestamp: proto.Int64(time.Now().UnixNano()),
	}
	driver.HandleEvent(command)

	nonCommand := &Event{
		UserId:    proto.Int64(0),
		Source:    proto.String("terminal"),
		Timestamp: proto.Int64(time.Now().UnixNano()),
	}
	driver.HandleEvent(nonCommand)

	// Iterator creation and calling Next should return only terminal commands.
	iter, err := driver.Iterator("testStream")
	assert.NotNil(t, iter, "expected iterator to succeed")
	assert.Nil(t, err, "expected iterator to have no errors")

	var events []*Event
	for iter.Next() {
		events = append(events, iter.Event())
	}
	assert.Nil(t, iter.Err(), "expected err to be nil when reading terminal events")
	assert.Len(t, events, 1, "expected number of events read to be 1")
	assert.Equal(t, events[0], command)
}
