package keytypes

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"

// Kind represents the classification for a given Symbol
type Kind int

const (
	// NoneKind represents the zero value for Kind
	NoneKind Kind = iota
	// FunctionKind is the classification for nodes representing functions
	FunctionKind
	// TypeKind is the classification for nodes representing types
	TypeKind
	// ModuleKind is the classification for nodes representing modules
	ModuleKind
	// DescriptorKind is the classification for nodes representing descriptors
	DescriptorKind
	// ObjectKind is the classification for nodes that do not fall into any other category
	ObjectKind
)

// TypeInfo bundles the kind, type, and base classes for a Symbol
type TypeInfo struct {
	Type  pythonimports.DottedPath
	Bases []pythonimports.DottedPath
	Kind  Kind
}
