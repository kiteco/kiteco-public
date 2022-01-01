package dependent

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

// JSONWriter writes samples to an io.Writer
type JSONWriter struct {
	name string
	w    io.Writer
	ch   chan []byte

	parent *JSONWriter

	Errors  int64 // Number of errors that happened
	Written int64 // Number of records written
}

// NewJSONWriter creates a new JSONWriter
func NewJSONWriter(name string, w io.Writer) *JSONWriter {
	ch := make(chan []byte)

	j := &JSONWriter{
		name: name,
		w:    w,
		ch:   ch,
	}

	go func() {
		for buf := range ch {
			if _, err := w.Write(buf); err != nil {
				log.Printf("error writing record: %v", err)
				atomic.AddInt64(&j.Errors, 1)
			} else {
				atomic.AddInt64(&j.Written, 1)
			}
		}
	}()

	return j
}

// Name implements pipeline.Dependent
func (j *JSONWriter) Name() string {
	return j.name
}

// Clone implements pipeline.Dependent
func (j *JSONWriter) Clone() pipeline.Dependent {
	jw := JSONWriter{
		name:   j.name,
		ch:     j.ch,
		parent: j,
	}
	return &jw
}

// In implements pipeline.Dependent
func (j *JSONWriter) In(s pipeline.Sample) {
	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(s)
	if err != nil {
		log.Printf("error encoding record: %v", err)
		atomic.AddInt64(&j.parent.Errors, 1)
	}

	j.ch <- buf.Bytes()
}
