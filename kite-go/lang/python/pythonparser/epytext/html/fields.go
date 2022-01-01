package html

import (
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
	nethtml "golang.org/x/net/html"
)

var (
	paramsKwVarsFields = []string{
		"param",
		"var",
		"cvar",
		"ivar",
		"keyword",
		"type",
	}

	retAndRaiseFields = []string{
		"return",
		"rtype",
		"raise",
	}

	restKnownFields = []string{
		"summary",
		"note",
		"version",
		"todo",
		"see",
		"requires",
		"precondition",
		"postcondition",
		"invariant",
		"status",
		"change",
		"permission",
		"bug",
		"since",
		"attention",
		"deprecated",
		"author",
		"organization",
		"copyright",
		"warning",
		"license",
		"contact",
	}

	// value is a slice of labels, 1 is singular, 2 is plural
	// if different (otherwise 1 is used).
	fieldLabels = map[string][]string{
		"summary":       {"Summary"},
		"note":          {"Note", "Notes"},
		"version":       {"Version", "Versions"},
		"todo":          {"To Do"},
		"see":           {"See Also"},
		"requires":      {"Requires"},
		"precondition":  {"Precondition", "Preconditions"},
		"postcondition": {"Postcondition", "Postconditions"},
		"invariant":     {"Invariant", "Invariants"},
		"status":        {"Status"},
		"change":        {"Change Log"},
		"permission":    {"Permission", "Permissions"},
		"bug":           {"Bug", "Bugs"},
		"since":         {"Since"},
		"attention":     {"Attention"},
		"deprecated":    {"Deprecated"},
		"author":        {"Author", "Authors"},
		"organization":  {"Organization", "Organizations"},
		"copyright":     {"Copyright"},
		"warning":       {"Warning", "Warnings"},
		"license":       {"License"},
		"contact":       {"Contact", "Contacts"},
	}
)

type (
	// subFieldDef holds the slice of *nethtml.Node (the rendered content of the
	// field definition) for a "sub-field" (field argument, e.g. param name, type of param,
	// may be empty if the field has no argument such as "note" or "warning").
	subFieldDef map[string][]*nethtml.Node

	// fieldsDef holds the definitions for a list of fields. The map's key
	// is the normalized field name (e.g. "param", "type", "note").
	fieldsDef map[string]subFieldDef
)

// called during the visiting of the epytext AST, this stores the field blocks
// for deferred processing at the end, so that fields can be grouped correctly.
func (s *stack) saveFieldBlock(f *ast.FieldBlock, node *nethtml.Node) {
	if s.fields == nil {
		s.fields = make(fieldsDef)
	}

	field := normalizeFieldName(f.Name)
	if field == "param" {
		// for parameters only, keep original ordering
		s.paramsOrder = append(s.paramsOrder, f.Arg)
	}

	m := s.fields[field]
	if m == nil {
		m = make(subFieldDef)
		s.fields[field] = m
	}
	nodes := m[f.Arg]
	nodes = append(nodes, node)
	m[f.Arg] = nodes
}

// normalize the field name (e.g. arg is a synonym for param),
// case-insensitive.
func normalizeFieldName(name string) string {
	name = strings.ToLower(name)
	switch name {
	case "parameter", "arg", "argument":
		name = "param"
	case "returns":
		name = "return"
	case "returntype":
		name = "rtype"
	case "raises", "except", "exception":
		name = "raise"
	case "kwarg", "kwparam":
		name = "keyword"
	case "ivariable":
		name = "ivar"
	case "cvariable":
		name = "cvar"
	case "variable":
		name = "var"
	case "seealso":
		name = "see"
	case "warn":
		name = "warning"
	case "require", "requirement":
		name = "requires"
	case "precond":
		name = "precondition"
	case "postcod":
		name = "postcondition"
	case "org":
		name = "organization"
	case "(c)":
		name = "copyright"
	case "changed":
		name = "change"
	}
	return name
}

