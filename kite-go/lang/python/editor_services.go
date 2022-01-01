package python

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/status"
)

const (
	memberLimit      = 5  // number of members returned in TypeDetails and ModuleDetails structs
	completionsLimit = 20 // number of completions returned
)

type editorServices struct {
	services *Services
}

func newEditorServices(services *Services) *editorServices {
	return &editorServices{
		services: services,
	}
}

// RenderSymbol for the provided namespace, referent, and value.
// NOTE: only works for global values atm
// TODO(juan): nasty
func RenderSymbol(ctx kitectx.Context, rm pythonresource.Manager, namespace, referent pythontype.Value, ident string) *editorapi.Symbol {
	if namespace == nil || referent == nil {
		return nil
	}

	es := editorServices{
		services: &Services{
			ResourceManager: rm,
		},
	}

	sb := newValueBundle(ctx, namespace, indexBundle{graph: rm}).memberSymbol(ctx, referent, ident)
	return es.renderSymbol(ctx, sb)
}

//
// Top-level responses
//

// renderValue renders a non-Union value
func (s *editorServices) renderValue(ctx kitectx.Context, vb valueBundle) *editorapi.Value {
	ctx.CheckAbort()

	if vb.val == nil {
		return nil
	}

	typeVb := vb.valueType(ctx)
	var typeName string
	if typeVb.val != nil {
		typeName = pythonenv.Name(typeVb.val)
	}
	return &editorapi.Value{
		ID:     s.renderID(ctx, vb),
		Kind:   s.renderKind(vb),
		Repr:   s.renderValueRepr(ctx, vb),
		Type:   typeName,
		TypeID: s.renderID(ctx, typeVb),
	}
}

// renderValueExt renders a non-Union value with details
func (s *editorServices) renderValueExt(ctx kitectx.Context, vb valueBundle) *editorapi.ValueExt {
	ctx.CheckAbort()

	if vb.val == nil {
		return nil
	}

	var synopsis string
	if _, docs := s.services.findDocumentation(ctx, vb, vb.bufferIndex()); docs != nil {
		synopsis = docs.Description
	}
	return &editorapi.ValueExt{
		Value:     *s.renderValue(ctx, vb),
		Synopsis:  synopsis,
		Details:   s.renderDetails(ctx, vb),
		Ancestors: s.renderValueAncestors(ctx, vb),
	}
}

func (s *editorServices) renderUnion(ctx kitectx.Context, ub valueBundle) editorapi.Union {
	ctx.CheckAbort()

	var union editorapi.Union
	seen := make(map[string]bool)
	for _, vb := range ub.disjuncts(ctx) {
		rendered := s.renderValue(ctx, vb)
		id := fmt.Sprintf("%s-%s", rendered.ID.String(), rendered.TypeID.String())
		if seen[id] {
			continue
		}
		seen[id] = true

		union = append(union, rendered)
	}

	return union
}

func (s *editorServices) renderUnionExt(ctx kitectx.Context, ub valueBundle) editorapi.UnionExt {
	ctx.CheckAbort()

	var union editorapi.UnionExt
	seen := make(map[string]bool)
	for _, vb := range ub.disjuncts(ctx) {
		rendered := s.renderValueExt(ctx, vb)
		id := fmt.Sprintf("%s-%s", rendered.ID.String(), rendered.TypeID.String())
		if seen[id] {
			continue
		}
		seen[id] = true
		union = append(union, rendered)
	}
	return union
}

func (s *editorServices) renderSymbolBase(ctx kitectx.Context, sb symbolBundle) editorapi.SymbolBase {
	ctx.CheckAbort()

	res := editorapi.SymbolBase{
		IDName: editorapi.IDName{
			ID:   s.renderSymbolID(ctx, sb),
			Name: sb.name,
		},
	}

	if sb.ns.val != nil {
		res.Parent = &editorapi.IDName{
			ID:   s.renderID(ctx, sb.ns),
			Name: valueName(sb.ns.val),
		}
		if sb.nsName != "" {
			res.Name = sb.nsName
			// otherwise the "parentName.name" rendered by the UX won't be consistent
		}
	}

	return res
}

func (s *editorServices) renderSymbol(ctx kitectx.Context, sb symbolBundle) *editorapi.Symbol {
	ctx.CheckAbort()

	return &editorapi.Symbol{
		SymbolBase: s.renderSymbolBase(ctx, sb),
		Value:      s.renderUnion(ctx, sb.valueBundle),
		Namespace:  s.renderValue(ctx, sb.ns),
	}
}

