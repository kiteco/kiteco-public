package sandbox

import (
	"bytes"
	"sync"
)

// syncBuffer is a small synchronized wrapper around a bytes.Buffer
type syncBuffer struct {
	buf   bytes.Buffer
	lines int
	mutex sync.Mutex
}

func (b *syncBuffer) Write(buf []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.lines += bytes.Count(buf, []byte("\n"))
	return b.buf.Write(buf)
}

func (b *syncBuffer) Counts() (lines, bytes int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.lines, b.buf.Len()
}

func (b *syncBuffer) Bytes() []byte {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buf.Bytes()
}
