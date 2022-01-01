package diskmap

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
)

// StreamBuilder builds and serializes a diskmap file directly to a io.Writer. Keys
// must be added in sorted order.
type StreamBuilder struct {
	writer   io.Writer
	offset   int
	keyBuf   []byte
	entryBuf bytes.Buffer
	lastKey  string
	closed   bool
	footer   indexFooter
}

// NewStreamBuilder returns a new StreamBuilder object.
func NewStreamBuilder(w io.Writer) *StreamBuilder {
	footer := indexFooter{
		Version:      1,
		BlockEntries: 5,
	}
	return &StreamBuilder{
		writer: w,
		keyBuf: make([]byte, binary.MaxVarintLen64),
		footer: footer,
	}
}

// Add adds the provided key/value to the builder. If the key is lexigraphically smaller
// than the previous key, it will return an error.
func (m *StreamBuilder) Add(key string, value []byte) error {
	if m.closed {
		return fmt.Errorf("diskmap: StreamBuilder closed")
	}
	if key < m.lastKey {
		return fmt.Errorf("diskmap: StreamBuilder.Add keys must be sorted, got %s after %s", key, m.lastKey)
	}

	m.lastKey = key

	if m.footer.Len%m.footer.BlockEntries == 0 {
		// Only add every Nth entry to the index. This reduces the memory footprint of
		// Map (because it needs to load the index). Because keys are sorted, we can find
		// the closest key and read through the subsequent N entries on disk until we find
		// the key we are looking for. TL;DR: Higher N -> smaller memory footprint, slower
		// lookup speed. Lower N -> larger memory footprint, faster lookup speed.
		m.footer.Index = append(m.footer.Index, indexEntry{key, m.offset})
	}

	// encode the key/value pair
	n, err := encodeKeyValue(m.writer, key, value, m.keyBuf, m.entryBuf)
	if err != nil {
		return err
	}

	// update the offset
	m.offset += n

	// update the total entry count
	m.footer.Len++

	return nil
}

// Close closes the StreamBuilder and finishes writing to the the provided io.Writer
func (m *StreamBuilder) Close() error {
	// encode the sentinal key/value pair
	n, err := encodeKeyValue(m.writer, lastEntrySentinal, []byte(lastEntrySentinal), m.keyBuf, m.entryBuf)
	if err != nil {
		return err
	}

	// update the offset
	m.offset += n

	if err := gob.NewEncoder(m.writer).Encode(&m.footer); err != nil {
		return err
	}

	// Encode offset where the footer started
	if err := binary.Write(m.writer, binary.BigEndian, int64(m.offset)); err != nil {
		return err
	}

	m.closed = true
	return nil
}
