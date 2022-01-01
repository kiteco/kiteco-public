package skeleton

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kr/pretty"
)

// RawNode represents a generic node parsed from yaml
type RawNode map[string]interface{}

// Builder builds nodes from raw yaml files
type Builder struct {
	Types     map[pythonimports.Hash]*Type
	Functions map[pythonimports.Hash]*Function
	Modules   map[pythonimports.Hash]*Module
	Attrs     map[pythonimports.Hash]*Attr
}

// NewBuilder returns a new Builder
func NewBuilder() Builder {
	return Builder{
		Types:     make(map[pythonimports.Hash]*Type),
		Functions: make(map[pythonimports.Hash]*Function),
		Modules:   make(map[pythonimports.Hash]*Module),
		Attrs:     make(map[pythonimports.Hash]*Attr),
	}
}

// Build nodes from raw yaml representation
func (b Builder) Build(raw []RawNode) (err error) {
	defer func() {
		if ex := recover(); ex != nil {
			err = fmt.Errorf("error building nodes: %v", ex)
		}
	}()
	for _, rns := range raw {
		for t, rn := range rns {
			n := mustRawNode(rn, "error building top level raw node:")
			switch t {
			case "module":
				m := b.buildModule("", n)
				b.Modules[m.Path.Hash] = m
			case "function":
				fn := b.buildFunction("", n)
				b.Functions[fn.Path.Hash] = fn
			case "type":
				ty := b.buildType("", n)
				b.Types[ty.Path.Hash] = ty
			case "attr":
				a := b.buildAttr(n)
				b.Attrs[a.Path.Hash] = a
			default:
				return fmt.Errorf("unknown key `%s` associated with raw node \n%s", t, pretty.Sprintf("%# v", n))
			}
		}
	}
	return nil
}

// TODO(juan): these are treated a bit different than types and functions right now...
func (b Builder) buildAttr(rawattr RawNode) *Attr {
	rawpath, found := rawattr["path"]
	if !found {
		panic(fmt.Errorf("no path specified for attr %v", rawattr))
	}
	path := mustString(rawpath, "error building attr:")

	rawtype, found := rawattr["type"]
	if !found {
		panic(fmt.Errorf("error building attr %s: no type specified", path))
	}

	t := parseTypes(mustString(rawtype, "error building attr %s: error building type:", path))
	if len(t) != 1 {
		panic(fmt.Errorf("error building attr %s: must specify exactly one type (specified %d)", path, len(t)))
	}

	return &Attr{
		Path: pythonimports.NewDottedPath(path),
		Type: t[0],
	}
}

func (b Builder) buildModule(prefix string, rawmodule RawNode) *Module {
	// verify path
	path := path(rawmodule)
	switch {
	case path == "" && prefix == "":
		panic(fmt.Errorf("error building module, no path specified: %v", rawmodule))
	case path == "":
		path = prefix
	}

	m := &Module{
		Path:       pythonimports.NewDottedPath(path),
		SubModules: make(map[string]*Module),
		Types:      make(map[string]string),
		Functions:  make(map[string]string),
		Attrs:      make(map[string]string),
	}
	for t, rawnode := range rawmodule {
		switch t {
		case "path": // already handled above
		case "submodules":
			rawsubmodules := mustRawNode(rawnode, "error building submodules for %s:", path)
			for name, rawsubmod := range rawsubmodules {
				submodule := b.buildModule(path+"."+name, mustRawNode(rawsubmod, "error building %s.%s", path, name))

				// add sub module to global table
				b.Modules[submodule.Path.Hash] = submodule

				// add sub module to module table
				m.SubModules[name] = submodule
			}
		case "functions":
			rawfuncs := mustRawNode(rawnode, "error building functions for %s:", path)
			for name, rawfunc := range rawfuncs {
				switch rf := rawfunc.(type) {
				case string:
					// function specified via a path
					m.Functions[name] = rf
				default:
					fn := b.buildFunction(path+"."+name, mustRawNode(rawfunc, "error building %s.%s:", path, name))

					// add function to global table
					b.Functions[fn.Path.Hash] = fn

					// add function to module table
					m.Functions[name] = fn.Path.String()
				}
			}
		case "attrs":
			rattrs := mustRawNode(rawnode, "error building attrs for %s:", path)
			for a, rty := range rattrs {
				types := parseTypes(mustString(rty, "error building attr %s.%s:", path, a))
				if len(types) < 1 {
					panic(fmt.Errorf("error building attr %s.%s, need to specify at least one type, got %v (%d)", path, a, rty, len(types)))
				}
				m.Attrs[a] = types[0]
			}
		case "types":
			rawtypes := mustRawNode(rawnode, "error building types for %s:", path)
			for name, rawtype := range rawtypes {
				switch rt := rawtype.(type) {
				case string:
					// type specified via a path
					types := parseTypes(rt)
					if len(types) != 1 {
						panic(fmt.Errorf("error building type %s.%s, exactly one type allowed, got %v (%d)", path, name, rt, len(types)))
					}
					m.Types[name] = types[0]
				default:
					ty := b.buildType(path+"."+name, mustRawNode(rawtype, "error building %s.%s:", path, name))

					// add type to global table
					b.Types[ty.Path.Hash] = ty

					// add type to module table
					m.Types[name] = ty.Path.String()
				}
			}
		}
	}

	return m
}