func (s *editorServices) renderMemberSymbol(ctx kitectx.Context, sb symbolBundle) *editorapi.Symbol {
	ctx.CheckAbort()

	return &editorapi.Symbol{
		SymbolBase: s.renderSymbolBase(ctx, sb),
		Namespace:  s.renderValue(ctx, sb.ns),
		Value:      s.renderUnion(ctx, sb.valueBundle),
	}
}

func (s *editorServices) renderMemberSymbolExt(ctx kitectx.Context, sb symbolBundle) *editorapi.SymbolExt {
	ctx.CheckAbort()

	// TODO: parse and use actual docstring
	sym := &editorapi.SymbolExt{
		SymbolBase: s.renderSymbolBase(ctx, sb),
		Namespace:  s.renderValue(ctx, sb.ns),
		Value:      s.renderUnionExt(ctx, sb.valueBundle),
	}
	for _, v := range sym.Value {
		v.Synopsis = truncateText(v.Synopsis, maxDocLength)
	}
	return sym
}

const maxDocLength = 500

func truncateText(txt string, n int) string {
	if len(txt) > n {
		i := n + strings.Index(txt[n:], " ")
		return txt[:i]
	}
	return txt
}

func (s *editorServices) renderValueReport(ctx kitectx.Context, vb valueBundle) *editorapi.Report {
	ctx.CheckAbort()

	resp := &editorapi.Report{
		Definition: s.renderDefinition(ctx, vb),
		Examples:   s.renderExamples(ctx, vb),
	}

	handleGlobal := func(val pythontype.GlobalValue) {
		var symbol pythonresource.Symbol
		switch global := val.(type) {
		case pythontype.External:
			symbol = global.Symbol()
		case pythontype.ExternalInstance:
			symbol = global.TypeExternal.Symbol()
		}

		if doc := s.services.ResourceManager.Documentation(symbol); doc != nil {
			resp.DescriptionText = doc.Text
			resp.DescriptionHTML = doc.HTML
		}
	}

	switch val := vb.val.(type) {
	case pythontype.ConstantValue:
		if s.services.ResourceManager == nil {
			return resp
		}
		handleGlobal(pythontype.WidenConstant(val, s.services.ResourceManager))

	case pythontype.GlobalValue:
		// global docs
		if s.services.ResourceManager == nil {
			return resp
		}
		handleGlobal(val)

	case pythontype.SourceValue:
		if vb.idx == nil {
			return resp
		}
		docs, err := vb.idx.Documentation(ctx, val)
		if err != nil {
			return resp
		}

		resp.DescriptionText = docs.Description
		resp.DescriptionHTML = docs.HTML
	}

	return resp
}

func (s *editorServices) renderSymbolReport(ctx kitectx.Context, sb symbolBundle) *editorapi.Report {
	// TODO: Parse symbol docstrings and differentiate between symbol and
	// value docs. For now we'll just return the report for the value this
	// symbol holds.
	return s.renderValueReport(ctx, sb.valueBundle)
}

//
// ID's and repr's
//

// ValidateID checks that an ID built by pythonenv.Locator/SymbolLocator actually resolves to a value or symbol
func ValidateID(ctx kitectx.Context, rm pythonresource.Manager, idx *pythonlocal.SymbolIndex, bi *bufferIndex, origID string) editorapi.ID {
	ctx.CheckAbort()

	var finalID string
	if origID != "" {
		if addr, attr, err := pythonenv.ParseLocator(origID); err == nil {
			path := addr.Path
			if attr != "" {
				path = path.WithTail(attr)
			}
			if addr.File != "" {
				if idx != nil {
					if val, err := idx.FindValue(ctx, addr.File, path.Parts); err == nil && val != nil {
						finalID = origID
					}
				}

				if finalID == "" && bi != nil && bi.filepath == addr.File {
					if def, err := bi.FindValue(ctx, addr.File, path.Parts); err == nil && def != nil {
						finalID = origID
					}
				}
			} else {
				if _, err := rm.PathSymbol(path); err == nil {
					finalID = origID
				}
			}
		}
	}
	return editorapi.NewID(lang.Python, finalID)
}

func renderID(ctx kitectx.Context, rm pythonresource.Manager, vb valueBundle) editorapi.ID {
	ctx.CheckAbort()

	if vb.val == nil {
		validValueRatio.Miss()
		invalidValueSource.HitAndAdd("null value")
		invalidValueKind.HitAndAdd("undefined")
		return editorapi.NewID(lang.Python, "")
	}
	return ValidateID(ctx, rm, vb.idx, vb.bi, pythonenv.Locator(vb.val))
}

