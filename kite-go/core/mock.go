package core

import (
	"context"

	"github.com/kiteco/kiteco/kite-go/event"
)

// MockFileDriver is an implementation of FileDriver that can be initialized with a
// buffer and cursor position. It ignores all events.
type MockFileDriver struct {
	context context.Context
	buf     []byte
	cursor  int64
}

// NewMockFileDriver constructs a FileDriver that always returns the specified data
func NewMockFileDriver(ctx context.Context, buf []byte, cursor int64) *MockFileDriver {
	return &MockFileDriver{
		context: ctx,
		buf:     buf,
		cursor:  cursor,
	}
}

// HandleEvent does nothing
func (d *MockFileDriver) HandleEvent(*event.Event) {
}

// CollectOutput returns an empty list
func (d *MockFileDriver) CollectOutput() []interface{} {
	return []interface{}{}
}

// Bytes returns the contents of the buffer passed to the constructor
func (d *MockFileDriver) Bytes() []byte {
	return d.buf
}

// Cursor returns the cursor position passed to the constructor
func (d *MockFileDriver) Cursor() int64 {
	return d.cursor
}

// SetContents sets the contents of the buffer
func (d *MockFileDriver) SetContents(buf []byte) {
	d.buf = buf
}

// ResendText returns true if the full text of the file needs to
// be resent to the driver.
func (d *MockFileDriver) ResendText() bool {
	return false
}
