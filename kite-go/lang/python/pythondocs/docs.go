package pythondocs

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// LangEntityKind is the set of possible language entities.
type LangEntityKind int

func (k LangEntityKind) String() string {
	switch k {
	case ClassKind:
		return "class"
	case ExceptionKind:
		return "exception"
	case FunctionKind:
		return "function"
	case MethodKind:
		return "method"
	case ModuleKind:
		return "module"
	case VariableKind:
		return "variable"
	case AttributeKind:
		return "attribute"
	}
	return "unknown"
}

// The language entities identified by LangEntityKind.
const (
	UnknownKind   LangEntityKind = iota
	ClassKind                    // Class
	ExceptionKind                // Exception.
	FunctionKind                 // Typical function.
	MethodKind                   // Class method.
	ModuleKind                   // Module
	VariableKind                 // Typical variable.
	AttributeKind                // Class attribute.
)

// LangEntity groups together language entity information for python. All
// modules, functions, classes and variables can be represented with LangEntity.
type LangEntity struct {
	Kind    LangEntityKind `json:"kind"`
	Module  string         `json:"module"`
	Version string         `json:"version"`
	Ident   string         `json:"ident"`
	Full    string         `json:"full"`
	NodeID  int64          `json:"node_id,omitempty"`

	Sel           string         `json:"sel,omitempty"`
	Signature     string         `json:"signature,omitempty"`
	SignatureHTML string         `json:"signature_html,omitempty"`
	Doc           string         `json:"doc,omitempty"`
	DocHTML       string         `json:"doc_html,omitempty"`
	StructuredDoc *StructuredDoc `json:"structured_doc,omitempty"`

	// Generic ranking metric
	Score float64
}

// FullIdent returns the full Python identifier (fully qualified name)
func (e *LangEntity) FullIdent() string {
	if len(e.Full) != 0 {
		return e.Full
	}
	if len(e.Ident) == 0 {
		e.Full = e.Sel
		return e.Sel
	}
	full := e.Ident
	if len(e.Sel) > 0 {
		full = full + "." + e.Sel
	}
	e.Full = full
	return full
}

// Name returns the package-hierarchial name
func (e *LangEntity) Name() string {
	if len(e.Sel) == 0 {
		return e.Ident
	}
	return e.Sel
}

// merge fills in empty fields with fields from another LangEntity.
// It modifies the calling LangEntity and leaves the other unchanged.
func (e *LangEntity) merge(other *LangEntity) {
	if e.Sel == "" {
		e.Sel = other.Sel
	}
	if e.Signature == "" {
		e.Signature = other.Signature
	}
	if e.SignatureHTML == "" {
		e.SignatureHTML = other.SignatureHTML
	}
	if e.Doc == "" {
		e.Doc = other.Doc
	}
	if e.DocHTML == "" {
		e.DocHTML = other.DocHTML
	}
	if e.StructuredDoc == nil {
		e.StructuredDoc = other.StructuredDoc
	} else if other.StructuredDoc != nil {
		if e.StructuredDoc.DescriptionHTML == "" {
			e.StructuredDoc.DescriptionHTML = other.StructuredDoc.DescriptionHTML
		}
		if e.StructuredDoc.ReturnType == "" {
			e.StructuredDoc.ReturnType = other.StructuredDoc.ReturnType
		}
	}
}

// --

// Module contains all the language entities within a python module
type Module struct {
	Name            string
	Version         string
	Documentation   *LangEntity
	Classes         []*LangEntity
	ClassMethods    []*LangEntity
	ClassAttributes []*LangEntity
	Funcs           []*LangEntity
	Vars            []*LangEntity
	Exceptions      []*LangEntity
	Unknown         []*LangEntity
}

// VisitFn is used to perform some logic on a LangEntity object
type VisitFn func(entity *LangEntity)

func (m *Module) visit(visitor VisitFn) {
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	for _, cat := range categories {
		for _, e := range cat {
			visitor(e)
		}
	}
}

// Entities returns the total number of entities in the module
func (m *Module) Entities() int {
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	var total int
	for _, cat := range categories {
		total += len(cat)
	}
	return total
}

