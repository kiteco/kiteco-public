package skeleton

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// Type represents a skeleton python type
type Type struct {
	Path    pythonimports.DottedPath
	Attrs   map[string]string // TODO(JUAN): union types
	Methods map[string]string // TODO(JUAN): union types
	Bases   []string
}

// Attr represents an attr for a python type/module
type Attr struct {
	Path pythonimports.DottedPath
	Type string // TODO(juan): union types
}

// Param represents a skeleton python function parameter
type Param struct {
	// Types for the parameter
	Types []string

	// Name of the parameter
	Name string

	// Default type of the parameter
	// Need this to distinguish between Postional and Keyword arguments
	Default string
}

// Function represents a skeleton python function
type Function struct {
	Path    pythonimports.DottedPath
	Params  []Param
	Varargs *Param
	Kwargs  *Param
	Return  []string
}

// Module represents a skeleton python module
type Module struct {
	Path       pythonimports.DottedPath
	SubModules map[string]*Module
	Types      map[string]string
	Functions  map[string]string
	Attrs      map[string]string
}
