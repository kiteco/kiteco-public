package diskmap

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrNotFound is returned when the key-value could not be found.
	ErrNotFound = errors.New("key not found")
)

// Getter provides a read-only interface into the diskmap.
type Getter interface {
	Get(key string) ([]byte, error)
	Len() int
}

// Map represents a read-only map structure
// Deprecated: clients should use the Getter interface instead.
type Map struct {
	path   string
	footer indexFooter
}

// NewMap creates a new Map object using the diskmap file provided.
func NewMap(path string) (*Map, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	m := &Map{path: path}

	err := m.loadIndex()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Len returns the number of entries in the map.
func (m *Map) Len() int {
	return m.footer.Len
}

// Path returns the location of the diskmap file; the same value provided to NewMap
func (m *Map) Path() string {
	return m.path
}

// IterateSlowly scans over the entire diskmap and emits key/value pairs.
func (m *Map) IterateSlowly(emit func(key string, val []byte) error) error {
	dataf, err := os.Open(m.path)
	if err != nil {
		return err
	}
	defer dataf.Close()

	// Need a io.ByteReader so that binary.ReadVarint works. So just
	// wrap dataf in a bufio.Reader.
	bufr := bufio.NewReader(dataf)

	var numKeys int
	var entryBuf bytes.Buffer
	for numKeys < m.footer.Len {
		entrySize, err := binary.ReadVarint(bufr)
		switch err {
		case nil:
		case io.EOF:
			return nil
		default:
			return err
		}

		entryBuf.Reset()
		_, err = io.CopyN(&entryBuf, bufr, entrySize)
		if err != nil {
			return err
		}

		numKeys++
		key, val := entryKeyValue(entryBuf.Bytes())
		if err := emit(key, val); err != nil {
			return err
		}
	}
	return nil
}

// Keys returns all keys in the diskmap. Note that this function scans the entire file to construct
// a list of keys. Do not use this method lightly. (Useful for debugging purposes)
func (m *Map) Keys() ([]string, error) {
	var keys []string
	err := m.IterateSlowly(func(key string, _ []byte) error {
		keys = append(keys, key)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Get returns a []byte slice associated with the key, or an error
func (m *Map) Get(key string) ([]byte, error) {
	getCounter.Add(1)

	offset, ok := m.offsetForKey(key)
	if !ok {
		notFoundCounter.Add(1)
		return nil, ErrNotFound
	}

	dataf, err := os.Open(m.path)
	if err != nil {
		return nil, err
	}
	defer dataf.Close()

	_, err = dataf.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Need a io.ByteReader so that binary.ReadVarint works. So just
	// wrap dataf in a bufio.Reader.
	bufr := bufio.NewReader(dataf)

	getDiskCounter.Add(1)

	var nn int64
	var entryBuf bytes.Buffer
	for i := 0; i < m.footer.BlockEntries; i++ {
		entrySize, err := binary.ReadVarint(bufr)
		if err == io.EOF {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}

		entryBuf.Reset()
		n, err := io.CopyN(&entryBuf, bufr, entrySize)
		if err != nil {
			return nil, err
		}
		nn += n

		curKey, val := entryKeyValue(entryBuf.Bytes())
		if curKey == lastEntrySentinal {
			bytesReadPerGetSample.Record(nn)
			notFoundCounter.Add(1)
			return nil, ErrNotFound
		}

		if curKey == key {
			valueSizeSample.Record(int64(len(val)))
			bytesReadPerGetSample.Record(nn)

			// Create a copy of the value, we do this because
			// under the hood bytes.Buffer maintains a single
			// slice that grows over time and then bytes.Buffer.Bytes()
			// returns a pointer to this slice. This means that
			// val will store a pointer into a slice that has
			// capacity equal to the largest value we read
			// and this memory will not be freed until the
			// client is done using val.
			cv := make([]byte, len(val))
			copy(cv, val)
			return cv, nil
		}
	}

	bytesReadPerGetSample.Record(nn)
	notFoundCounter.Add(1)
	return nil, ErrNotFound
}

// --

func entryKeyValue(entry []byte) (string, []byte) {
	keySize, n := binary.Varint(entry)
	return string(entry[n : int64(n)+keySize]), entry[int64(n)+keySize:]
}

// --

// TODO(tarak): Worth making this a binary search for larger indexes
func (m *Map) offsetForKey(key string) (int, bool) {
	offset := -1
	for _, entry := range m.footer.Index {
		if entry.Key > key {
			return offset, offset != -1
		}
		offset = entry.Offset
	}

	return offset, offset != -1
}

func (m *Map) loadIndex() error {
	indexf, err := os.Open(m.path)
	if err != nil {
		return err
	}
	defer indexf.Close()

	// Seek to relative to the end of the file to find the the 64-bit offset
	// where the indexFooter starts.
	_, err = indexf.Seek(int64(-8), io.SeekEnd)
	if err != nil {
		return err
	}

	// Read the footer offset
	var footerOffset int64
	err = binary.Read(indexf, binary.BigEndian, &footerOffset)
	if err != nil && err != io.EOF {
		return err
	}

	// Seek to the footer ofset
	_, err = indexf.Seek(footerOffset, io.SeekStart)
	if err != nil {
		return err
	}

	// Gob decode the indexFooter object
	if err := gob.NewDecoder(indexf).Decode(&m.footer); err != nil {
		return fmt.Errorf("error decoding footer: %v", err)
	}

	return nil
}
