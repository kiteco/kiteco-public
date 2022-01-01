package diskmap

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Builder builds and serializes a diskmap file.
type Builder struct {
	data map[string][]byte
}

// NewBuilder returns a new Builder object.
func NewBuilder() *Builder {
	return &Builder{
		data: make(map[string][]byte),
	}
}

// Add adds the provided key/value to the builder. If the key already was added, it will
// be replaced with the new value.
func (m *Builder) Add(key string, value []byte) error {
	m.data[key] = value
	return nil
}

// WriteToFile will serialize the diskmap object to the provided path.
func (m *Builder) WriteToFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = m.WriteTo(f)
	return err
}

// WriteTo serializes to a io.Writer
func (m *Builder) WriteTo(w io.Writer) (int64, error) {
	var entries []mapEntry
	for key, value := range m.data {
		entries = append(entries, mapEntry{key, value})
	}

	m.data = nil
	sort.Sort(byKey(entries))

	var offset int
	var entryBuf bytes.Buffer
	var index []indexEntry
	keyBuf := make([]byte, binary.MaxVarintLen64)

	block := 5 // TODO(tarak): Parameterize? Probably not...
	for idx, entry := range entries {
		if idx%block == 0 {
			// Only add every Nth entry to the index. This reduces the memory footprint of
			// Map (because it needs to load the index). Because keys are sorted, we can find
			// the closest key and read through the subsequent N entries on disk until we find
			// the key we are looking for. TL;DR: Higher N -> smaller memory footprint, slower
			// lookup speed. Lower N -> larger memory footprint, faster lookup speed.
			index = append(index, indexEntry{entry.key, offset})
		}

		// encode the key/value pair
		n, err := encodeKeyValue(w, entry.key, entry.data, keyBuf, entryBuf)
		if err != nil {
			return -1, err
		}

		offset += n
	}

	// encode the sentinal key/value pair
	n, err := encodeKeyValue(w, lastEntrySentinal, []byte(lastEntrySentinal), keyBuf, entryBuf)
	if err != nil {
		return -1, err
	}

	// update the offset
	offset += n

	footer := indexFooter{
		Version:      1, // change this when format changes
		Index:        index,
		BlockEntries: block,
		Len:          len(entries),
	}

	// Use entryBuf here so we can get the number of bytes written
	entryBuf.Reset()
	if err := gob.NewEncoder(&entryBuf).Encode(&footer); err != nil {
		return -1, err
	}

	n, err = w.Write(entryBuf.Bytes())
	if err != nil {
		return -1, err
	}
	total := offset + n

	// Encode offset where the footer started
	if err := binary.Write(w, binary.BigEndian, int64(offset)); err != nil {
		return -1, err
	}

	// Add 8 bytes for the int64 encoded at the end
	return int64(total) + int64(8), nil
}

// --

const (
	lastEntrySentinal = "_diskmap_end"
)

type indexFooter struct {
	Version      int
	Index        []indexEntry
	BlockEntries int
	Len          int
}

type indexEntry struct {
	Key    string
	Offset int
}

// --

type mapEntry struct {
	key  string
	data []byte
}

type byKey []mapEntry

func (b byKey) Len() int           { return len(b) }
func (b byKey) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byKey) Less(i, j int) bool { return b[i].key < b[j].key }

// --

func encodeKeyValue(w io.Writer, key string, value []byte, keyBuf []byte, entryBuf bytes.Buffer) (int, error) {
	var written int
	// Encode the individual entry as < varint(sizeof(key)), key, value >
	entryBuf.Reset()
	n := binary.PutVarint(keyBuf, int64(len(key)))
	entryBuf.Write(keyBuf[:n])
	entryBuf.WriteString(key)
	entryBuf.Write(value)

	// Encode the whole entry as < varint(sizeof(entry)), entry >
	n = binary.PutVarint(keyBuf, int64(entryBuf.Len()))
	nn, err := w.Write(keyBuf[:n])
	if err != nil {
		return -1, err
	}
	written += nn
	nn, err = w.Write(entryBuf.Bytes())
	if err != nil {
		return -1, err
	}
	written += nn

	return written, nil
}