// EncodeGob encodes python Module struct to a dense json representation
func (m *Module) EncodeGob(enc *gob.Encoder) error {
	d := LangEntity{
		Module:  m.Name,
		Version: m.Version,
		Ident:   m.Name,
		Kind:    ModuleKind,
	}
	err := enc.Encode(&d)
	if err != nil {
		return err
	}
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	for _, category := range categories {
		for _, v := range category {
			err := enc.Encode(v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Encode encodes python Module struct to a dense json representation
func (m *Module) Encode(enc *json.Encoder) error {
	d := LangEntity{
		Module: m.Name,
		Ident:  m.Name,
		Kind:   ModuleKind,
	}
	err := enc.Encode(&d)
	if err != nil {
		return err
	}
	categories := [][]*LangEntity{
		m.Classes,
		m.ClassMethods,
		m.ClassAttributes,
		m.Funcs,
		m.Vars,
		m.Exceptions,
		m.Unknown,
		[]*LangEntity{
			m.Documentation,
		},
	}
	for _, category := range categories {
		for _, v := range category {
			err := enc.Encode(v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// --

// Modules is a map of module name to Module object.
type Modules map[string]*Module

// NewModules loads a Modules structure from provided path
func NewModules(path string) (Modules, error) {
	modules := make(Modules)
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decomp, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(path, "gob.gz") {
		err = modules.DecodeGob(gob.NewDecoder(decomp))
	} else if strings.HasSuffix(path, "json.gz") {
		err = modules.Decode(json.NewDecoder(decomp))
	}
	if err != nil {
		return nil, err
	}
	return modules, nil
}

// Visit calls the supplied VisitFn on each module in Modules.
func (m Modules) Visit(visitor VisitFn) {
	for _, module := range m {
		module.visit(visitor)
	}
}

// FindIdent returns the matching LangEntity object given a full identifier
func (m Modules) FindIdent(ident string) (*LangEntity, bool) {
	if len(ident) == 0 {
		return nil, false
	}

	parts := strings.Split(ident, ".")
	x := strings.Join(parts[:len(parts)-1], ".")
	sel := parts[len(parts)-1]

	entities := m.Find(x, sel)
	for _, e := range entities {
		if sel == e.Sel {
			return e, true
		}
	}
	return nil, false
}

// FindModuleSelector returns matching LangEntity objects given a module and a selector name.
func (m Modules) FindModuleSelector(mod, sel string) []*LangEntity {
	if len(mod) == 0 {
		return nil
	}

	module, exists := m[mod]
	if !exists {
		return nil
	}

	var ret []*LangEntity
	categories := [][]*LangEntity{
		module.Classes,
		module.ClassMethods,
		module.ClassAttributes,
		module.Funcs,
		module.Vars,
		module.Exceptions,
		[]*LangEntity{
			module.Documentation,
		},
	}
	for _, cat := range categories {
		for _, l := range cat {
			if l.Sel == sel {
				ret = append(ret, l)
			}
		}
	}

	return ret
}

// Find returns matching LangEntity objects given a selector.
func (m Modules) Find(x, sel string) []*LangEntity {
	if len(x) == 0 {
		return nil
	}

	parts := strings.Split(x, ".")
	module, exists := m[parts[0]]
	if !exists {
		return nil
	}

	var ret []*LangEntity
	categories := [][]*LangEntity{
		module.Classes,
		module.ClassMethods,
		module.ClassAttributes,
		module.Funcs,
		module.Vars,
		module.Exceptions,
		module.Unknown,
		[]*LangEntity{
			module.Documentation,
		},
	}
	for _, cat := range categories {
		for _, l := range cat {
			if l.Ident == x && strings.HasPrefix(l.Sel, sel) {
				ret = append(ret, l)
			}
		}
	}

	return ret
}

// FindPrefix returns matching LangEntity objects given a prefix
func (m Modules) FindPrefix(x string) []*LangEntity {
	if len(x) == 0 {
		return nil
	}

	parts := strings.Split(x, ".")
	module, exists := m[parts[0]]
	if !exists {
		return nil
	}

	var ret []*LangEntity
	categories := [][]*LangEntity{
		module.Classes,
		module.ClassMethods,
		module.ClassAttributes,
		module.Funcs,
		module.Vars,
		module.Exceptions,
		module.Unknown,
		[]*LangEntity{
			module.Documentation,
		},
	}
	for _, cat := range categories {
		for _, l := range cat {
			if l.Ident == x {
				ret = append(ret, l)
			}
		}
	}

	return ret
}

// DecodeGob reads in a series of python.Module objects from the provided gob decoder.
func (m Modules) DecodeGob(dec *gob.Decoder) error {
	var count int
	var module *Module
	for {
		var entity LangEntity
		err := dec.Decode(&entity)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch entity.Kind {
		case ModuleKind:
			module = m.ensureModuleExists(entity.FullIdent())
			if module.Documentation.NodeID != 0 {
				// There's already documentation set for this module name,
				// so avoid throwing away the new LangEntity by setting it on unknown.
				// This is obviously a hack, but doesn't matter because the
				// `Modules` flow is all legacy, and should be removed.
				module.Unknown = append(module.Unknown, &entity)
				// The only modules for which this matters as of 2017.10.19 are
				// unittest & unittest2, due to there being two nodes for each
				// module.
			} else {
				module.Documentation = &entity
				module.Version = entity.Version
			}
		case ClassKind:
			module = m.ensureModuleExists(entity.Module)
			module.Classes = append(module.Classes, &entity)
		case MethodKind:
			module = m.ensureModuleExists(entity.Module)
			module.ClassMethods = append(module.ClassMethods, &entity)
		case AttributeKind:
			module = m.ensureModuleExists(entity.Module)
			module.ClassAttributes = append(module.ClassAttributes, &entity)
		case FunctionKind:
			module = m.ensureModuleExists(entity.Module)
			module.Funcs = append(module.Funcs, &entity)
		case VariableKind:
			module = m.ensureModuleExists(entity.Module)
			module.Vars = append(module.Vars, &entity)
		case ExceptionKind:
			module = m.ensureModuleExists(entity.Module)
			module.Exceptions = append(module.Exceptions, &entity)
		case UnknownKind:
			module = m.ensureModuleExists(entity.Module)
			module.Unknown = append(module.Unknown, &entity)
		}
		count++
	}

	log.Println("loaded", len(m), "modules")
	return nil
}

// Decode reads in a series of python.Module objects from the provided json decoder.
func (m Modules) Decode(dec *json.Decoder) error {
	var count int
	var module *Module
	for {
		var entity LangEntity
		err := dec.Decode(&entity)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch entity.Kind {
		case ModuleKind:
			_ = m.ensureModuleExists(entity.Ident)
		case ClassKind:
			module = m.ensureModuleExists(entity.Module)
			module.Classes = append(module.Classes, &entity)
		case MethodKind:
			module = m.ensureModuleExists(entity.Module)
			module.ClassMethods = append(module.ClassMethods, &entity)
		case AttributeKind:
			module = m.ensureModuleExists(entity.Module)
			module.ClassAttributes = append(module.ClassAttributes, &entity)
		case FunctionKind:
			module = m.ensureModuleExists(entity.Module)
			module.Funcs = append(module.Funcs, &entity)
		case VariableKind:
			module = m.ensureModuleExists(entity.Module)
			module.Vars = append(module.Vars, &entity)
		case ExceptionKind:
			module = m.ensureModuleExists(entity.Module)
			module.Exceptions = append(module.Exceptions, &entity)
		case UnknownKind:
			module = m.ensureModuleExists(entity.Module)
			module.Unknown = append(module.Unknown, &entity)
		}
		count++
	}

	log.Println("loaded", len(m), "modules")
	return nil
}

// ensureModuleExists returns module with the given name. Creates module if necessary.
func (m Modules) ensureModuleExists(name string) *Module {
	module, exists := m[name]
	if !exists {
		module = &Module{
			Name: name,
			Documentation: &LangEntity{
				Module: name,
				Ident:  name,
				Kind:   ModuleKind,
			},
		}
		m[name] = module
	}
	return module
}
