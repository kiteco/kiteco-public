package zseek

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/golang/snappy"
)

var defaultBlockSize = 256 << 20 // 256k

// Writer writes compressed blocks of data, when then can be Seeked to
// arbitrarily using Reader.
type Writer struct {
	bs      int
	w       io.Writer
	buf     bytes.Buffer
	compBuf []byte

	off  int64
	comp int64

	footer indexFooter
}

// NewWriter creates a Writer with the default block size.
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, defaultBlockSize)
}

// NewWriterSize creates a writer with a provided block size.
func NewWriterSize(w io.Writer, bs int) *Writer {
	return &Writer{
		w:  w,
		bs: bs,
	}
}

// Write implements io.Writer
func (w *Writer) Write(buf []byte) (int, error) {
	if len(buf) > w.available() {
		err := w.flush()
		if err != nil {
			return 0, err
		}
	}
	return w.buf.Write(buf)
}

// Close implements io.Closer
func (w *Writer) Close() error {
	if w.buf.Len() > 0 {
		err := w.flush()
		if err != nil {
			return err
		}
	}

	// Encode the footer, which contains a mapping from compressed offset
	// to real file offset.
	w.footer.End = w.off
	if err := gob.NewEncoder(w.w).Encode(w.footer); err != nil {
		return err
	}

	// Encode the location of the footer
	if err := binary.Write(w.w, binary.BigEndian, w.comp); err != nil {
		return fmt.Errorf("error encoding footer location: %v", err)
	}

	return nil
}

// --

func (w *Writer) available() int {
	return w.bs - w.buf.Len()
}

func (w *Writer) flush() error {
	// Compress the data accumulated in w.buf
	comp := snappy.Encode(w.compBuf, w.buf.Bytes())

	// Add an entry to the index
	w.footer.Index = append(w.footer.Index, indexEntry{
		Offset:           w.off,
		CompressedOffset: w.comp,
		Size:             len(comp),
	})

	// Increment real and compressed offsets
	w.off += int64(w.buf.Len())
	w.comp += int64(len(comp))
	w.buf.Reset()

	// Write compressed data to underlying writer
	_, err := w.w.Write(comp)
	return err
}

// --

type indexFooter struct {
	Index []indexEntry
	End   int64
}

type indexEntry struct {
	Offset           int64
	CompressedOffset int64
	Size             int
}