func (b Builder) buildType(prefix string, rawtype RawNode) *Type {
	// verify path
	path := path(rawtype)
	switch {
	case path == "" && prefix == "":
		panic(fmt.Errorf("error building type, no path specified: %v", rawtype))
	case path == "":
		path = prefix
	}

	// build type
	typ := &Type{
		Path:    pythonimports.NewDottedPath(path),
		Methods: make(map[string]string),
		Attrs:   make(map[string]string),
	}

	for t, rawnode := range rawtype {
		switch t {
		case "path": // already handled above
		case "attrs":
			rattrs := mustRawNode(rawnode, "error building attrs for %s:", path)
			for a, rty := range rattrs {
				types := parseTypes(mustString(rty, "error building attr %s.%s:", path, a))
				if len(types) < 1 {
					panic(fmt.Errorf("error building attr %s.%s, need atleast one type, got %v (%d)", path, a, rty, len(types)))
				}

				// if type is self then replace with class instance
				ty := types[0]
				if ty == "self" {
					ty = path
				}

				typ.Attrs[a] = ty
			}
		case "bases":
			typ.Bases = parseTypes(mustString(rawnode, "error building bases for %s:", path))
		case "methods":
			rawfuncs := mustRawNode(rawnode, "error building functions for %s:", path)
			for name, rawfunc := range rawfuncs {
				switch rf := rawfunc.(type) {
				case string:
					// function specified via a path
					typ.Methods[name] = rf
				default:
					fn := b.buildFunction(path+"."+name, mustRawNode(rawfunc, "error building %s.%s:", path, name))

					// make sure first parameter is self or cls and make sure properly set
					if len(fn.Params) > 0 {
						switch fn.Params[0].Name {
						case "self":
							// self is assumed to be instance of type
							fn.Params[0].Types = []string{path}
						case "cls":
							// cls is assumed to be the type
							fn.Params[0].Types = []string{path + ".type"}
						default:
							p := Param{
								Name:  "self",
								Types: []string{path},
							}
							fn.Params = append([]Param{p}, fn.Params...)
						}
					} else {
						fn.Params = []Param{Param{Name: "self", Types: []string{path}}}
					}

					// if return types contain self, then replace with the current type
					for i, rt := range fn.Return {
						if rt == "self" {
							fn.Return[i] = path
							break
						}
					}

					// add function to global table
					b.Functions[fn.Path.Hash] = fn

					// add function to type table
					typ.Methods[name] = fn.Path.String()
				}
			}
		}
	}

	return typ
}