func renderSymbolID(ctx kitectx.Context, rm pythonresource.Manager, sb symbolBundle) editorapi.ID {
	ctx.CheckAbort()

	switch sb.ns.val.(type) {
	case nil:
		validSymbolRatio.Miss()

		// Break down by whether or not buffer index is available
		var bufSymbolSource, bufSymbolValueKind *status.Breakdown
		if sb.bufferIndex() != nil {
			bufSymbolSource = invalidBufferedSymbolSource
			bufSymbolValueKind = invalidMissingNamespaceBufferedSymbolValueKind
		} else {
			bufSymbolSource = invalidUnbufferedSymbolSource
			bufSymbolValueKind = invalidMissingNamespaceUnbufferedSymbolValueKind
		}

		invalidSymbolSource.HitAndAdd("missing namespace")
		bufSymbolSource.HitAndAdd("missing namespace")
		if sb.val == nil {
			invalidMissingNamespaceSymbolValueKind.HitAndAdd("null value")
			bufSymbolValueKind.HitAndAdd("null value")
		} else {
			invalidMissingNamespaceSymbolValueKind.HitAndAdd(sb.val.Kind().String())
			bufSymbolValueKind.HitAndAdd(sb.val.Kind().String())
		}

		return renderID(ctx, rm, sb.valueBundle)
	}

	raw := pythonenv.SymbolLocator(sb.ns.val, sb.nsName)
	id := ValidateID(ctx, rm, sb.idx, sb.bi, raw)
	trackSymbolID(raw, id.LanguageSpecific(), sb)
	return id
}

func (s *editorServices) renderID(ctx kitectx.Context, vb valueBundle) editorapi.ID {
	return renderID(ctx, s.services.ResourceManager, vb)
}

func (s *editorServices) renderSymbolID(ctx kitectx.Context, sb symbolBundle) editorapi.ID {
	return renderSymbolID(ctx, s.services.ResourceManager, sb)
}

// renderKind renders a non-union kind
func (s *editorServices) renderKind(vb valueBundle) editorapi.Kind {
	if vb.val == nil {
		return ""
	}
	return editorapi.Kind(vb.val.Kind().String())
}

func (s *editorServices) renderValueRepr(ctx kitectx.Context, vb valueBundle) string {
	ctx.CheckAbort()
	return strings.Join(pythontype.Reprs(ctx, vb.val, vb.graph, false), " | ")
}

//
// Value details
//

func (s *editorServices) renderDetails(ctx kitectx.Context, vb valueBundle) editorapi.Details {
	ctx.CheckAbort()

	var details editorapi.Details
	switch vb.val.Kind() {
	case pythontype.FunctionKind:
		details.Function = s.renderFunctionDetail(ctx, vb, pythonresource.Symbol{})
	case pythontype.TypeKind:
		details.Type = s.renderTypeDetail(ctx, vb)
	case pythontype.ModuleKind:
		details.Module = s.renderModuleDetail(ctx, vb)
	case pythontype.InstanceKind:
		details.Instance = s.renderInstanceDetail(ctx, vb)
	}
	return details
}

