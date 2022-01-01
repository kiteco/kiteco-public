package serialization

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Encode writes the object to the path, using the format specified by the file
// extension, which can be .json, .gob, .xml., .yml, or .yaml. The path may
// additionally have a .gz suffix, in which case the stream will be compressed.
func Encode(path string, obj interface{}) error {
	enc, err := NewEncoder(path)
	if err != nil {
		return err
	}
	defer enc.Close()
	return enc.Encode(obj)
}

// Encoder is an interface that matches gob.Encoder, json.Encoder, and xml.Encoder
type Encoder interface {
	// Encoder adds an item to the stream
	Encode(interface{}) error
}

// EncodeCloser is an encoder that can also close its underlying stream
type EncodeCloser struct {
	encoder Encoder
	closers []io.Closer
}

// Encode writes an object to the underlying stream
func (e *EncodeCloser) Encode(x interface{}) error {
	return e.encoder.Encode(x)
}

// Close closes the underlying stream
func (e *EncodeCloser) Close() error {
	var closeErr error
	// We must close in reverse order
	for i := len(e.closers) - 1; i >= 0; i-- {
		if err := e.closers[i].Close(); err != nil {
			closeErr = err
		}
	}
	return closeErr
}

// NewEncoder opens the specified path and returns an ecoder that writes in the format
// specified by the file extension, which can be .json, .gob, .xml., .yml, or .yaml. The
// path may additionally have a .gz suffix, in which case the stream will be comrpessed.
func NewEncoder(path string) (*EncodeCloser, error) {
	var w io.WriteCloser
	w, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	closers := []io.Closer{w}

	// Switch on compression
	switch {
	case strings.HasSuffix(path, ".gz"):
		path = strings.TrimSuffix(path, ".gz")
		w = gzip.NewWriter(w)
		closers = append(closers, w)
	}

	// Switch on encoding
	var e Encoder
	switch {
	case strings.HasSuffix(path, ".json"):
		e = json.NewEncoder(w)
	case strings.HasSuffix(path, ".gob"):
		e = gob.NewEncoder(w)
	case strings.HasSuffix(path, ".xml"):
		e = xml.NewEncoder(w)
	default:
		return nil, fmt.Errorf("could not find encoder for %s", path)
	}

	return &EncodeCloser{
		encoder: e,
		closers: closers,
	}, nil
}
