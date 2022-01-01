package event

import (
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/stretchr/testify/assert"
)

func Test_WriteEvent(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	fs := newInMemoryBlockFileSystem()
	bw := newBlockWriter(1, "", 1000, "test", fs, mm)

	// Size of this event is 860 bytes
	event := &Event{
		UserId:    proto.Int64(0),
		Filename:  proto.String(string(make([]byte, 32))),
		Source:    proto.String(string(make([]byte, 32))),
		Action:    proto.String(string(make([]byte, 32))),
		Text:      proto.String(string(make([]byte, 32))),
		Timestamp: proto.Int64(time.Now().UnixNano()),
	}

	// Add first event
	err := bw.writeEvent(event)
	assert.Nil(t, err, "expected write event to succeed")
	// Wait for any flushing to complete
	bw.wait()

	// This event should be in memory
	assert.Len(t, fs.files, 0, "num files written should be 0")

	// Add second event
	err = bw.writeEvent(event)
	assert.Nil(t, err, "expected write event to succeed")
	// Wait for any flushing to complete
	bw.wait()

	// Adding second event should have caused a flush
	assert.Len(t, fs.files, 1, "num files written should be 1")
}

func Test_EventExceedsBlockSize(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	fs := newInMemoryBlockFileSystem()
	bw := newBlockWriter(1, "", 150, "test", fs, mm)

	// This is under the block size (size 140)
	event := &Event{
		UserId:    proto.Int64(0),
		Filename:  proto.String(string(make([]byte, 2))),
		Source:    proto.String(string(make([]byte, 2))),
		Action:    proto.String(string(make([]byte, 2))),
		Text:      proto.String(string(make([]byte, 2))),
		Timestamp: proto.Int64(time.Now().UnixNano()),
	}
	err := bw.writeEvent(event)
	assert.Nil(t, err, "expected write event to succeed")

	// This event exceeds the block size, (size 860)
	event = &Event{
		UserId:    proto.Int64(0),
		Filename:  proto.String(string(make([]byte, 32))),
		Source:    proto.String(string(make([]byte, 32))),
		Action:    proto.String(string(make([]byte, 32))),
		Text:      proto.String(string(make([]byte, 32))),
		Timestamp: proto.Int64(time.Now().UnixNano()),
	}
	err = bw.writeEvent(event)
	assert.Nil(t, err, "expected write event to succeed")

	// Wait for flushing to complete
	bw.wait()
	// Adding second event should have caused a flush
	assert.Len(t, fs.files, 1, "num files written should be 1")

	// This event is under the block size, (size 140)
	event = &Event{
		UserId:    proto.Int64(0),
		Filename:  proto.String(string(make([]byte, 2))),
		Source:    proto.String(string(make([]byte, 2))),
		Action:    proto.String(string(make([]byte, 2))),
		Text:      proto.String(string(make([]byte, 2))),
		Timestamp: proto.Int64(time.Now().UnixNano()),
	}
	err = bw.writeEvent(event)
	assert.Nil(t, err, "expected write event to succeed")

	// Wait for flushing to complete
	bw.wait()
	// Adding third event should have caused a flush
	assert.Len(t, fs.files, 2, "num files written should be 2")
}