func (s *editorServices) renderFunctionDetail(ctx kitectx.Context, vb valueBundle, typeSymbol pythonresource.Symbol) *editorapi.FunctionDetails {
	ctx.CheckAbort()

	// If renderFunctionDetail is called by renderTypeDetail, typeNode should be non-nil.
	// typeNode allows us to lookup an argspec from Typeshed even when vb.Node is builtins.object.__init__,
	// which is usually the case when we don't have an import graph ArgSpec
	switch v := vb.val.(type) {
	case pythontype.External:
		// look for a Symbol-containing External in the disjuncts that has an argspec
		spec := s.services.ResourceManager.ArgSpec(v.Symbol())
		if spec == nil && !typeSymbol.Nil() {
			spec = s.services.ResourceManager.ArgSpec(typeSymbol)
		}
		if spec == nil {
			return nil
		}

		resolveString := func(str string) pythontype.Value {
			// - constants?

			switch str {
			case "", "...":
				return nil
			case "True":
				return pythontype.BoolConstant(true)
			case "False":
				return pythontype.BoolConstant(false)
			case "None":
				return pythontype.NoneConstant{}
			default:
				// try int
				pi, err := strconv.ParseInt(str, 10, 64)
				if err == nil {
					return pythontype.IntConstant(pi)
				}

				// try float
				pf, err := strconv.ParseFloat(str, 64)
				if err == nil {
					return pythontype.FloatConstant(pf)
				}
			}

			// - paths?

			tPath := pythonimports.NewDottedPath(str)
			sym, err := s.services.ResourceManager.PathSymbol(tPath)
			if err != nil {
				sym, err = s.services.ResourceManager.PathSymbol(pythonimports.NewPath("builtins").WithTail(tPath.Parts...))
			}
			if err != nil {
				sym, err = s.services.ResourceManager.PathSymbol(pythonimports.NewPath("types").WithTail(tPath.Parts...))
			}

			if err != nil && str == "generator" {
				sym, err = s.services.ResourceManager.PathSymbol(pythonimports.NewPath("types", "GeneratorType"))
			}

			if err != nil && str == "function" {
				sym, err = s.services.ResourceManager.PathSymbol(pythonimports.NewPath("types", "FunctionType"))
			}

			if err != nil && str == "builtins.NoneType" {
				sym, err = s.services.ResourceManager.PathSymbol(pythonimports.NewDottedPath("builtins.None.__class__"))
			}

			if err != nil {
				return nil
			}

			// canonicalize because the data has some oddities e.g. Tix.NoneType instead of types.NoneType
			// TODO(naman) fix data and remove from here?
			sym = sym.Canonical()
			return pythontype.NewExternal(sym, s.services.ResourceManager)
		}

		resolveTypeString := func(str string) pythontype.Value {
			// remove .type suffix if present
			if pos := strings.Index(str, ".type"); pos > -1 {
				str = str[:pos]
			}

			ty, ok := resolveString(str).(pythontype.Callable)
			if !ok {
				return nil
			}
			return ty.Call(pythontype.Args{})
		}

		var start int
		var recv *editorapi.Parameter
		if len(spec.Args) > 0 && (spec.Args[0].Name == "self" || spec.Args[0].Name == "cls") {
			recv = &editorapi.Parameter{
				Name: spec.Args[0].Name,
			}
			start = 1
		}

		params := make([]*editorapi.Parameter, 0, len(spec.Args)-start)
		for _, arg := range spec.Args[start:] {
			var types []pythontype.Value
			for _, t := range arg.Types {
				if val := resolveTypeString(t); val != nil {
					types = append(types, val)
				}
			}

			var defaultValue editorapi.Union
			if val := resolveString(arg.DefaultValue); val != nil {
				defaultValue = s.renderUnion(ctx, newValueBundle(ctx, val, vb.indexBundle))
			} else if val := resolveTypeString(arg.DefaultType); val != nil {
				defaultValue = s.renderUnion(ctx, newValueBundle(ctx, val, vb.indexBundle))
			} else if arg.DefaultValue != "" || arg.DefaultType != "" { // TODO(naman) can (arg.DefaultValue == "" && arg.DefaultType != "")?
				var repr string
				var kind string
				if arg.DefaultValue != "" {
					// pretend it's a string
					// TODO(naman) is it always a string here? we should store & pass around a more structured representation
					kind = "instance"
					if arg.DefaultValue == "..." || strings.HasPrefix(arg.DefaultValue, "\"") || strings.HasPrefix(arg.DefaultValue, "'") {
						repr = fmt.Sprintf("%s", arg.DefaultValue)
					} else {
						repr = fmt.Sprintf("\"%s\"", arg.DefaultValue)
					}
				} else if arg.DefaultType != "" {
					kind = "unknown"
					repr = arg.DefaultType
				} else {
					kind = "unknown"
					repr = "..."
				}
				defaultValue = []*editorapi.Value{&editorapi.Value{
					Kind: editorapi.Kind(kind),
					Repr: repr,
				}}
			}

			params = append(params, &editorapi.Parameter{
				Name: arg.Name,
				// explicitly render as a union so that we
				// can use unite logic to dedupe values
				InferredValue: s.renderUnion(ctx, newValueBundle(ctx, pythontype.Unite(ctx, types...), vb.indexBundle)),
				LanguageDetails: editorapi.LanguageParameterDetails{
					Python: &editorapi.PythonParameterDetails{
						DefaultValue: defaultValue,
						KeywordOnly:  arg.KeywordOnly,
					},
				},
			})
		}

		var kwparams []*editorapi.Parameter
		if args := s.services.ResourceManager.Kwargs(v.Symbol()); args != nil {
			for _, arg := range args.Kwargs {
				var types []pythontype.Value
				for _, t := range arg.Types {
					if val := resolveTypeString(t); val != nil {
						types = append(types, val)
					}
				}

				kwparams = append(kwparams, &editorapi.Parameter{
					Name: arg.Name,
					// explicitly render as a union so that we
					// can use unite logic to dedupe values
					InferredValue: s.renderUnion(ctx, newValueBundle(ctx, pythontype.Unite(ctx, types...), vb.indexBundle)),
				})
			}
		}

		var vararg *editorapi.Parameter
		if spec.Vararg != "" {
			vararg = &editorapi.Parameter{
				Name: spec.Vararg,
			}
		}

		var kwarg *editorapi.Parameter
		if spec.Kwarg != "" {
			kwarg = &editorapi.Parameter{
				Name: spec.Kwarg,
			}
		}

		retSyms := s.services.ResourceManager.ReturnTypes(v.Symbol())
		var retVals []pythontype.Value
		for _, sym := range retSyms {
			retVals = append(retVals, pythontype.ExternalInstance{TypeExternal: pythontype.NewExternal(sym, s.services.ResourceManager)})
		}
		ret := pythontype.Unite(ctx, retVals...)

		return &editorapi.FunctionDetails{
			Parameters:  params,
			ReturnValue: s.renderUnion(ctx, newValueBundle(ctx, ret, vb.indexBundle)),
			Signatures:  s.renderSignatures(ctx, vb),
			LanguageDetails: editorapi.LanguageFunctionDetails{
				Python: &editorapi.PythonFunctionDetails{
					Receiver:        recv,
					Vararg:          vararg,
					Kwarg:           kwarg,
					KwargParameters: kwparams,
				},
			},
		}

	case *pythontype.SourceFunction:
		var start int
		var recv *editorapi.Parameter
		if (v.HasReceiver || v.HasClassReceiver) && len(v.Parameters) > 0 {
			param := v.Parameters[0]
			recv = &editorapi.Parameter{
				Name:          param.Name,
				InferredValue: s.renderUnion(ctx, newValueBundle(ctx, param.Symbol.Value, vb.indexBundle)),
			}
			start = 1
		}

		params := make([]*editorapi.Parameter, 0, len(v.Parameters)-start)
		for _, param := range v.Parameters[start:] {
			dvs := s.renderUnion(ctx, newValueBundle(ctx, param.Default, vb.indexBundle))
			if len(dvs) > 0 && vb.idx != nil {
				if spec, _ := vb.idx.ArgSpec(ctx, vb.val); spec != nil {
					for _, arg := range spec.Args {
						if arg.Name == param.Name && arg.DefaultValue != "" {
							for _, dv := range dvs {
								dv.Repr = arg.DefaultValue
							}
							break
						}
					}
				}
			}
			params = append(params, &editorapi.Parameter{
				Name:          param.Name,
				InferredValue: s.renderUnion(ctx, newValueBundle(ctx, param.Symbol.Value, vb.indexBundle)),
				LanguageDetails: editorapi.LanguageParameterDetails{
					Python: &editorapi.PythonParameterDetails{
						KeywordOnly:  param.KeywordOnly,
						DefaultValue: dvs,
					},
				},
			})
		}

		var kwparams []*editorapi.Parameter
		if v.KwargDict != nil {
			for name, val := range v.KwargDict.Entries {
				kwparams = append(kwparams, &editorapi.Parameter{
					Name:          name,
					InferredValue: s.renderUnion(ctx, newValueBundle(ctx, val, vb.indexBundle)),
				})
			}
		}

		var vararg *editorapi.Parameter
		if v.Vararg != nil {
			vararg = &editorapi.Parameter{
				Name: v.Vararg.Name,
			}
		}

		var kwarg *editorapi.Parameter
		if v.Kwarg != nil {
			kwarg = &editorapi.Parameter{
				Name: v.Kwarg.Name,
			}
		}

		return &editorapi.FunctionDetails{
			Parameters:  params,
			ReturnValue: s.renderUnion(ctx, newValueBundle(ctx, v.Return.Value, vb.indexBundle)),
			Signatures:  s.renderSignatures(ctx, vb),
			LanguageDetails: editorapi.LanguageFunctionDetails{
				Python: &editorapi.PythonFunctionDetails{
					Receiver:        recv,
					Vararg:          vararg,
					Kwarg:           kwarg,
					KwargParameters: kwparams,
				},
			},
		}

	case pythontype.ExplicitFunc, pythontype.BoundMethod:
		// NOTE(juan): this should no longer happen?
		return s.renderFunctionDetail(ctx, newValueBundle(ctx, vb.val, vb.indexBundle), typeSymbol)

	default:
		log.Printf("got unexpected type: %T", v)
		return nil
	}
}

