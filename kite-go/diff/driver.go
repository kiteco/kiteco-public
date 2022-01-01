package diff

import (
	"crypto/md5"
	"fmt"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// BufferDriver is a driver for processing diffs.
type BufferDriver struct {
	cursor  int64
	patcher *Patcher

	// resend is used to indicate if the driver needs for the full text
	// of the file to be sent from the client.
	resend bool
}

// NewBufferDriver returns a pointer to a newly initialized BufferDriver.
func NewBufferDriver() *BufferDriver {
	return &BufferDriver{
		patcher: NewPatcher([]byte("")),
	}
}

// HandleEvent implements core.Driver.
func (b *BufferDriver) HandleEvent(ctx kitectx.Context, event *event.Event) string {
	ctx.CheckAbort()

	// If the driver ever recieves text, it assumes its the full text of the
	// buffer and replaces the existing buffer with the new text. How frequently and
	// when this happens is controlled by kited.
	text := event.GetText()
	diffs := event.GetDiffs()
	switch {
	case len(text) > 0:
		b.patcher = NewPatcher([]byte(text))
	case len(diffs) > 0:
		b.patcher.Apply(diffs)
	}

	for _, sel := range event.GetSelections() {
		if sel.GetStart() == sel.GetEnd() && sel.GetStart() != b.cursor {
			b.cursor = sel.GetStart()
		}
	}

	// Check that diffs were applied correctly by comparing hash of
	// new text with hash of file contents from event, if not matching
	// then set resend flag so that a `ResendText` response can be sent to the client,
	// if hashes match then make sure resend flag is reset.
	// First clause is to avoid false positives from older
	// clients in which the hash was not set.
	if event.GetTextChecksum() > 0 && event.GetTextChecksum() != spooky.Hash64(b.Bytes()) {
		// This can occur due to unicode characters, or this can occur due to issues
		// transforming from bytes -> string -> rune -> string -> bytes
		b.resend = true
		resendRatio.Hit()
	} else {
		resendRatio.Miss()
		b.resend = false
	}

	// return hash of state
	return b.state()
}

// state computes a hash of the buffer contents
func (b *BufferDriver) state() string {
	return fmt.Sprintf("%x", md5.Sum(b.Bytes()))
}

// CollectOutput implements core.Driver.
func (b *BufferDriver) CollectOutput() []interface{} {
	return nil
}

// Bytes returns a copy of the underlying byte slice for the driver's buffer.
func (b *BufferDriver) Bytes() []byte {
	return b.patcher.Bytes()
}

// Cursor returns the current position of the cursor.
func (b *BufferDriver) Cursor() int64 {
	return b.cursor
}

// SetContents sets the driver's buffer to the given byte slice.
func (b *BufferDriver) SetContents(buf []byte) {
	b.patcher.SetContents(buf)
}

// ResendText returns true if the FileDriver needs the full text of the file to be resent.
func (b *BufferDriver) ResendText() bool {
	return b.resend
}