func (b Builder) buildFunction(path string, rawfunc RawNode) *Function {
	var fn Function
	for t, rawnode := range rawfunc {
		switch t {
		case "path":
			fn.Path = pythonimports.NewDottedPath(mustString(rawnode, "error building function path %v:", rawnode))
		case "params":
			rawparams, ok := rawnode.([]interface{})
			if !ok {
				panic(fmt.Errorf("error building function, expected params to be of type []interface{}, got %T", rawnode))
			}

			var paramnodes []RawNode
			for _, rp := range rawparams {
				paramnodes = append(paramnodes, mustRawNode(rp, "error building function params:"))
			}

			fn.Params = b.buildParams(paramnodes...)
		case "return":
			types := parseTypes(mustString(rawnode, "error building function return: "))
			if len(types) == 0 {
				panic("error building function, must have atleast one return type if return is specified")
			}
			fn.Return = types
		case "varargs":
			param := b.buildParams(mustRawNode(rawnode, "error building function starargs:"))
			if len(param) != 1 {
				panic(fmt.Errorf("error building function, expected exactly one starargs, got %d", len(param)))
			}
			fn.Varargs = &param[0]
		case "kwargs":
			param := b.buildParams(mustRawNode(rawnode, "error building function kwargs:"))
			if len(param) != 1 {
				panic(fmt.Errorf("error building function, expected exactly one kwarg, got %d", len(param)))
			}
			fn.Kwargs = &param[0]
		default:
			panic("error building function, unexpected key " + t)
		}
	}

	// make sure function path is properly set
	switch {
	case fn.Path.Empty() && path == "":
		panic(fmt.Errorf("error building function, no path specified: %v", rawfunc))
	case fn.Path.Empty():
		fn.Path = pythonimports.NewDottedPath(path)
	}

	return &fn
}

func (b Builder) buildParams(rawparams ...RawNode) []Param {
	var params []Param
	for _, rawparam := range rawparams {
		var p Param
		for t, v := range rawparam {
			vv := mustString(v, "error building param field %s:", t)
			switch t {
			case "name":
				p.Name = vv
			case "default":
				types := parseTypes(vv)
				if len(types) != 1 {
					panic(fmt.Errorf("error building param, default must have exactly 1 type, got %d", len(types)))
				}
				p.Default = types[0]
			case "types":
				types := parseTypes(vv)
				if len(types) == 0 {
					panic("error building param, types must have atleast 1 type")
				}
				p.Types = types
			}
		}

		// make sure types always includes default if it was set
		if p.Default != "" {
			var found bool
			for _, ty := range p.Types {
				if ty == p.Default {
					found = true
					break
				}
			}
			if !found {
				p.Types = append(p.Types, p.Default)
			}
		}
		params = append(params, p)
	}

	return params
}

func parseTypes(ts string) []string {
	// remove structured componnents for now
	for strings.Contains(ts, "<") || strings.Contains(ts, ">") {
		s, e := -1, -1
		var level int
		for i, c := range ts {
			if c == '<' {
				level++
				if s < 0 {
					s = i
				}
			}
			if c == '>' {
				level--
				e = i
			}
			if level == 0 && s > -1 && e > -1 {
				ts = ts[:s] + ts[e+1:]
				break
			}
		}
	}

	// make sure types are unique
	seen := make(map[string]struct{})
	for _, t := range strings.Split(ts, "|") {
		t = strings.TrimSpace(t)
		if len(t) > 0 {
			// remove .type suffix for now
			if pos := strings.Index(t, ".type"); pos > -1 {
				t = t[:pos]
			}
			if t == "None" {
				t = "builtins.None.__class__"
			}
			if isBuiltin(t) {
				t = "builtins." + t
			}
			seen[t] = struct{}{}
		}
	}

	var types []string
	for t := range seen {
		types = append(types, t)
	}
	return types
}