func (s *editorServices) renderTypeDetail(ctx kitectx.Context, vb valueBundle) *editorapi.TypeDetails {
	ctx.CheckAbort()

	// get a symbol if vb is external
	var vbSym pythonresource.Symbol
	// get base class bundles
	var bbs []valueBundle
	switch v := vb.val.(type) {
	case pythontype.ExplicitType:
		// TODO(juan): this should no longer happen?
		return s.renderTypeDetail(ctx, newValueBundle(ctx, vb.val, vb.indexBundle))

	case pythontype.External:
		vbSym = v.Symbol()
		for _, baseSym := range s.services.ResourceManager.Bases(v.Symbol()) {
			bbs = append(bbs, newValueBundle(ctx, pythontype.NewExternal(baseSym, vb.graph), vb.indexBundle))
		}

	case *pythontype.SourceClass:
		for _, base := range v.Bases {
			bbs = append(bbs, newValueBundle(ctx, base, vb.indexBundle))
		}

	default:
		log.Printf("got unexpected type: %T", v)
		return nil
	}

	// get member symbol bundles, and render
	msbs, total := s.memberSymbols(ctx, vb, 0, memberLimit)
	var members []*editorapi.Symbol
	for _, ms := range msbs {
		members = append(members, s.renderMemberSymbol(ctx, ms))
	}

	var bases []*editorapi.PythonBase
	for _, bb := range bbs {
		bases = append(bases, s.baseResponse(ctx, bb))
	}

	var cb valueBundle
	if init, err := pythontype.Attr(ctx, vb.val, "__init__"); err == nil && init.Found() {
		cb = vb.memberSymbol(ctx, init.Value(), "__init__").valueBundle
	}

	return &editorapi.TypeDetails{
		Members:      members,
		TotalMembers: total,
		LanguageDetails: editorapi.LanguageTypeDetails{
			Python: &editorapi.PythonTypeDetails{
				Bases:       bases,
				Constructor: s.renderFunctionDetail(ctx, cb, vbSym),
			},
		},
	}
}

