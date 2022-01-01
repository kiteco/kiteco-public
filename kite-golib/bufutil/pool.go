package bufutil

import (
	"sync"

	spooky "github.com/dgryski/go-spooky"
)

const (
	defaultPoolSize = 1 << 20 // 1mb
)

// Pool is a []byte pool to help avoid duplicate copies of bytes. Helpful
// for tokenization or other token processing algorithms.
type Pool struct {
	m      sync.Mutex
	buf    []byte
	off    int
	tokens map[uint64][]byte
}

// NewPool returns a Pool with the default size (1mb)
func NewPool() *Pool {
	return NewPoolSize(defaultPoolSize)
}

// NewPoolSize returns a Pool with the previded size.
func NewPoolSize(size int) *Pool {
	return &Pool{
		buf:    make([]byte, size),
		tokens: make(map[uint64][]byte),
	}
}

// Get returns a []byte pointing to the provided token. If it already exists,
// it will point to the existing bytes. If it did not exist, it will be copied
// to the internal buffer and point to that location.
func (p *Pool) Get(token []byte) []byte {
	p.m.Lock()
	defer p.m.Unlock()
	fp := spooky.Hash64(token)
	tok, exists := p.tokens[fp]
	if !exists {
		tok = p.addToken(fp, token)
	}
	return tok
}

// Exists checks whther the provided token exists in the Pool.
func (p *Pool) Exists(token []byte) bool {
	p.m.Lock()
	defer p.m.Unlock()
	fp := spooky.Hash64(token)
	_, exists := p.tokens[fp]
	return exists
}

// Available returns the available capacity in the Pool.
func (p *Pool) Available() int {
	return len(p.buf) - p.off
}

// --

func (p *Pool) addToken(fp uint64, tok []byte) []byte {
	slot := p.buf[p.off : p.off+len(tok)]
	p.off += copy(slot, tok)
	p.tokens[fp] = slot
	return slot
}
