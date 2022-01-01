package fileutil

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
)

// Offset delimits the start and end of a file.
type Offset struct {
	Start int
	End   int
}

// FileOffset encapsulate a file path and its offset
type FileOffset struct {
	Path   string
	Offset Offset
}

type byPath []FileOffset

func (f byPath) Len() int           { return len(f) }
func (f byPath) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f byPath) Less(i, j int) bool { return f[i].Path < f[j].Path }

// FileMapWriter writes a set of files to a file map.
type FileMapWriter struct {
	data    *bytes.Buffer
	m       sync.Mutex
	offsets []FileOffset
}

// NewFileMapWriter creates a FileMapWriter
func NewFileMapWriter() *FileMapWriter {
	return &FileMapWriter{
		data: &bytes.Buffer{},
	}
}

// WriteOffsets takes a writer and writes the offset map.
func (fw *FileMapWriter) WriteOffsets(dest io.Writer) error {
	sort.Sort(byPath(fw.offsets))
	buf := new(bytes.Buffer)
	b64 := base64.NewEncoder(base64.StdEncoding, buf)
	encoder := gob.NewEncoder(b64)
	if err := encoder.Encode(fw.offsets); err != nil {
		return err
	}
	err := b64.Close()
	if err != nil {
		return err
	}
	_, err = dest.Write(buf.Bytes())
	return err
}

// WriteData takes a writer and writes the dataset buffer.
func (fw *FileMapWriter) WriteData(dest io.Writer) error {
	_, err := dest.Write(fw.data.Bytes())
	return err
}

// AddFile writes a file to the writer and adds it to the offset map
func (fw *FileMapWriter) AddFile(path string, src io.Reader) error {
	fw.m.Lock()
	defer fw.m.Unlock()
	var start, end int
	if fw.data != nil {
		start = fw.data.Len()
	}

	buf := &bytes.Buffer{}
	b64 := base64.NewEncoder(base64.StdEncoding, buf)
	_, err := io.Copy(b64, src)
	if err != nil {
		return err
	}
	err = b64.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(fw.data, buf)
	if err != nil {
		return err
	}
	if fw.data != nil {
		end = fw.data.Len()
	}
	fw.offsets = append(fw.offsets, FileOffset{
		Path: path,
		Offset: Offset{
			Start: start,
			End:   end,
		},
	})
	return nil
}

// --

// FileMap encapsulates a filemap
type FileMap interface {
	GetOffset(string) (Offset, bool)
	GetDataFile() (io.ReadSeeker, error)
}

// TestFileMap implements FileMap using local filesystem
type TestFileMap struct {
	dataPath string
	offsets  map[string]Offset
}

// NewTestFileMap creates a TestFileMap
func NewTestFileMap(dataPath, offsetPath string) (*TestFileMap, error) {
	tf := &TestFileMap{
		dataPath: dataPath,
	}
	f, err := os.Open(offsetPath)
	if err != nil {
		return nil, err
	}
	var fileOffsets []FileOffset
	b64 := base64.NewDecoder(base64.StdEncoding, f)
	decoder := gob.NewDecoder(b64)
	if err := decoder.Decode(&fileOffsets); err != nil {
		return nil, err
	}
	tf.offsets = make(map[string]Offset)
	for _, fo := range fileOffsets {
		tf.offsets[fo.Path] = fo.Offset
	}
	return tf, nil
}

// GetOffset returns the offset for the given path
func (tf *TestFileMap) GetOffset(path string) (Offset, bool) {
	offset, ok := tf.offsets[path]
	return offset, ok
}

// GetDataFile returns the data file
func (tf *TestFileMap) GetDataFile() (io.ReadSeeker, error) {
	return os.Open(tf.dataPath)
}

// --

// FileMapReader encapsulates a file in the file map.
type FileMapReader struct {
	limited io.Reader
	len     int64
}

// NewFileMapReader creates a new FileMapReader.
func NewFileMapReader(path string, fm FileMap) (*FileMapReader, error) {
	// Look up path in file map
	offset, ok := fm.GetOffset(path)
	if !ok {
		return nil, fmt.Errorf("file %s does not exist in file map", path)
	}

	// Created limited reader using offset
	f, err := fm.GetDataFile()
	if err != nil {
		return nil, err
	}

	_, err = f.Seek(int64(offset.Start), os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	l := int64(offset.End) - int64(offset.Start)
	if l < 0 {
		return nil, fmt.Errorf("failed to read %s, length < 0", path)
	}

	limited := io.LimitReader(f, l)
	len := int64(base64.StdEncoding.DecodedLen(int(l)))
	b64 := base64.NewDecoder(base64.StdEncoding, limited)

	return &FileMapReader{
		limited: b64,
		len:     len,
	}, nil
}

// Len returns the total number of bytes in this reader
func (r *FileMapReader) Len() int64 { // called in kite-golib/tensorflow/model.go
	return r.len
}

// Read reads the file from the file map.
func (r *FileMapReader) Read(buf []byte) (int, error) {
	return r.limited.Read(buf)
}

// Close closes the limited reader.
func (r *FileMapReader) Close() error {
	return nil
}