func (s *editorServices) renderModuleDetail(ctx kitectx.Context, vb valueBundle) *editorapi.ModuleDetails {
	msbs, total := s.memberSymbols(ctx, vb, 0, memberLimit)

	var members []*editorapi.Symbol
	for _, msb := range msbs {
		members = append(members, s.renderMemberSymbol(ctx, msb))
	}
	return &editorapi.ModuleDetails{
		Members:      members,
		TotalMembers: total,
	}
}

func (s *editorServices) renderInstanceDetail(ctx kitectx.Context, vb valueBundle) *editorapi.InstanceDetails {
	return &editorapi.InstanceDetails{Type: s.renderUnion(ctx, vb.valueType(ctx))}
}

func (s *editorServices) baseResponse(ctx kitectx.Context, vb valueBundle) *editorapi.PythonBase {
	ctx.CheckAbort()

	if vb.val == nil {
		return nil
	}
	name := vb.val.Address().ShortName()
	moduleVb := vb.valueModule(ctx)
	var moduleName string
	if moduleVb.val != nil {
		moduleName = moduleVb.val.Address().ShortName()
	}
	return &editorapi.PythonBase{
		ID:       s.renderID(ctx, vb),
		Name:     name,
		Module:   moduleName,
		ModuleID: s.renderID(ctx, moduleVb),
	}
}

//
// Definitions, examples
//

func (s *editorServices) renderDefinition(ctx kitectx.Context, vb valueBundle) *editorapi.Definition {
	ctx.CheckAbort()

	if vb.val == nil {
		return nil
	}
	if _, ok := vb.val.(pythontype.SourceValue); !ok {
		// non-source node, do not return any definition
		return nil
	}
	if vb.idx == nil {
		return nil
	}

	// TODO(juan): move this into the lcw once we have sorted out the
	// defintion situation and moved everyting into values.
	switch v := vb.val.(type) {
	case *pythontype.SourceModule:
		native, err := pythonlocal.DefinitionPathForFile(v.Address().File)
		if err != nil {
			return nil
		}

		filepath, err := fileutil.GetProperCasingPath(native)
		if err != nil {
			log.Printf("error retrieving long path name: %s", err.Error())
		}

		return &editorapi.Definition{
			Filename: filepath,
			Line:     1,
		}
	case *pythontype.SourcePackage:
		// use the address of the underlying __init__.py
		if v.Init == nil {
			return nil
		}

		native, err := pythonlocal.DefinitionPathForFile(v.Init.Address().File)
		if err != nil {
			return nil
		}

		filepath, err := fileutil.GetProperCasingPath(native)
		if err != nil {
			log.Printf("error retrieving long path name: %s", err.Error())
		}

		return &editorapi.Definition{
			Filename: filepath,
			Line:     1,
		}
	default:
		// make sure to try index for current file
		// first to get the most recent version.
		var def *pythonlocal.Definition
		if buf := vb.bufferIndex(); buf != nil {
			def, _ = buf.Definition(v)
		}

		if def == nil {
			// static index
			def, _ = vb.idx.Definition(ctx, v)
		}

		if def == nil {
			return nil
		}

		filepath, err := fileutil.GetProperCasingPath(def.Path)
		if err != nil {
			log.Printf("error retrieving long path name: %s", err.Error())
		}

		return &editorapi.Definition{
			Filename: filepath,
			Line:     def.Line + 1,
		}
	}
}