func isBuiltin(ty string) bool {
	switch ty {
	case "ArithmeticError",
		"AssertionError",
		"AttributeError",
		"BaseException",
		"BufferError",
		"BytesWarning",
		"DeprecationWarning",
		"EOFError",
		"Ellipsis",
		"EnvironmentError",
		"Exception",
		"False",
		"FloatingPointError",
		"FutureWarning",
		"GeneratorExit",
		"IOError",
		"ImportError",
		"ImportWarning",
		"IndentationError",
		"IndexError",
		"KeyError",
		"KeyboardInterrupt",
		"LookupError",
		"MemoryError",
		"NameError",
		"None",
		"NotImplemented",
		"NotImplementedError",
		"OSError",
		"OverflowError",
		"PendingDeprecationWarning",
		"ReferenceError",
		"RuntimeError",
		"RuntimeWarning",
		"StopIteration",
		"SyntaxError",
		"SyntaxWarning",
		"SystemError",
		"SystemExit",
		"TabError",
		"True",
		"TypeError",
		"UnboundLocalError",
		"UnicodeDecodeError",
		"UnicodeEncodeError",
		"UnicodeError",
		"UnicodeTranslateError",
		"UnicodeWarning",
		"UserWarning",
		"ValueError",
		"Warning",
		"ZeroDivisionError",
		"_",
		"__debug__",
		"__doc__",
		"__import__",
		"__name__",
		"__package__",
		"abs",
		"all",
		"any",
		"basestring",
		"bin",
		"bool",
		"bytearray",
		"bytes",
		"callable",
		"chr",
		"classmethod",
		"compile",
		"complex",
		"copyright",
		"credits",
		"delattr",
		"dict",
		"dir",
		"divmod",
		"enumerate",
		"eval",
		"exit",
		"file",
		"filter",
		"float",
		"format",
		"frozenset",
		"getattr",
		"globals",
		"hasattr",
		"hash",
		"help",
		"hex",
		"id",
		"input",
		"int",
		"isinstance",
		"issubclass",
		"iter",
		"len",
		"license",
		"list",
		"locals",
		"map",
		"max",
		"memoryview",
		"min",
		"next",
		"object",
		"oct",
		"open",
		"ord",
		"pow",
		"print",
		"property",
		"quit",
		"range",
		"repr",
		"reversed",
		"round",
		"set",
		"setattr",
		"slice",
		"sorted",
		"staticmethod",
		"str",
		"sum",
		"super",
		"tuple",
		"type",
		"vars",
		"zip":
		return true
	default:
		return false
	}
}

func mustRawNode(r interface{}, fmtstr string, args ...interface{}) RawNode {
	if r == nil {
		panic("nil raw node")
	}
	rr, ok := r.(map[interface{}]interface{})
	if !ok {
		panic(fmt.Errorf(fmtstr+" expected raw node to be of type map[string]interface{}, but got %T:", append(args, r)...))
	}

	rn := make(RawNode)
	for k, v := range rr {
		kk := mustString(k, fmtstr+" error parsing key in raw node: ", args...)
		rn[kk] = v
	}

	return rn
}

func mustString(s interface{}, fmtstr string, args ...interface{}) string {
	ss, ok := s.(string)
	if !ok {
		panic(fmt.Errorf(fmtstr+" expected type string, got %T", append(args, s)...))
	}
	return ss
}

func path(n RawNode) string {
	for k, v := range n {
		if k == "path" {
			return mustString(v, "error getting path from node: ")
		}
	}
	return ""
}

