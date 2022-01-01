package stringutil

import (
	"sync"

	spooky "github.com/dgryski/go-spooky"
)

var (
	m    sync.RWMutex
	data = make(map[uint64]string)
)

// Clear will reset interned strings and bytes
func Clear() {
	m.Lock()
	defer m.Unlock()
	data = make(map[uint64]string)
}

// Intern returns an interned copy of the provided string
func Intern(s string) string {
	m.Lock()
	defer m.Unlock()
	h := Hash64(s)
	if out, ok := data[h]; ok {
		return out
	}
	data[h] = s
	return s
}

// InternBytes interns a byte slice into a string
func InternBytes(b []byte) string {
	m.Lock()
	defer m.Unlock()
	h := spooky.Hash64(b)
	if out, ok := data[h]; ok {
		return out
	}
	s := string(b)
	data[h] = s
	return s
}

// ToUint64 stores & returns a 64-bit hash of the provided string
func ToUint64(s string) uint64 {
	m.Lock()
	defer m.Unlock()
	h := Hash64(s)
	if _, ok := data[h]; ok {
		return h
	}
	data[h] = s
	return h
}

// ToUint64Bytes stores & returns a 64-bit hash of the provided byte slice
func ToUint64Bytes(b []byte) uint64 {
	m.Lock()
	defer m.Unlock()
	h := spooky.Hash64(b)
	if _, ok := data[h]; ok {
		return h
	}
	data[h] = string(b)
	return h
}

// FromUint64 returns a string from a 64-bit hash
func FromUint64(h uint64) string {
	m.Lock()
	defer m.Unlock()
	if s, ok := data[h]; ok {
		return s
	}
	panic("called FromUint32 without using ToUint32")
}

// Hash64 computes a uint64 hash of the provided string
func Hash64(s string) uint64 {
	return spooky.Hash64([]byte(s))
}
