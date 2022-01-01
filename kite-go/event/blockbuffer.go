package event

import (
	"bytes"
	"compress/gzip"
	"io"
)

type blockBuffer struct {
	size   int
	buf    *bytes.Buffer
	events [][]byte
}

func newBlockBuffer(size int) *blockBuffer {
	return &blockBuffer{
		size: size,
		buf:  bytes.NewBuffer(make([]byte, 0, size)),
	}
}

func (r *blockBuffer) len() int {
	return r.buf.Len()
}

func (r *blockBuffer) available() int {
	return r.size - r.buf.Len()
}

func (r *blockBuffer) writeEvent(buf []byte) error {
	s := r.buf.Len()
	_, err := r.buf.Write(buf)
	if err != nil {
		return err
	}
	e := r.buf.Len()
	r.events = append(r.events, r.buf.Bytes()[s:e])
	return nil
}

func (r *blockBuffer) count() int {
	return len(r.events)
}

func (r *blockBuffer) writeTo(w io.Writer) error {
	comp := gzip.NewWriter(w)
	for i := len(r.events) - 1; i >= 0; i-- {
		_, err := comp.Write(r.events[i])
		if err != nil {
			return err
		}
	}
	return comp.Close()
}
