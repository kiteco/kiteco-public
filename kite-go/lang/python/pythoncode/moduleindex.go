package pythoncode

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/diskmapindex"
)

// hash gets a hash of the input code block.
func hash(src []byte) codeHash {
	var h codeHash
	spooky.Hash128(src, &h[0], &h[1])
	return h
}

// codeHash represents a 128-bit hash of a piece of code.
type codeHash [2]uint64

// String returns a base64-encoded string representation of the hash.
func (h codeHash) String() string {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, h[0])
	binary.Write(&buf, binary.LittleEndian, h[1])
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// KeywordCounts describes a map from the name of keyword argument to its frequency for certain function call
type KeywordCounts map[string]int32

// KeywordCountsByFunc describes a map from the name of a function call to its KeywordCounts
type KeywordCountsByFunc map[string]KeywordCounts

// SymbolContext in which a symbol appeared
type SymbolContext string

const (
	symbolContextImportAliasPrefix = "symbol_context_import_alias:"
	// SymbolContextImport denotes a symbol appearing in an import context
	SymbolContextImport SymbolContext = "symbol_context_import"
	// SymbolContextAttribute denotes a symbol appearing in an attribute context
	SymbolContextAttribute SymbolContext = "symbol_context_attribute"
	// SymbolContextName denotes a symbol appearing in a name context
	SymbolContextName SymbolContext = "symbol_context_name"
	// SymbolContextExpr denotes a symbol appearing in an expression context
	SymbolContextExpr SymbolContext = "symbol_context_expr"
	// SymbolContextCallFunc denotes a symbol appearing in the Func portion of a call expression
	SymbolContextCallFunc SymbolContext = "symbol_context_call_func"
	// SymbolContextAll denotes a symbol appearing in any context
	SymbolContextAll SymbolContext = "symbol_context_all"
)

// SymbolContextImportAlias denotes a symbol appearing in an import context with an alias
func SymbolContextImportAlias(alias string) SymbolContext {
	return SymbolContext(fmt.Sprintf("%s%s", symbolContextImportAliasPrefix, alias))
}

// ImportAlias returns a bool indicating whether this context is a SymbolContextImportAlias, and the alias string
func (sc SymbolContext) ImportAlias() (string, bool) {
	if !strings.HasPrefix(string(sc), symbolContextImportAliasPrefix) {
		return "", false
	}
	return string(sc)[len(symbolContextImportAliasPrefix):], true
}

// ConstInfo represents a frequency count for constants
type ConstInfo map[string]int32

// TypedConstInfo contains both int and string const infos
type TypedConstInfo struct {
	IntConstInfo    ConstInfo `json:"int_const"`
	StringConstInfo ConstInfo `json:"str_const"`
}

// ArgConstInfo describes a map from a argument name to its const info
type ArgConstInfo map[string]TypedConstInfo

// ArgConstInfoByFunc describes a map from a function call to its keyword parameters' const info
type ArgConstInfoByFunc map[string]ArgConstInfo

// Valid symbol context
func (sc SymbolContext) Valid() bool {
	switch sc {
	case SymbolContextImport, SymbolContextAttribute, SymbolContextName,
		SymbolContextExpr, SymbolContextCallFunc, SymbolContextAll:
		return true
	default:
		return strings.HasPrefix(string(sc), symbolContextImportAliasPrefix)
	}
}

// SymFileCounts keeps counts of how many times a symbol is used in a Python file, broken down by context.
type SymFileCounts struct {
	ImportAliases map[string]int32

	Import    int32
	Attribute int32
	Name      int32
	Expr      int32
	CallFunc  int32
}

// CountFor the specified symbol context
func (s SymFileCounts) CountFor(sc SymbolContext) int32 {
	switch sc {
	case SymbolContextImport:
		return s.Import
	case SymbolContextAttribute:
		return s.Attribute
	case SymbolContextName:
		return s.Name
	case SymbolContextExpr:
		return s.Expr
	case SymbolContextCallFunc:
		return s.CallFunc
	case SymbolContextAll:
		return s.Import + s.Attribute + s.Name + s.Expr + s.CallFunc
	default:
		return 0
	}
}

// HashCounts keeps track of a Python file hash along with the relevant counts for a given symbol.
type HashCounts struct {
	Hash   string
	Counts SymFileCounts
}

// CodeHash for the provided source code.
func CodeHash(src []byte) string {
	return hash(src).String()
}

// EncodeHashes into a slice of bytes for serialization
func EncodeHashes(counts []HashCounts) ([]byte, error) {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(counts); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// DecodeHashes from a slice of bytes
func DecodeHashes(buf []byte) ([]HashCounts, error) {
	var counts []HashCounts
	if err := gob.NewDecoder(bytes.NewBuffer(buf)).Decode(&counts); err != nil {
		return nil, err
	}
	return counts, nil
}

// SymbolToHashesIndex maps from a symbol to the hashes of files that contain references
// to the symbol.
type SymbolToHashesIndex struct {
	dmi *diskmapindex.Index
}

// NewSymbolToHashesIndex from the specified path or error.
func NewSymbolToHashesIndex(path, cache string) (*SymbolToHashesIndex, error) {
	dmi, err := diskmapindex.NewIndex(path, cache)
	if err != nil {
		return nil, err
	}

	return &SymbolToHashesIndex{
		dmi: dmi,
	}, nil
}

// HashesFor the specified symbol.
func (s *SymbolToHashesIndex) HashesFor(symbol pythonresource.Symbol) ([]HashCounts, error) {
	bufs, err := s.dmi.Get(symbol.PathString())
	if err != nil {
		return nil, err
	}

	var hcs []HashCounts
	for _, buf := range bufs {
		hc, err := DecodeHashes(buf)
		if err != nil {
			return nil, fmt.Errorf("error decoding hashes: %v", err)
		}

		if len(hcs) == 0 {
			hcs = make([]HashCounts, 0, len(hc)*len(bufs))
		}

		hcs = append(hcs, hc...)
	}

	return hcs, nil
}

// IterateSlowly emits pairs of symbol path string + HashCounts.
// The same path string may be emitted multiple times.
func (s *SymbolToHashesIndex) IterateSlowly(emit func(string, []HashCounts) error) error {
	return s.dmi.IterateSlowly(func(key string, buf []byte) error {
		hc, err := DecodeHashes(buf)
		if err != nil {
			return err
		}
		return emit(key, hc)
	})
}

// HashToSourceIndex maps from hash of source code contents to the source code contents.
type HashToSourceIndex struct {
	dmi *diskmapindex.Index
}

// NewHashToSourceIndex loads an index from the provided path.
func NewHashToSourceIndex(path string, cachedir string) (*HashToSourceIndex, error) {
	dmi, err := diskmapindex.NewIndex(path, cachedir)
	if err != nil {
		return nil, err
	}
	return &HashToSourceIndex{
		dmi: dmi,
	}, nil
}

// SourceFor the provided hash or an error if the source code cannot be found.
func (i *HashToSourceIndex) SourceFor(hash string) ([]byte, error) {
	srcs, err := i.dmi.Get(hash)
	if err != nil {
		return nil, err
	}
	return srcs[0], nil
}
