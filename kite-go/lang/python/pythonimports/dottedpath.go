package pythonimports

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	spooky "github.com/dgryski/go-spooky"
)

// DottedPath represents a dot-separated python path.
// It is stored in this way because many of the constituent path
// components are common to a large number of nodes, so since Go
// strings are immutable we can share the path components between
// various nodes. This reduces the sum of the sizes of all the
// strings in the production graph by as much as 80% (as of March 2016).
// While it is possible to join the path components together on
// demand, this should be avoided in favour of applying string
// manipulation algorithms directly to the slices.
// Hash is a 64 bit non-cryptographic hash of the path. It should
// only be used to determine whether two paths are equal; do not rely
// on this using any particular hash function.
// The empty path is represented by an empty list of parts and a
// hash of zero (even though the underlying hash function does not
// return zero when provided with the empty string as input).
type DottedPath struct {
	Hash  Hash
	Parts []string
}

// PathHash hashes the dotted path p
func PathHash(b []byte) Hash {
	return Hash(spooky.Hash64(b))
}

// NewDottedPath constructs a dotted path by splitting a string at periods
func NewDottedPath(s string) DottedPath {
	if s == "" {
		return DottedPath{} // because strings.Split returns a length-1 array for empty input
	}
	return DottedPath{
		Hash:  PathHash([]byte(s)),
		Parts: strings.Split(s, "."),
	}
}

var bufferPool = &sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

func discardBuffer(b *bytes.Buffer) {
	b.Reset()
	bufferPool.Put(b)
}

// NewPath constructs a dotted path from a sequence of parts
func NewPath(parts ...string) DottedPath {
	if len(parts) == 0 {
		return DottedPath{} // because strings.Split returns a length-1 array for empty input
	}

	sz := len(parts) - 1 // number of "."s
	for _, part := range parts {
		sz += len(part)
	}
	buf := bufferPool.Get().(*bytes.Buffer)

	buf.Write([]byte(parts[0]))
	for _, part := range parts[1:] {
		buf.Write([]byte("."))
		buf.Write([]byte(part))
	}

	dp := DottedPath{
		Hash:  PathHash(buf.Bytes()),
		Parts: parts,
	}
	discardBuffer(buf)
	return dp
}

// Empty returns true if the path is empty
func (p DottedPath) Empty() bool {
	return len(p.Parts) == 0
}

// Head returns the first path component, e.g. "numpy" for
// "numpy.ndarray.sum", or empty string if the path is empty.
func (p DottedPath) Head() string {
	if len(p.Parts) == 0 {
		return ""
	}
	return p.Parts[0]
}

// Last returns the last path component, e.g. "sum" for
// "numpy.ndarray.sum", or empty string if the path is empty.
func (p DottedPath) Last() string {
	if len(p.Parts) == 0 {
		return ""
	}
	return p.Parts[len(p.Parts)-1]
}

// HasPrefix returns true if the argument string is a prefix of the receiver path
func (p DottedPath) HasPrefix(s string) bool {
	if s == "" {
		return p.Empty()
	}

	if strings.Count(s, ".")+1 > len(p.Parts) {
		return false
	}

	for _, part := range p.Parts {
		if !strings.HasPrefix(s, part) {
			return false
		}
		s = s[len(part):]
		if s == "" {
			return true
		}
		if !strings.HasPrefix(s, ".") {
			return false
		}
		s = s[1:]
	}

	return true
}

// Predecessor returns the path without the tail e.g. for
// "x.y.z", this function returns "x.y". If the path has fewer
// than 2 components, this function returns the empty path.
func (p DottedPath) Predecessor() DottedPath {
	if len(p.Parts) < 2 {
		return DottedPath{}
	}
	return NewPath(p.Parts[:len(p.Parts)-1]...)
}

// Equals returns true if p.String() == s (but avoids allocs)
func (p DottedPath) Equals(s string) bool {
	for i, part := range p.Parts {
		if !strings.HasPrefix(s, part) {
			return false
		}
		s = s[len(part):]
		if i < len(p.Parts)-1 {
			if !strings.HasPrefix(s, ".") {
				return false
			}
			s = s[1:]
		}
	}
	return len(s) == 0
}

// Less returns true if either the the length of p is less than (number of components)
// than that of other, or if p and other are of equal length and p is lexicographically
// (component-wise) less than other.
func (p DottedPath) Less(other DottedPath) bool {
	return p.compare(other) < 0
}

func (p DottedPath) compare(other DottedPath) int {
	fstParts := p.Parts
	sndParts := other.Parts
	if len(fstParts) < len(sndParts) {
		return -1
	}
	if len(fstParts) > len(sndParts) {
		return 1
	}

	for len(fstParts) > 0 && len(sndParts) > 0 {
		if fstParts[0] < sndParts[0] {
			return -1
		}
		if fstParts[0] > sndParts[0] {
			return 1
		}
		fstParts = fstParts[1:]
		sndParts = sndParts[1:]
	}

	if len(fstParts) < len(sndParts) {
		return -1
	}
	if len(fstParts) == len(sndParts) {
		return 0
	}
	return 1
}

// Copy returns a deep copy of the DottedPath
func (p DottedPath) Copy() DottedPath {
	partsCopy := make([]string, len(p.Parts))
	copy(partsCopy, p.Parts)
	return NewPath(partsCopy...)
}

// Valid returns false if any path component is empty or
// contains periods
func (p DottedPath) Valid() bool {
	for _, part := range p.Parts {
		if part == "" || strings.Contains(part, ".") {
			return false
		}
	}
	return true
}

// String returns the path components joined with periods,
// e.g. "numpy.ndarray.sum"
func (p DottedPath) String() string {
	return strings.Join(p.Parts, ".")
}

// WithTail returns a copy of this path with one or more components appended
func (p DottedPath) WithTail(components ...string) DottedPath {
	parts := make([]string, len(p.Parts)+len(components))
	copy(parts, p.Parts)
	copy(parts[len(p.Parts):], components)
	return NewPath(parts...)
}

// Hash represents the hash of a dotted path
type Hash uint64

// String representation of h
func (h Hash) String() string {
	return fmt.Sprintf("%x", uint64(h))
}
