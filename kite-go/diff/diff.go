// Package diff is a simple wrapper around github.com/sergi/go-diff that converts
// the diffs return by that package into a more concise representation suitable for
// transmission over the wire.
package diff

import (
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Differ computes Patch objects (array of Diff), which is a compact
// representation of changes a buffer.
type Differ struct {
	dmp *diffmatchpatch.DiffMatchPatch
}

// NewDiffer returns a new Differ.
func NewDiffer() *Differ {
	return &Differ{
		dmp: diffmatchpatch.New(),
	}
}

// Diff takes two strings and returns a Patch.
func (d *Differ) Diff(text1, text2 string) []event.Diff {
	var localDiffs []event.Diff
	diffs := d.dmp.DiffMain(text1, text2, true)

	var offset int
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			// Ignore equal components. Simply update the offset.
			offset += len(diff.Text)

		case diffmatchpatch.DiffDelete:
			// Create a DiffDelete Diff, do not need to update the offset.
			localDiffs = append(localDiffs, event.Diff{
				Type:   event.DiffType.Enum(event.DiffType_DELETE),
				Text:   proto.String(diff.Text),
				Offset: proto.Int(offset),
			})

		case diffmatchpatch.DiffInsert:
			// Update the offset with the size of the inserted text.
			localDiffs = append(localDiffs, event.Diff{
				Type:   event.DiffType.Enum(event.DiffType_INSERT),
				Text:   proto.String(diff.Text),
				Offset: proto.Int(offset),
			})
			offset += len(diff.Text)
		}
	}
	return localDiffs
}

// Patcher applies patches to the provided buffer. Implemented using gap buffers.
type Patcher struct {
	buf *GapBuffer
}

// NewPatcher returns a pointer to a newly initialized Patcher.
func NewPatcher(buf []byte) *Patcher {
	return &Patcher{
		buf: NewGapBuffer(buf),
	}
}

// Bytes returns a copy of the underlying byte slice for the patcher's buffer.
func (p *Patcher) Bytes() []byte {
	return p.buf.Bytes()
}

// Apply processes the given diffs into the patcher.
func (p *Patcher) Apply(diffs []*event.Diff) {
	for _, diff := range diffs {
		switch diff.GetType() {
		case event.DiffType_INSERT:
			p.buf.Insert(int(diff.GetOffset()), []byte(diff.GetText()))
		case event.DiffType_DELETE:
			p.buf.Delete(int(diff.GetOffset()), []byte(diff.GetText()), true)
		}
	}
}

// SetContents sets the buffer to given byte slice.
func (p *Patcher) SetContents(buf []byte) {
	p.buf = NewGapBuffer(buf)
}