func (s *editorServices) renderExamples(ctx kitectx.Context, vb valueBundle) []*editorapi.Example {
	if s.services.Curation == nil {
		return nil
	}

	if vb.val == nil {
		return nil
	}

	var global pythontype.GlobalValue
	switch val := vb.val.(type) {
	case pythontype.GlobalValue:
		global = val
	case pythontype.ConstantValue:
		global = pythontype.WidenConstant(val, s.services.ResourceManager)
	default:
		// examples are only for global code
		return nil
	}

	var css []*pythoncuration.Snippet
	var exists bool
	if global.Kind() == pythontype.ModuleKind {
		var symbol pythonresource.Symbol
		if ext, ok := global.(pythontype.External); ok {
			symbol = ext.Symbol()
		}
		css, exists = s.services.Curation.Canonical(symbol.PathString())
	} else {
		var symbol pythonresource.Symbol
		switch global := global.(type) {
		case pythontype.External:
			symbol = global.Symbol()
		case pythontype.ExternalInstance:
			symbol = global.TypeExternal.Symbol()
		}

		if node, err := s.services.ImportGraph.Navigate(symbol.Path()); node != nil && err == nil {
			css, exists = s.services.Curation.Examples(node)
		}
	}
	if !exists {
		return nil
	}

	var exs []*editorapi.Example
	for _, cs := range css {
		exs = append(exs, &editorapi.Example{
			ID:    cs.Curated.Snippet.SnippetID,
			Title: cs.Curated.Snippet.Title,
		})
	}
	return exs
}

//
// Signature patterns
//

func (s *editorServices) renderSignatures(ctx kitectx.Context, vb valueBundle) []*editorapi.Signature {
	ctx.CheckAbort()

	switch val := vb.val.(type) {
	case pythontype.GlobalValue:
		if ext, ok := val.(pythontype.External); ok {
			if patterns := s.services.ResourceManager.PopularSignatures(ext.Symbol()); patterns != nil {
				return patterns
			}
		}
	case pythontype.SourceValue:
		if vb.idx == nil {
			break
		}
		if patterns, err := vb.idx.MethodPatterns(ctx, val); err == nil {
			return pythoncode.EditorSignatures(patterns)
		}
	}
	return []*editorapi.Signature{}
}

//
// Members
//

func (s *editorServices) scoreMember(ctx kitectx.Context, idx *pythonlocal.SymbolIndex, val pythontype.Value) int {
	choice := pythontype.MostSpecific(ctx, val)
	if choice, ok := choice.(pythontype.External); ok {
		path := pythoncode.ValuePath(ctx, choice, s.services.ResourceManager)
		if path.Empty() {
			return 0
		}
		sym, err := s.services.ResourceManager.PathSymbol(path)
		if err != nil {
			return 0
		}

		counts := s.services.ResourceManager.SymbolCounts(sym)
		if counts == nil {
			return 0
		}

		return counts.Attribute + counts.Import
	}

	if idx != nil {
		if count, err := idx.ValueCount(ctx, val); err == nil {
			return count
		}
	}
	return 0
}

func (s *editorServices) memberSymbols(ctx kitectx.Context, vb valueBundle, offset, limit int) ([]symbolBundle, int) {
	ctx.CheckAbort()

	var members []symbolBundle
	switch val := vb.val.(type) {
	case pythontype.External, *pythontype.SourceClass, *pythontype.SourceModule, *pythontype.SourcePackage:
		allMembers := pythontype.Members(ctx, s.services.ResourceManager, vb.val)

		type scoredMember struct {
			attr  string
			val   pythontype.Value
			score int
		}
		scoredMembers := make([]scoredMember, 0, len(allMembers))
		for attr, val := range allMembers {
			scoredMembers = append(scoredMembers, scoredMember{
				attr:  attr,
				val:   val,
				score: s.scoreMember(ctx, vb.idx, val),
			})
		}

		sort.Slice(scoredMembers, func(i, j int) bool { return scoredMembers[i].score > scoredMembers[j].score })
		for _, scoredMember := range scoredMembers {
			members = append(members, vb.memberSymbol(ctx, scoredMember.val, scoredMember.attr))
		}
	case pythontype.ExplicitType:
		// TODO: wouldn't this cause infinite recursion? Figure out what to do here
		// TODO(juan): this should no longer happen?
		return s.memberSymbols(ctx, newValueBundle(ctx, val, vb.indexBundle), offset, limit)
	}

	total := len(members)

	if offset > 0 && offset < len(members) {
		members = members[offset:]
	}

	if limit > -1 && len(members) > limit {
		members = members[:limit]
	}

	return members, total
}

