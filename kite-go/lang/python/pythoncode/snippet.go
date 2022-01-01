package pythoncode

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"path/filepath"
	"strings"

	spooky "github.com/dgryski/go-spooky"
)

// Snippet represents a block of code
type Snippet struct {
	FromFile string `json:"From"` // path to the file containing this snippet, relative to a source root
	Code     string // The code as a string

	NumLines     int            // Number of lines
	Width        int            // Length of longest line
	Area         int            // width * numLines
	FullFunction bool           // True if snippet contains a full function def
	Terms        map[string]int // see Parser.terms()
	TermCount    int            // sum of all values in the Terms dictionary

	Incantations []*Incantation // all incantations generated from this snippet
	Decorators   []*Incantation // all decorators used on this snippet
	Attributes   []string       // all attributes referenced in this snippet
}

// From returns a short string containing the path to this file but not the filename itself
//   "fmt/fmt.go"                                    -> "fmt"
//   "go/ast/parser.go"                              -> "go/ast"
//   "github.com/dgryski/go-spooky/spookyhash.go"    -> "dgryski/go-spooky"
//   "github.com/dgryski/go-spooky/internal/bits.go" -> "dgryski/go-spooky/internal"
//   "github.com/foo/bar.go"                         -> "foo"
func (s *Snippet) From() string {
	// Remove the filename in the short "from"
	from := filepath.Dir(s.FromFile)

	// Also remove the first item in the path if it is a hostname
	if i := strings.Index(from, "/"); i != -1 && strings.Contains(from[:i], ".") {
		from = from[i+1:]
	}

	return from
}

// Hash gets a hash of the code block associated with this snippet
func (s *Snippet) Hash() SnippetHash {
	var h SnippetHash
	spooky.Hash128([]byte(s.Code), &h[0], &h[1])
	return h
}

// SnippetHash represents a 128-bit hash of the code in a snippet
type SnippetHash [2]uint64

// String returns a base64-encoded string representation of the hash
func (h SnippetHash) String() string {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, h[0])
	binary.Write(&buf, binary.LittleEndian, h[1])
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
