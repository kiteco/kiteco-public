package python

import (
	"fmt"
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

type resolveInputs struct {
	LocalIndex  *pythonlocal.SymbolIndex
	BufferIndex *bufferIndex
	Resolved    *pythonanalyzer.ResolvedAST
	Graph       pythonresource.Manager
	PrintDebug  func(fmtstr string, values ...interface{})
}

func (i resolveInputs) print(fmtstr string, values ...interface{}) {
	if i.PrintDebug == nil {
		return
	}
	i.PrintDebug("  "+fmtstr+"\n", values...)
}

func valueName(val pythontype.Value) string {
	addr := val.Address()
	// if addr.Nil(), we'll end up returning ""
	if addr.Path.Empty() {
		return strings.TrimSuffix(path.Base(addr.File), ".py")
	}
	return addr.Path.Last()
}

func sourceValueNamespace(ctx kitectx.Context, idx *pythonlocal.SymbolIndex, val pythontype.SourceValue) pythontype.Value {
	ctx.CheckAbort()

	addr := val.Address()
	if addr.Nil() {
		return nil
	}

	if len(addr.Path.Parts) == 0 {
		namespace, _ := idx.Package(ctx, addr.File)
		// we can't just return namespace here, as namespace might be a nil *pythontype.SourcePackage!
		// instead we must return a nil pythontype.Value
		if namespace == nil {
			return nil
		}
		return namespace
	}

	namespace, _ := idx.FindValue(ctx, addr.File, addr.Path.Predecessor().Parts)
	return namespace
}

func globalValueNamespace(inputs resolveInputs, val pythontype.GlobalValue) pythontype.Value {
	switch val := val.(type) {
	case pythontype.External:
		sym := val.Symbol().Canonical()
		inputs.print("symbol namespace %s", sym.String())

		path := sym.Path()
		switch len(path.Parts) {
		case 0:
			// no name, nothing else we can do.
			return nil
		case 1:
			// top level pacakge; nothing we can do
			return pythontype.ExternalRoot{Graph: inputs.Graph}
		default:
			if pred, err := inputs.Graph.NewSymbol(sym.Dist(), path.Predecessor()); err == nil {
				return pythontype.NewExternal(pred, inputs.Graph)
			}
		}
		return nil
	case pythontype.ExternalInstance:
		// TODO what to do here?
		return nil
	}

	return nil
}

func resolveName(ctx kitectx.Context, inputs resolveInputs, name *pythonast.NameExpr) []symbolBundle {
	safePrint := inputs.print
	safePrint("resolveName:\n")

	ref := inputs.Resolved.RefinedValue(name)
	switch {
	case name == nil:
		rollbar.Error(fmt.Errorf("resolveName: nil name"))
		safePrint("nil name")
		return nil
	case ref == nil:
		safePrint("nil value")
		return nil
	}

	valueNamespace := func(val pythontype.Value) (pythontype.Value, string) {
		// if the value is global, then try computing the appropriate
		switch val := val.(type) {
		case pythontype.GlobalValue:
			if inputs.Graph == nil {
				safePrint("no global graph, done")
				return nil, ""
			}
			return globalValueNamespace(inputs, val), valueName(val)
		case pythontype.SourceValue:
			if inputs.LocalIndex == nil {
				safePrint("no index, done")
				return nil, ""
			}
			safePrint("trying local for address %v", val.Address())
			return sourceValueNamespace(ctx, inputs.LocalIndex, val), valueName(val)
		case pythontype.NoneConstant:
			if builtinSym, err := inputs.Graph.PathSymbol(pythonimports.NewPath("builtins")); err == nil {
				return pythontype.NewExternal(builtinSym, inputs.Graph), "None"
			}
			return nil, ""
		case pythontype.BoolConstant:
			attr := "False"
			if bool(val) {
				attr = "True"
			}

			if builtinSym, err := inputs.Graph.PathSymbol(pythonimports.NewPath("builtins")); err == nil {
				return pythontype.NewExternal(builtinSym, inputs.Graph), attr
			}
			return nil, ""
		case pythontype.ConstantValue:
			// TODO what to do here?
			return nil, ""
		case pythontype.Union:
			safePrint("cannot handle unions in valueNamespace")
			return nil, ""
		default:
			safePrint("unhandled type in valueNamespace %T", val)
		}
		return nil, ""
	}

	bufferNamespace := func(val pythontype.Value) (pythontype.Value, string) {
		safePrint("trying buffer namespace")
		// try looking for the given nameexpr in the local index for the current file
		if inputs.LocalIndex != nil || inputs.BufferIndex != nil {
			safePrint("trying symbol table for current module")

			table, _ := inputs.Resolved.TableAndScope(name)
			if table != nil {
				safePrint("found table %s", table.Name.String())

				sym := table.Find(name.Ident.Literal)
				if sym != nil {
					safePrint("found symbol %s", sym.Name.String())

					if inputs.LocalIndex != nil {
						namespace, val, _ := inputs.LocalIndex.FindSymbol(ctx, sym.Name.File, sym.Name.Path.Predecessor().Parts, sym.Name.Path.Last())
						if namespace != nil && val != nil {
							safePrint("found namespace %v for val %v", pythontype.String(namespace), pythontype.String(val))
							// make sure namespace and symbol value both resolve.
							return namespace, sym.Name.Path.Last()
						}
					}

					if inputs.BufferIndex != nil {
						switch val.(type) {
						case pythontype.GlobalValue:
							// don't look up global values in the buffer index
						default:
							safePrint("falling back to buffer index based namespace")

							namespace, val, _ := inputs.BufferIndex.FindSymbol(ctx, sym.Name.File, sym.Name.Path.Predecessor().Parts, sym.Name.Path.Last())
							if namespace != nil && val != nil {
								safePrint("found namespace %v for val %v", pythontype.String(namespace), pythontype.String(val))
								return namespace, sym.Name.Path.Last()
							}
						}
					}
				}
			}
		}

		// fall back to value-based logic
		safePrint("falling back to value based namespace")
		return valueNamespace(val)
	}

	// NOTE: we MUST translate and disjunct the symbol value before
	// looking for a namespace in order to address
	// https://github.com/kiteco/kiteco/issues/5743
	nameVal := pythontype.Translate(ctx, ref, inputs.Graph)
	safePrint("name value: %s", pythontype.String(nameVal))

	indexBundle := indexBundle{
		idx:   inputs.LocalIndex,
		graph: inputs.Graph,
		bi:    inputs.BufferIndex,
	}

	// map from namespace hash to symbolBundle so that we can uniquify over namespaces
	sbs := make(map[pythontype.FlatID]symbolBundle)

	parentStmt := inputs.Resolved.ParentStmts[name]

	for _, val := range pythontype.Disjuncts(ctx, nameVal) {
		var namespace pythontype.Value
		var nsName string
		switch imp := parentStmt.(type) {
		case *pythonast.ImportNameStmt:
			safePrint("import name stmt")
			// determine where in the import statement the name is
			// e.g import foo.bar as car, far.bar as star
			var dotted *pythonast.DottedAsName
			for _, n := range imp.Names {
				if !tokenBetween(name.Ident, n.Begin(), n.End()) {
					continue
				}
				dotted = n
				break
			}

			// approximate node; nothing to do
			if dotted == nil {
				safePrint("no dotted name found")
				break
			}

			if dotted.Internal != nil && tokenAt(name.Ident, dotted.Internal.Begin(), dotted.Internal.End()) {
				// in "car" or "star", this symbol is defined in
				// the enclosing lexical scope.
				namespace, nsName = bufferNamespace(val)
				break
			}

			// in "foo.bar" or "far.bar"
			for i, n := range dotted.External.Names {
				if !tokenAt(name.Ident, n.Begin(), n.End()) {
					// find the part matching the NameExpr
					continue
				}
				if i == 0 {
					if dotted.Internal != nil {
						// if we have an internal alias then the root name
						// is not added to the namespace and symbol
						// is defined in an external module/pacakge
						namespace, nsName = valueNamespace(val)
					} else {
						// in "foo" or "far", symbol is defined
						// in enclosing lexical scope.
						// TODO we may want to do
						// namespace = bufferNamespace()
						// instead of
						namespace, nsName = valueNamespace(val)
					}
				} else {
					// in "bar", symbol is defined in namespace of previous name.
					namespace = inputs.Resolved.References[dotted.External.Names[i-1]]
					nsName = dotted.External.Names[i].Ident.Literal
				}
				break
			}

		case *pythonast.ImportFromStmt:
			safePrint("import from stmt")
			// determine where in the import from statement the name is
			// e.g from foo.bar import car as star, mar as far

			if imp.Package != nil && tokenBetween(name.Ident, imp.Package.Begin(), imp.Package.End()) {
				// in "foo.bar"
				for i, n := range imp.Package.Names {
					if !tokenAt(name.Ident, n.Begin(), n.End()) {
						continue
					}

					if i == 0 {
						// in "foo", symbol is defined in an external module/package
						namespace, nsName = valueNamespace(val)
					} else {
						// in "bar", symbol is defined in namespace of previous name.
						namespace = inputs.Resolved.References[imp.Package.Names[i-1]]
						nsName = imp.Package.Names[i].Ident.Literal
					}
					break
				}
				break
			}

			// in "car as star" or "mar as far"
			for _, ia := range imp.Names {
				if !tokenBetween(name.Ident, ia.Begin(), ia.End()) {
					continue
				}

				// TODO(juan): hmm in the case we have internal == nil
				// and the user is hovering over the external name
				// then it also makes sense to define the id of the symbol
				// using the name in the current module,
				// e.g if we have from foo import bar the symbol bar
				// also lives in the namespace of the scope that encloses the import...
				// NOTE: need to make sure Package is not nil in the case of an approximate node
				if tokenAt(name.Ident, ia.External.Begin(), ia.External.End()) {
					// in "car" or "mar", symbol defined in namespace of package
					if imp.Package != nil {
						namespace = inputs.Resolved.References[imp.Package]
						nsName = ia.External.Ident.Literal
					} else if inputs.LocalIndex != nil {
						// TODO nil checks necessary?
						namespace, _ = inputs.LocalIndex.Package(ctx, inputs.Resolved.Module.Address().File)
						nsName = ia.External.Ident.Literal
					}
				} else if ia.Internal != nil {
					// "star" or "far", symbol is defined in enclosing lexical scope
					namespace, nsName = bufferNamespace(val)
				}
				break
			}

		default:
			safePrint("default parent %s", pythonast.String(imp))
			// first check local index, if this is available then
			// use the result from the local index,
			// otherwise check if the name's value
			// resolved to a global node.
			namespace, nsName = bufferNamespace(val)
		}

		if namespace == nil {
			safePrint("no namespace for %s, skipping", pythontype.String(val))
			continue
		}
		if nsName == "" {
			nsName = name.Ident.Literal
		}

		safePrint("found namespace %v", namespace)
		safePrint("found nsName %s", nsName)

		// NOTE: still need to do the disjuncts here
		// since the namespace can still be a union.
		// e.g consider `from json import dumps as dd` with the cursor
		// over `dumps`, in this case the namespace (`json`) is still
		// a union because we have a version of json for python 2 and 3.
		// NOTE: since removing python 2 libraries, the above example
		// is no longer relevant but it is still the case that the namespace
		// can be a union.
		for _, ns := range pythontype.Disjuncts(ctx, namespace) {
			nsHash, err := pythontype.Hash(ctx, ns)
			if err != nil {
				safePrint("error hashing namespace disjunct %s: %s", pythontype.String(ns), err)
				continue
			}
			if _, ok := sbs[nsHash]; ok {
				continue
			}

			if nsFun, ok := ns.(*pythontype.SourceFunction); ok {
				sym, ok := nsFun.Locals.Table[nsName]
				if !ok {
					safePrint("could not find symbol in namespace disjunct %s: %s", pythontype.String(nsFun), err)
					continue
				}
				val = sym.Value
			} else {
				attrResult, err := pythontype.Attr(ctx, ns, nsName)
				if err != nil || !attrResult.Found() {
					safePrint("could not find symbol in namespace disjunct %s: %s", pythontype.String(ns), err)
					continue
				}
				val = attrResult.Value()
			}

			safePrint("namespace: %s, val: %v, nsName: %v\n", pythontype.String(namespace), pythontype.String(val), nsName)
			sbs[nsHash] = symbolBundle{
				ns:          newValueBundle(ctx, ns, indexBundle),
				name:        name.Ident.Literal,
				nsName:      nsName,
				valueBundle: newValueBundle(ctx, val, indexBundle),
			}
		}
	}

	safePrint("found %d symbol bundles", len(sbs))
	sbSlice := make([]symbolBundle, 0, len(sbs))
	for _, sb := range sbs {
		sbSlice = append(sbSlice, sb)
	}
	return sbSlice
}

func resolveAttr(ctx kitectx.Context, inputs resolveInputs, attr *pythonast.AttributeExpr) []symbolBundle {
	safePrint := inputs.print
	safePrint("resolveAttr:\n")
	ctx.CheckAbort()

	baseRef := inputs.Resolved.RefinedValue(attr.Value)
	if baseRef == nil {
		return nil
	}

	baseVal := pythontype.Translate(ctx, baseRef, inputs.Graph)
	safePrint("base val: %s", pythontype.String(baseVal))

	var sbs []symbolBundle
	for _, base := range pythontype.Disjuncts(ctx, baseVal) {
		safePrint("disjunct val: %s", pythontype.String(base))
		// Always do attr lookup on the instance,
		// this is because structured models such as list<int>
		// may have specialized attribute overrides
		// that return more specific information for the value
		// of the symbol.
		// Note that it is safe to do the attr lookup
		// on the type for a specialized model because
		// we always have that type(x<y>) == type(x<z>)
		// for all types x,y, and z.
		safePrint("looking up attr: %s", attr.Attribute.Literal)
		res, _ := pythontype.Attr(ctx, base, attr.Attribute.Literal)
		if !res.Found() {
			continue
		}

		val := res.Value()
		safePrint("res val: %s", pythontype.String(val))
		if base.Kind() == pythontype.InstanceKind {
			// selecting the attribute of an instance,
			// the symbol we want to link back to is defined on
			// the type of the instance rather than the instance itself
			base = base.Type()

			// TODO(naman) ideally this check could be encapsulated in another function in pythonstatic/pythontype
			// that is reused by the propagator, but that is a larger change, so we do this here for now.
			if prop, ok := val.(*pythontype.PropertyInstance); ok {
				if fun, ok := prop.FGet.(*pythontype.SourceFunction); ok {
					val = fun.Return.Value
					safePrint("updated res val: %s", pythontype.String(val))
				}
				// for non-source-function values, we could do:
				// if ref := inputs.Resolved.References[attr]; ref != nil {
				// 	val = ref.Value
				// }
				// but this may result in complications with consistency over unions, so we ignore this case for now
			}
		}

		sb := newValueBundle(ctx, base, indexBundle{
			idx:   inputs.LocalIndex,
			graph: inputs.Graph,
			bi:    inputs.BufferIndex,
		}).memberSymbol(ctx, val, attr.Attribute.Literal)

		sbs = append(sbs, sb)
	}
	return sbs
}

type nodeNotFoundError struct{}
type unsupportedNodeError struct{}
type resolutionError string

func (e nodeNotFoundError) Error() string {
	return "no node containing selection"
}
func (e unsupportedNodeError) Error() string {
	return "deepest node containing selection is not name/attribute expression"
}
func (e resolutionError) Error() string {
	return fmt.Sprintf("could not resolve symbol for %s expression", string(e))
}

func resolveNode(ctx kitectx.Context, node pythonast.Node, inputs resolveInputs) (string, []symbolBundle, error) {
	var nodeType string
	var sbs []symbolBundle
	switch expr := node.(type) {
	case *pythonast.NameExpr:
		nodeType = "name"
		sbs = resolveName(ctx, inputs, expr)
	case *pythonast.AttributeExpr:
		nodeType = "attr"
		sbs = resolveAttr(ctx, inputs, expr)
	case *pythonast.CallExpr:
		nodeType = "call"
		switch expr.Func.(type) {
		case *pythonast.NameExpr:
			sbs = resolveName(ctx, inputs, expr.Func.(*pythonast.NameExpr))
		case *pythonast.AttributeExpr:
			sbs = resolveAttr(ctx, inputs, expr.Func.(*pythonast.AttributeExpr))
		}
	case nil:
		return "", nil, nodeNotFoundError{}
	default:
		return "", nil, unsupportedNodeError{}
	}

	if len(sbs) == 0 {
		return "", nil, resolutionError(nodeType)
	}

	sbChoice(sbs)
	return nodeType, sbs, nil
}