//
// Ancestors
//
func (s *editorServices) renderValueAncestors(ctx kitectx.Context, vb valueBundle) []editorapi.Ancestor {
	ctx.CheckAbort()

	if vb.val == nil {
		return nil
	}

	addr := vb.val.Address()
	if addr.Nil() {
		return nil
	}

	var ancestors []editorapi.Ancestor

	// check for global node
	if addr.File == "" {
		var ancestor pythontype.Address
		for _, part := range addr.Path.Parts[:len(addr.Path.Parts)-1] {
			ancestor = ancestor.WithTail(part)
			ancestors = append(ancestors, editorapi.Ancestor{
				ID:   editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(ancestor)),
				Name: part,
			})
		}
		return ancestors
	}

	// local node
	if vb.idx == nil {
		return nil
	}

	// find top level pkg that contains more than one entry
	parts := strings.Split(addr.File, "/")
	pkg := vb.idx.SourceTree.Dirs["/"]
	name := "/"
	var mod *pythontype.SourceModule
parts_loop:
	for i, part := range parts {
		if i == 0 {
			continue
		}
		if pkg == nil {
			return nil
		}
		part = strings.TrimSuffix(part, ".py")

		// first package with more than one entry
		// so we are done
		if len(pkg.DirEntries.Table) > 1 {
			parts = parts[i:]
			break
		}

		// look explicitly in
		// the dir entries to avoid collisions
		// between entiries in init that have the
		// same name as a directory.
		s := pkg.DirEntries.Table[part]
		if s == nil || s.Value == nil {
			return nil
		}
		val := s.Value

		switch val := val.(type) {
		case *pythontype.SourcePackage:
			pkg = val
			name = part
		case *pythontype.SourceModule:
			// hit a module without ever hitting
			// a package with more than one entry,
			// so we are done
			name = part
			parts = parts[i+1:]
			mod = val
			break parts_loop
		default:
			return nil
		}
	}

	switch {
	case mod != nil && len(parts) > 0:
		// sanity check, this should not happen
		return nil
	case mod != nil && addr.Path.Empty():
		// ended at a module and no sub components,
		// so the only ancestor is the package containing
		// the module.
		return []editorapi.Ancestor{
			editorapi.Ancestor{
				ID:   editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(pkg.DirEntries.Name)),
				Name: name,
			},
		}
	case mod != nil && !addr.Path.Empty():
		// ended at module but we are referencing a
		// component of the module so ancestor
		// starts with module.
		ancestors = append(ancestors, editorapi.Ancestor{
			ID:   editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(mod.Members.Name)),
			Name: name,
		})
	default:
		// ended in a package, ancestors start at that package.
		ancestors = append(ancestors, editorapi.Ancestor{
			ID:   editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(pkg.DirEntries.Name)),
			Name: name,
		})
	}

	// append rest of the packages/modules in the path
	val := pythontype.Value(pkg)
	for _, part := range parts {
		part = strings.TrimSuffix(part, ".py")
		res, _ := pythontype.Attr(ctx, val, part)
		if !res.ExactlyOne() {
			return nil
		}

		val = res.Value()
		if val == nil || val.Address().Nil() {
			return nil
		}

		ancestors = append(ancestors, editorapi.Ancestor{
			ID:   editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(val.Address())),
			Name: part,
		})
	}

	if addr.Path.Empty() {
		// looking for a module or package,
		// drop last member since it is the current value
		// and we only want ancestors
		return ancestors[:len(ancestors)-1]
	}

	if mod != nil {
		val = mod
	}
	for _, part := range addr.Path.Parts[:len(addr.Path.Parts)-1] {
		res, _ := pythontype.Attr(ctx, val, part)
		if !res.ExactlyOne() {
			return nil
		}

		val = res.Value()
		if val == nil || val.Address().Nil() {
			// this should not happen
			return nil
		}

		ancestors = append(ancestors, editorapi.Ancestor{
			ID:   editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(val.Address())),
			Name: part,
		})
	}

	return ancestors
}

//
// Completions
//

func (s *editorServices) typeHintValueCompletion(ctx kitectx.Context, vb valueBundle) string {
	ctx.CheckAbort()

	if vb.val == nil {
		return ""
	}

	vals := pythontype.Disjuncts(ctx, vb.val)
	knd := vals[0].Kind()
	for _, val := range vals[1:] {
		if val.Kind() != knd {
			knd = pythontype.UnionKind
			break
		}
	}

	switch knd {
	case pythontype.InstanceKind:
		return s.renderValueRepr(ctx, vb.valueType(ctx))
	}
	return knd.String()
}
