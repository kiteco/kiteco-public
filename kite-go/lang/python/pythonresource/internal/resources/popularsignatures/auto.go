//go:generate msgp -marshal=false

package popularsignatures

import (
	"unsafe"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
)

// Language is a structural analog of lang.Language
type Language int

//msgp:tuple ID

// ID is a structural analog of editorapi.ID
type ID struct {
	ID   string   `msg:"id"`
	Lang Language `msg:"lang"`
}

//msgp:tuple ParameterTypeExample

// ParameterTypeExample is a structural analog of editorapi.ParameterTypeExample
type ParameterTypeExample struct {
	// ID for the value of the type.
	ID ID `msg:"ID"`

	// Name is a human readable name for the type (value).
	Name string `msg:"name"`

	// Examples are plain string examples from codebases on Github.
	Examples  []string `msg:"examples"`
	Frequency float64  `msg:"frequency"`
}

//msgp:tuple ParameterExample

// ParameterExample is a structural analog of editorapi.ParameterExample
type ParameterExample struct {
	// Name of the argument used in a function call.
	Name string `msg:"name"`

	// Types of the argument used in a function call.
	Types []*ParameterTypeExample `msg:"types"`
}

//msgp:tuple PythonSignatureDetails

// PythonSignatureDetails is a structural analog of editorapi.PythonSignatureDetails
type PythonSignatureDetails struct {
	// Kwargs passed into a python function
	Kwargs []*ParameterExample `msg:"kwargs"`
}

//msgp:tuple LanguageSignatureDetails

// LanguageSignatureDetails is a structural analog of editorapi.LanguageSignatureDetails
type LanguageSignatureDetails struct {
	Python *PythonSignatureDetails `msg:"python"`
}

//msgp:tuple Signature

// Signature is a structural analog of editorapi.Signature
type Signature struct {
	Args            []*ParameterExample      `msg:"args"`
	LanguageDetails LanguageSignatureDetails `msg:"language_details"`
	Frequency       float64                  `msg:"frequency"`
}

// Entity represents signature patterns for a given Symbol
type Entity []*Signature

// Cast casts to []*editorapi.Signature
func (e Entity) Cast() []*editorapi.Signature {
	return *(*[]*editorapi.Signature)(unsafe.Pointer(&e))
}

// CastEntity casts []*editorapi.Signature to Entity
func CastEntity(e []*editorapi.Signature) Entity {
	return *(*Entity)(unsafe.Pointer(&e))
}