// rendering of the field blocks is called near the end of the HTML rendering,
// before closing the <body> (the parent parameter).
func (s *stack) renderFieldBlocks(parent *nethtml.Node) {
	paramsOrder := deduplicateKeepLast(s.paramsOrder)

	// declare the "return" field if a return type exists.
	if _, ok := s.fields["rtype"]; ok {
		if _, ok := s.fields["return"]; !ok {
			s.fields["return"] = subFieldDef{"": nil}
		}
	}

	var paramsKwVarsCount int
	for _, field := range paramsKwVarsFields {
		paramsKwVarsCount += len(s.fields[field])
	}
	var retAndRaiseCount int
	for _, field := range retAndRaiseFields {
		retAndRaiseCount += len(s.fields[field])
	}
	if paramsKwVarsCount+retAndRaiseCount > 0 {
		// render the "param"/"var"/"cvar"/"ivar"/"keyword" + "type",
		// "return" + "rtype" and "raise" fields in the same <dl>
		// (as epydoc does).
		dln := appendTree(parent, dl)

		if paramsKwVarsCount > 0 {
			appendTree(dln, dt, text("Parameters:"))
			ddn := appendTree(dln, dd, ul)
			uln := ddn.FirstChild

			// params first
			renderNamesWithOptionalTypes(uln, li, s.fields["param"], s.fields["type"], paramsOrder)
			// then keyword
			renderNamesWithOptionalTypes(uln, li, s.fields["keyword"], s.fields["type"], nil)
			// then ivars
			renderNamesWithOptionalTypes(uln, li, s.fields["ivar"], s.fields["type"], nil)
			// then vars
			renderNamesWithOptionalTypes(uln, li, s.fields["var"], s.fields["type"], nil)
			// then cvars
			renderNamesWithOptionalTypes(uln, li, s.fields["cvar"], s.fields["type"], nil)
			// then types without corresponding param/var
			renderNamesWithOptionalTypes(uln, li, nil, s.fields["type"], nil)
		}

		// then return value
		if ret, ok := s.fields["return"]; ok {
			dtn := appendTree(dln, dt, text("Returns: "))
			if retType := s.fields["rtype"]; retType != nil {
				if nodes := retType[""]; len(nodes) > 0 { // no arg name for rtype
					// use last definition of rtype, coherent with @type for parameters
					typ := nodes[len(nodes)-1]
					appendUnwrappedNode(dtn, typ)
				}
			}

			ddn := appendTree(dln, dd)
			if nodes := ret[""]; len(nodes) > 0 {
				// first definition of @return, coherent with @param
				node := nodes[0]
				appendUnwrappedNode(ddn, node)
			}
		}

		// then exceptions
		if exc := s.fields["raise"]; exc != nil {
			appendTree(dln, dt, text("Raises:"))
			ddn := appendTree(dln, dd, ul)
			uln := ddn.FirstChild

			// no types for exceptions (the name is the type)
			renderNamesWithOptionalTypes(uln, li, exc, nil, nil)
		}
	}

	// remove already processed fields
	for _, field := range append(paramsKwVarsFields, retAndRaiseFields...) {
		delete(s.fields, field)
	}

	// the remaining fields are all rendered as <p> (single field) or <ul>
	// (multiple fields), inside a common <div>
	if len(s.fields) > 0 {
		divn := appendTree(parent, div)

		// first, the known fields, in the expected order
		for _, knownField := range restKnownFields {
			defs := s.fields[knownField]
			renderStandardField(divn, knownField, defs)
			delete(s.fields, knownField)
		}

		// finally, the unknown fields, sorted
		var fields []string
		for field := range s.fields {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		for _, unknownField := range fields {
			defs := s.fields[unknownField]
			renderStandardField(divn, unknownField, defs)
			delete(s.fields, unknownField)
		}
	}
}

func renderStandardField(parent *nethtml.Node, field string, subDefs subFieldDef) {
	var names []string
	for nm := range subDefs {
		names = append(names, nm)
	}
	sort.Strings(names)

	// by default, the field is the label (if unknown)
	singular, plural := field, field
	if labels := fieldLabels[field]; len(labels) > 0 {
		singular = labels[0]
		plural = singular
		if len(labels) > 1 {
			plural = labels[1]
		}
	}

	for _, nm := range names {
		nodes := subDefs[nm]
		if len(nodes) == 1 {
			txt := singular
			if nm != "" {
				txt += " (" + nm + ")"
			}
			txt += ":"
			pn := appendTree(parent, p, strong, text(txt))
			appendUnwrappedNode(pn, nodes[0])
		} else {
			txt := plural
			if nm != "" {
				txt += " (" + nm + ")"
			}
			txt += ":"
			appendTree(parent, strong, text(txt))

			uln := appendTree(parent, ul)
			for _, node := range nodes {
				lin := appendTree(uln, li)
				appendUnwrappedNode(lin, node)
			}
		}
	}
}

func renderNamesWithOptionalTypes(parent *nethtml.Node, container elem,
	namesDefs subFieldDef, typesDefs subFieldDef, order []string) {

	names := order
	nameSource := namesDefs
	if nameSource == nil {
		// special-case: if namesDefs is nil, then process the remaining
		// @type fields without corresponding @param/var/etc.
		nameSource = typesDefs
	}
	if len(names) == 0 {
		// build the names and ordering from nameSource
		for nm := range nameSource {
			names = append(names, nm)
		}
		sort.Strings(names)
	}

	for _, nm := range names {
		nameNodes := namesDefs[nm]
		typeNodes := typesDefs[nm]
		delete(typesDefs, nm) // once types are used for a param name, cannot be used for others

		lin := appendTree(parent, container, strong, code, text(nm))
		if len(typeNodes) > 0 {
			typeNode := typeNodes[len(typeNodes)-1] // use last type definition for this name
			appendTree(lin, text(" ("))
			appendUnwrappedNode(lin, typeNode)
			appendTree(lin, text(")"))
		}
		if len(nameNodes) > 0 {
			appendTree(lin, text(" - "))
			nameNode := nameNodes[len(nameNodes)-1] // use last definition for this name
			appendUnwrappedNode(lin, nameNode)
		}
	}
}

// deduplicate values in the ordered slice, to keep only the
// last instance of each value (consistent with how we use the last
// definition of a field in renderNamesWithOptionalTypes).
func deduplicateKeepLast(vals []string) []string {
	var uniq []string
	seen := make(map[string]bool)

	// loop in reverse order, keeping only the first time a value is seen.
	for i := len(vals) - 1; i >= 0; i-- {
		v := vals[i]
		if !seen[v] {
			uniq = append(uniq, v)
			seen[v] = true
		}
	}

	// finally reverse the uniq slice and we have our deduplicated ordered values
	for i := len(uniq)/2 - 1; i >= 0; i-- {
		j := len(uniq) - 1 - i
		uniq[i], uniq[j] = uniq[j], uniq[i]
	}
	return uniq
}
