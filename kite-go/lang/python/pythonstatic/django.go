package pythonstatic

import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	djangoModelAddress         = pythontype.SplitAddress("django.db.models.Model")
	djangoRequestAddress       = pythontype.SplitAddress("django.http.request.HttpRequest")
	djangoGetModelAddress      = pythontype.SplitAddress("django.db.models.get_model") // TODO(juan): this is deprecated in django 1.10
	djangoDateFieldAddress     = pythontype.SplitAddress("django.db.models.fields.DateField")
	djangoDateTimeFieldAddress = pythontype.SplitAddress("django.db.models.fields.DateTimeField")
)

func init() {
	registerMetaClass(djangoModelMetaClass{})
}

type djangoModelMetaClass struct{}

func (d djangoModelMetaClass) IsMetaClass(c *pythontype.SourceClass) bool {
	for _, base := range c.Bases {
		if base.Address().Path.Hash == djangoModelAddress.Path.Hash {
			return true
		}
	}
	return false
}

func (d djangoModelMetaClass) Construct(ctx kitectx.Context, c *pythontype.SourceClass, symbols *pythontype.SymbolTable) {
	ctx.CheckAbort()

	// add django members
	s := symbols.LocalOrCreate("id")
	s.Value = pythontype.Unite(ctx, s.Value, pythontype.IntInstance{})

	s = symbols.LocalOrCreate("pk")
	s.Value = pythontype.Unite(ctx, s.Value, pythontype.IntInstance{})

	s = symbols.LocalOrCreate("objects")
	s.Value = pythontype.Unite(ctx, s.Value, pythontype.NewManager(c))

	s = symbols.LocalOrCreate("_meta")
	s.Value = pythontype.Unite(ctx, s.Value, pythontype.NewOptions(c))

	// merge other members, handle get_next_by_FOO, get_previous_by_FOO
	for name, symbol := range symbols.Table {
		member := c.Members.LocalOrCreate(name)
		member.Value = pythontype.Unite(ctx, member.Value, symbol.Value)
		if !symbol.Name.Equals(member.Name) {
			panic("different address for symbols with the same name")
		}

		// handle get_next_by_FOO, get_previous_by_FOO methods:
		// for each `DateField` or `DateTimeField` field on the model the django framework
		// adds a `get_next_by_FOO`` and a `get_previous_by_FOO` method where
		// `FOO` is the name of the field. These methods return the next/previous
		// instance of the model ordered by the specified `DateField`/`DateTimeField`.
		// SEE: https://docs.djangoproject.com/en/1.10/ref/models/instances/#extra-instance-methods
		for _, v := range pythontype.Disjuncts(ctx, member.Value) {
			ext, ok := v.(pythontype.ExternalInstance)
			if !ok {
				continue
			}

			hash := ext.TypeExternal.Symbol().PathHash()
			if hash != djangoDateFieldAddress.Path.Hash && hash != djangoDateTimeFieldAddress.Path.Hash {
				continue
			}

			mn := "get_previous_by_" + name
			s := c.Members.LocalOrCreate(mn)
			addr := c.Address().WithTail(mn)
			s.Value = pythontype.Unite(ctx, member.Value, pythontype.BoundMethod{
				Addr: addr,
				F:    func(pythontype.Args) pythontype.Value { return pythontype.SourceInstance{Class: c} },
			})

			mn = "get_next_by_" + name
			s = c.Members.LocalOrCreate(mn)
			addr = c.Address().WithTail(mn)
			s.Value = pythontype.Unite(ctx, member.Value, pythontype.BoundMethod{
				Addr: addr,
				F:    func(pythontype.Args) pythontype.Value { return pythontype.SourceInstance{Class: c} },
			})
		}
	}
}

func updateSymbolWithDjangoRequest(ctx kitectx.Context, s *pythontype.Symbol, importer Importer) {
	ctx.CheckAbort()

	// get django request node
	sym, err := importer.Navigate(djangoRequestAddress.Path)
	if err != nil {
		return
	}
	// update symbols value
	s.Value = pythontype.Unite(ctx, s.Value, pythontype.TranslateExternalInstance(sym, importer.Global))
}

func updateSymbolWithDjangoQuerySet(ctx kitectx.Context, s *pythontype.Symbol, f *pythontype.SourceFunction) {
	ctx.CheckAbort()

	if len(f.Parameters) < 1 {
		panic("updating a function with django queryset with fewer than 2 parameters")
	}

	// first parameter should be an instance of a django model
	model := f.Parameters[0].Symbol.Value
	s.Value = pythontype.Unite(ctx, s.Value, pythontype.NewQuerySet(model))
}

