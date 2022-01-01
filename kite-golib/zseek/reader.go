package zseek

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"

	"github.com/golang/snappy"
)

var (
	// ErrSeekBeyondFile is returned when attempting to seek beyond the real size of the file
	ErrSeekBeyondFile = errors.New("seeking beyond file")

	// ErrOffsetNotFound is returned when the offset does not map to a known offset
	ErrOffsetNotFound = errors.New("offset not found")
)

// Reader reads data written by Writer
type Reader struct {
	r   io.ReadSeeker
	cur int64

	entry int
	block []byte

	footer indexFooter
}

// NewReader creates a Reader using the provided io.ReadSeeker
func NewReader(r io.ReadSeeker) (*Reader, error) {
	reader := &Reader{r: r}
	err := reader.loadIndex()
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// Read implements io.Reader
func (r *Reader) Read(buf []byte) (int, error) {
	// Ensure we haven't reached the end of the file
	if r.entry >= len(r.footer.Index) {
		return 0, io.EOF
	}

	// If the block is empty, read it
	if len(r.block) == 0 {
		err := r.readBlock(r.entry)
		if err != nil {
			return 0, err
		}
	}

	// Compute the offset within the block we want to read
	boff := r.cur - r.footer.Index[r.entry].Offset

	// Read and update current offset
	n, err := bytes.NewReader(r.block[boff:]).Read(buf)
	r.cur += int64(n)

	// If io.EOF, increment r.entry, reset block and return. Subsequent calls
	// to Read will read the next block (if there are more blocks).
	if err == io.EOF {
		r.block = nil
		r.entry++
		return n, nil
	}

	return n, err
}

// Seek implements io.Seeker
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		// Nothing to do here
	case io.SeekEnd:
		// Set the offset relative to the end. offset will be a negative number here.
		offset = r.footer.End + offset
	case io.SeekCurrent:
		// Add offset to the current real offset
		offset = r.cur + offset
	}

	// Ensure we aren't trying to seek beyond the file's end
	if offset > r.footer.End {
		return -1, ErrSeekBeyondFile
	}

	// Find the compressed block with offset
	var ok bool
	r.entry, ok = r.offsetEntryIdx(offset)
	if !ok {
		return -1, ErrOffsetNotFound
	}

	// Reset the block. Block will be read when Read is called
	r.block = nil

	// Set current offset to what the user requested
	r.cur = offset

	return offset, nil
}

// --

// TODO(tarak): Worth making this a binary search for larger indexes
func (r *Reader) offsetEntryIdx(offset int64) (int, bool) {
	i := -1
	for idx, entry := range r.footer.Index {
		if entry.Offset > offset {
			return i, i != -1
		}
		i = idx
	}
	return i, i != -1
}

func (r *Reader) readBlock(i int) error {
	// Seek to the compressed offset for index entry i
	_, err := r.r.Seek(r.footer.Index[i].CompressedOffset, io.SeekStart)
	if err != nil {
		return err
	}

	// Read the compressed block
	var block bytes.Buffer
	_, err = io.CopyN(&block, r.r, int64(r.footer.Index[i].Size))
	if err != nil {
		return err
	}

	// Decompressed the block into r.block
	r.block, err = snappy.Decode(r.block, block.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (r *Reader) loadIndex() error {
	// Seek to relative to the end of the file to find the the 64-bit offset
	// where the indexFooter starts.
	_, err := r.r.Seek(int64(-8), io.SeekEnd)
	if err != nil {
		return err
	}

	// Read the footer offset
	var footerOffset int64
	err = binary.Read(r.r, binary.BigEndian, &footerOffset)
	if err != nil && err != io.EOF {
		return err
	}

	// Seek to the footer ofset
	_, err = r.r.Seek(footerOffset, io.SeekStart)
	if err != nil {
		return err
	}

	// Gob decode the indexFooter object
	if err := gob.NewDecoder(r.r).Decode(&r.footer); err != nil {
		return fmt.Errorf("error decoding footer index: %v", err)
	}

	return nil
}
