package token

import (
	"net/http"
	"sync"
)

const (
	tokenHeaderKey     = "Kite-Token"
	tokenDataHeaderKey = "Kite-TokenData"
)

// Token wraps hmac header value setting and extraction
type Token struct {
	rw      sync.RWMutex
	headers http.Header
}

// NewToken creates a new TokenAuth object
func NewToken() *Token {
	return &Token{
		headers: make(http.Header),
	}
}

// Clear clears tokens
func (t *Token) Clear() {
	t.rw.Lock()
	defer t.rw.Unlock()

	t.headers = make(http.Header)
}

// AddToHeader takes an http.Header and augments it with HMAC tokens
func (t *Token) AddToHeader(h http.Header) {
	t.rw.RLock()
	defer t.rw.RUnlock()

	token := t.headers.Get(tokenHeaderKey)
	tokenData := t.headers.Get(tokenDataHeaderKey)
	if token != "" && tokenData != "" {
		h.Set(tokenHeaderKey, token)
		h.Set(tokenDataHeaderKey, tokenData)
	}
}

// UpdateFromHeader takes an http.Header and extracts HMAC token info that may be present
func (t *Token) UpdateFromHeader(h http.Header) {
	token := h.Get(tokenHeaderKey)
	tokenData := h.Get(tokenDataHeaderKey)

	if token != "" && tokenData != "" {
		t.rw.Lock()
		defer t.rw.Unlock()

		t.headers.Set(tokenHeaderKey, token)
		t.headers.Set(tokenDataHeaderKey, tokenData)
	}
}