// Validate that the underlying nodes are syntactically correct
func (b Builder) Validate() error {
	// validate types
	var msgs []string
	for hash, typ := range b.Types {
		if typ == nil {
			msgs = append(msgs, fmt.Sprintf("type `%d` is nil", hash))
			continue
		}

		var ident string
		switch {
		case !typ.Path.Empty():
			ident = typ.Path.String()
		default:
			ident = pretty.Sprintf("%# v", typ)
		}

		if msg := validatePath(hash, typ.Path); msg != "" {
			msgs = append(msgs, fmt.Sprintf("type `%s` has invalid path: %s", ident, msg))
		}

		if msg := validateMap(typ.Attrs); msg != "" {
			msgs = append(msgs, fmt.Sprintf("attrs for type `%s` are invalid: %s", ident, msg))
		}

		if msg := validateMap(typ.Methods); msg != "" {
			msgs = append(msgs, fmt.Sprintf("methods for type `%s` are invalid: %s", ident, msg))
		}
	}

	// validate functions
	for hash, fn := range b.Functions {
		if fn == nil {
			msgs = append(msgs, fmt.Sprintf("function `%d` is nil", hash))
			continue
		}

		var ident string
		switch {
		case !fn.Path.Empty():
			ident = fn.Path.String()
		default:
			ident = pretty.Sprintf("%# v", fn)
		}

		if msg := validatePath(hash, fn.Path); msg != "" {
			msgs = append(msgs, fmt.Sprintf("type `%s` has invalid path: %s", ident, msg))
		}

		if fn.Varargs != nil {
			if msg := validateParam(*fn.Varargs); msg != "" {
				msgs = append(msgs, fmt.Sprintf("function `%s` has invalid varargs: %s", ident, msg))
			}
		}

		if fn.Kwargs != nil {
			if msg := validateParam(*fn.Kwargs); msg != "" {
				msgs = append(msgs, fmt.Sprintf("function `%s` has invalid kwargs: %s", ident, msg))
			}
		}

		for _, p := range fn.Params {
			if msg := validateParam(p); msg != "" {
				msgs = append(msgs, fmt.Sprintf("function `%s` has invalid parameter: %s", ident, msg))
			}
		}

		for _, ty := range fn.Return {
			if !validType(ty) {
				msgs = append(msgs, fmt.Sprintf("return type `%s` for `%s` contains illegal characters", ty, ident))
			}
		}
	}

	// validate modules
	for hash, m := range b.Modules {
		if m == nil {
			msgs = append(msgs, fmt.Sprintf("module `%d` is nil", hash))
			continue
		}

		var ident string
		switch {
		case !m.Path.Empty():
			ident = m.Path.String()
		default:
			ident = pretty.Sprintf("%# v", m)
		}

		if msg := validatePath(hash, m.Path); msg != "" {
			msgs = append(msgs, fmt.Sprintf("module `%s` has invalid path: %s", ident, msg))
		}

		if msg := validateMap(m.Attrs); msg != "" {
			msgs = append(msgs, fmt.Sprintf("attrs for module `%s` are invalid: %s", ident, msg))
		}

		if msg := validateMap(m.Functions); msg != "" {
			msgs = append(msgs, fmt.Sprintf("functions for module `%s` are invalid: %s", ident, msg))
		}

		if msg := validateMap(m.Types); msg != "" {
			msgs = append(msgs, fmt.Sprintf("types for module `%s` are invalid: %s", ident, msg))
		}

		// submodules will be verified separately just as modules, since they are included in the module table.
	}

	// validate attrs
	for hash, a := range b.Attrs {
		if a == nil {
			msgs = append(msgs, fmt.Sprintf("attr `%d` is nil", hash))
			continue
		}

		var ident string
		switch {
		case !a.Path.Empty():
			ident = a.Path.String()
		default:
			ident = pretty.Sprintf("%# v", a)
		}

		if msg := validatePath(hash, a.Path); msg != "" {
			msgs = append(msgs, fmt.Sprintf("attr `%s` has invalid path: %s", ident, msg))
		}

		if !validType(a.Type) {
			msgs = append(msgs, fmt.Sprintf("attr `%s` has type `%s` that contains illegal characters or is empty", ident, a.Type))
		}

	}

	if len(msgs) > 0 {
		return fmt.Errorf("python skeletons not valid\n%s", strings.Join(msgs, "\n"))
	}
	return nil
}

func validatePath(hash pythonimports.Hash, path pythonimports.DottedPath) string {
	var msgs []string
	if path.Hash != hash {
		msgs = append(msgs, fmt.Sprintf("global hash `%d` != local hash `%d` for path `%s`", hash, path.Hash, path.String()))
	}

	if path.Empty() {
		msgs = append(msgs, fmt.Sprintf("empty path for global hash `%d`", hash))
	}

	for _, part := range path.Parts {
		if !pythonscanner.IsValidIdent(part) {
			msgs = append(msgs, fmt.Sprintf("part `%s` (of path `%s`) contains illegal characters or is empty", part, path.String()))
		}
	}

	return strings.Join(msgs, ", ")
}

func validateMap(m map[string]string) string {
	var msgs []string
	for a, path := range m {
		if !pythonscanner.IsValidIdent(a) {
			msgs = append(msgs, fmt.Sprintf("name `%s` contains illegal characters or is empty", a))
		}
		if !validType(path) {
			msgs = append(msgs, fmt.Sprintf("path `%s` contains illegal characters or is empty", path))
		}
	}
	return strings.Join(msgs, ", ")
}

func validateParam(p Param) string {
	var msgs []string
	if p.Name == "" {
		msgs = append(msgs, "empty name")
	}

	if p.Default != "" && !validType(p.Default) {
		msgs = append(msgs, fmt.Sprintf("default type `%s` contains illegal characters or is empty", p.Default))
	}

	for _, t := range p.Types {
		if !validType(t) {
			msgs = append(msgs, fmt.Sprintf("type `%s` contains illegal characters or is empty", t))
		}
	}

	return strings.Join(msgs, ", ")
}

var validTypeRegex = regexp.MustCompile("([a-zA-Z0-9._]+)")

func validType(p string) bool {
	return p != "" && validTypeRegex.FindString(p) == p
}