// djangoViewRequestParam is a heuristic for selecting the paramter to a django view function
// that is an instance of django.http.HttpRequest
// SEE: https://docs.djangoproject.com/en/1.10/topics/http/views/
func djangoViewRequestParam(f *pythontype.SourceFunction) pythontype.Parameter {
	if len(f.Parameters) > 0 {
		if (f.HasReceiver || f.HasClassReceiver) && len(f.Parameters) > 1 {
			return f.Parameters[1]
		}
		return f.Parameters[0]
	}
	return pythontype.Parameter{}
}

// djangoAdminRequestParam is a heuristic for selecting the parameter to a django admin action function
// that is an instance of django.http.HttpRequest
// SEE: https://docs.djangoproject.com/en/dev/ref/contrib/admin/actions/#writing-action-functions
func djangoAdminRequestParam(f *pythontype.SourceFunction) pythontype.Parameter {
	if len(f.Parameters) > 2 {
		return f.Parameters[1]
	}
	return pythontype.Parameter{}
}

// djangoAdminQuerySetParam is a heuristic for selecting the parameter to a django admin action function
// that is an instance of django.db.models.QuerySet
// SEE: https://docs.djangoproject.com/en/dev/ref/contrib/admin/actions/#writing-action-functions
func djangoAdminQuerySetParam(f *pythontype.SourceFunction) pythontype.Parameter {
	if len(f.Parameters) > 2 {
		return f.Parameters[2]
	}
	return pythontype.Parameter{}
}

// isDjangoQuerySetParam is a heuristic for determining whether a parameter represents
// an instance of django.db.models.QuerySet
func isDjangoQuerySetParam(param pythontype.Parameter) bool {
	switch strings.ToLower(param.Name) {
	case "queryset", "query", "qs", "qset":
		return true
	default:
		return false
	}
}

// isDjangoRequestParam is a heuristic for determining whether a parameter
// represents an instance of django.http.HttpRequest
func isDjangoRequestParam(param pythontype.Parameter) bool {
	switch strings.ToLower(param.Name) {
	case "request", "req", "r":
		return true
	default:
		return false
	}
}

// isDjangoView is a heuristic for determining if a module represents a
// django view module, will also catch cases such as "special_views.py".
func isDjangoView(mod *pythontype.SourceModule) bool {
	if !strings.HasSuffix(mod.Members.Name.File, "views.py") {
		return false
	}
	return importsDjango(mod)
}

// isDjangoAdmin is a heuristic for determining if a module represents a
// django admin module, will also catch cases such as "special_admin.py".
func isDjangoAdmin(mod *pythontype.SourceModule) bool {
	if !strings.HasSuffix(mod.Members.Name.File, "admin.py") {
		return false
	}
	return importsDjango(mod)
}

func importsDjango(mod *pythontype.SourceModule) bool {
	for _, symb := range mod.Members.Table {
		if symb == nil || symb.Value == nil {
			continue
		}
		addr := symb.Value.Address()
		if !addr.Path.Empty() && addr.Path.Parts[0] == "django" {
			return true
		}
	}
	return false
}

func isDjangoGetModelCall(callee pythontype.Value) bool {
	external, isExt := callee.(pythontype.External)
	if !isExt {
		return false
	}

	return external.Symbol().PathHash() == djangoGetModelAddress.Path.Hash
}

func doDjangoGetModelHeuristic(ctx kitectx.Context, args pythontype.Args, prop *propagator) []pythontype.Value {
	ctx.CheckAbort()

	if len(args.Positional) < 2 {
		return nil
	}

	pkgName, isStr := args.Positional[0].(pythontype.StrConstant)
	if !isStr {
		return nil
	}

	modelName, isStr := args.Positional[1].(pythontype.StrConstant)
	if !isStr {
		return nil
	}

	p, found := prop.Importer.ImportAbs(ctx, string(pkgName))
	if !found {
		return nil
	}

	pkg, isPkg := p.(*pythontype.SourcePackage)
	if !isPkg {
		return nil
	}

	for _, entry := range pkg.DirEntries.Table {
		if entry == nil || entry.Value == nil {
			continue
		}

		res, _ := pythontype.Attr(ctx, entry.Value, string(modelName))
		if val := res.Value(); val != nil {
			return []pythontype.Value{val}
		}
	}

	return nil
}
