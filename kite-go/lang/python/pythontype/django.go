package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var djangoAddresses struct {
	Manager struct {
		Aggregate         Address
		All               Address
		Annotate          Address
		AutoCreated       Address
		BulkCreate        Address
		Check             Address
		ComplexFilter     Address
		ContributeToClass Address
		Count             Address
		Create            Address
		CreationCounter   Address
		Dates             Address
		DateTimes         Address
		DB                Address
		DBManager         Address
		Deconstruct       Address
		Defer             Address
		Distinct          Address
		Earliest          Address
		Exclude           Address
		Exists            Address
		Extra             Address
		Filter            Address
		First             Address
		FromQuerySet      Address
		Get               Address
		GetOrCreate       Address
		GetQuerySet       Address
		InBulk            Address
		Iterator          Address
		Last              Address
		Latest            Address
		None              Address
		Only              Address
		OrderBy           Address
		PrefetchRelated   Address
		Raw               Address
		Reverse           Address
		SelectForUpdate   Address
		SelectRelated     Address
		Update            Address
		UpdateOrCreate    Address
		UseInMigrations   Address
		Using             Address
		Values            Address
		ValuesList        Address
	}
	QuerySet struct {
		Aggregate               Address
		All                     Address
		Annotate                Address
		AsManager               Address
		BulkCreate              Address
		ComplexFilter           Address
		Count                   Address
		Create                  Address
		Dates                   Address
		DateTimes               Address
		DB                      Address
		Defer                   Address
		Delete                  Address
		Distinct                Address
		Earliest                Address
		Exclude                 Address
		Exists                  Address
		Extra                   Address
		Filter                  Address
		First                   Address
		Get                     Address
		GetOrCreate             Address
		InBulk                  Address
		IsCompatibleQueryObject Address
		Iterator                Address
		Last                    Address
		Latest                  Address
		None                    Address
		Only                    Address
		OrderBy                 Address
		Ordered                 Address
		PrefetchRelated         Address
		Raw                     Address
		Reverse                 Address
		SelectForUpdate         Address
		SelectRelated           Address
		Update                  Address
		UpdateOrCreate          Address
		Using                   Address
		ValueAnnotation         Address
		Values                  Address
		ValuesList              Address
	}
	Options struct {
		AddField          Address
		AddManager        Address
		CanMigrate        Address
		ContributeToClass Address
		GetAncestorLink   Address
		GetBaseChain      Address
		GetField          Address
		GetFields         Address
		GetParentList     Address
		SetupPK           Address
		SetupProxy        Address
	}
}

// Django contains values representing the members of the Django package
var Django struct {
	DB struct {
		Models struct {
			QuerySet Value
			Manager  Value
			Options  struct {
				Options Value
			}
		}
	}
	Shortcuts struct {
		GetObjectOr404 Value // django.shortcuts.get_object_or_404
		GetListOr404   Value // django.shortcuts.get_object_or_404
	}
}

// TODO(juan): add rest of members
func init() {
	optsPrefix := "django.db.models.options.Options."
	djangoAddresses.Options.AddField = SplitAddress(optsPrefix + "add_field")
	djangoAddresses.Options.AddManager = SplitAddress(optsPrefix + "add_manager")
	djangoAddresses.Options.CanMigrate = SplitAddress(optsPrefix + "can_migrate")
	djangoAddresses.Options.ContributeToClass = SplitAddress(optsPrefix + "contribute_to_class")
	djangoAddresses.Options.GetAncestorLink = SplitAddress(optsPrefix + "get_ancestor_link")
	djangoAddresses.Options.GetBaseChain = SplitAddress(optsPrefix + "get_base_chain")
	djangoAddresses.Options.GetField = SplitAddress(optsPrefix + "get_field")
	djangoAddresses.Options.GetFields = SplitAddress(optsPrefix + "get_fields")
	djangoAddresses.Options.GetParentList = SplitAddress(optsPrefix + "get_parent_list")
	djangoAddresses.Options.SetupPK = SplitAddress(optsPrefix + "setup_pk")
	djangoAddresses.Options.SetupProxy = SplitAddress(optsPrefix + "setup_proxy")
	Django.DB.Models.Options.Options = newRegType("django.db.models.options.Options", constructOptions, Builtins.Object, map[string]Value{
		"FORWARD_PROPERTIES":    nil,
		"REVERSE_PROPERTIES":    nil,
		"add_field":             nil,
		"add_manager":           nil,
		"app_config":            nil,
		"base_manager":          nil,
		"can_migrate":           nil,
		"concrete_fields":       nil,
		"contribute_to_class":   nil,
		"default_apps":          nil,
		"default_manager":       nil,
		"fields":                nil,
		"fields_map":            nil,
		"get_ancestor_link":     nil,
		"get_base_chain":        nil,
		"get_field":             nil,
		"get_fields":            nil,
		"get_parent_list":       nil,
		"installed":             nil,
		"label":                 nil,
		"label_lower":           nil,
		"local_concrete_fields": nil,
		"managers":              nil,
		"managers_map":          nil,
		"many_to_many":          nil,
		"related_objects":       nil,
		"setup_pk":              nil,
		"setup_proxy":           nil,
		"swapped":               nil,
		"verbose_name_raw":      nil,
		"virtual_fields":        nil,
	})

	qsPrefix := "django.db.models.query.QuerySet."
	djangoAddresses.QuerySet.Aggregate = SplitAddress(qsPrefix + "aggregate")
	djangoAddresses.QuerySet.All = SplitAddress(qsPrefix + "all")
	djangoAddresses.QuerySet.Annotate = SplitAddress(qsPrefix + "annotate")
	djangoAddresses.QuerySet.AsManager = SplitAddress(qsPrefix + "as_manager")
	djangoAddresses.QuerySet.BulkCreate = SplitAddress(qsPrefix + "bulk_create")
	djangoAddresses.QuerySet.ComplexFilter = SplitAddress(qsPrefix + "complex_filter")
	djangoAddresses.QuerySet.Count = SplitAddress(qsPrefix + "count")
	djangoAddresses.QuerySet.Create = SplitAddress(qsPrefix + "create")
	djangoAddresses.QuerySet.Dates = SplitAddress(qsPrefix + "dates")
	djangoAddresses.QuerySet.DateTimes = SplitAddress(qsPrefix + "datetimes")
	djangoAddresses.QuerySet.DB = SplitAddress(qsPrefix + "db")
	djangoAddresses.QuerySet.Defer = SplitAddress(qsPrefix + "defer")
	djangoAddresses.QuerySet.Delete = SplitAddress(qsPrefix + "delete")
	djangoAddresses.QuerySet.Distinct = SplitAddress(qsPrefix + "distinct")
	djangoAddresses.QuerySet.Earliest = SplitAddress(qsPrefix + "earliest")
	djangoAddresses.QuerySet.Exclude = SplitAddress(qsPrefix + "exclude")
	djangoAddresses.QuerySet.Exists = SplitAddress(qsPrefix + "exists")
	djangoAddresses.QuerySet.Extra = SplitAddress(qsPrefix + "extra")
	djangoAddresses.QuerySet.Filter = SplitAddress(qsPrefix + "filter")
	djangoAddresses.QuerySet.First = SplitAddress(qsPrefix + "first")
	djangoAddresses.QuerySet.Get = SplitAddress(qsPrefix + "get")
	djangoAddresses.QuerySet.GetOrCreate = SplitAddress(qsPrefix + "get_or_create")
	djangoAddresses.QuerySet.InBulk = SplitAddress(qsPrefix + "in_bulk")
	djangoAddresses.QuerySet.IsCompatibleQueryObject = SplitAddress(qsPrefix + "is_compatible_query_object_type")
	djangoAddresses.QuerySet.Iterator = SplitAddress(qsPrefix + "iterator")
	djangoAddresses.QuerySet.Last = SplitAddress(qsPrefix + "last")
	djangoAddresses.QuerySet.Latest = SplitAddress(qsPrefix + "latest")
	djangoAddresses.QuerySet.None = SplitAddress(qsPrefix + "none")
	djangoAddresses.QuerySet.Only = SplitAddress(qsPrefix + "only")
	djangoAddresses.QuerySet.OrderBy = SplitAddress(qsPrefix + "order_by")
	djangoAddresses.QuerySet.Ordered = SplitAddress(qsPrefix + "ordered")
	djangoAddresses.QuerySet.PrefetchRelated = SplitAddress(qsPrefix + "prefetch_related")
	djangoAddresses.QuerySet.Raw = SplitAddress(qsPrefix + "raw")
	djangoAddresses.QuerySet.Reverse = SplitAddress(qsPrefix + "reverse")
	djangoAddresses.QuerySet.SelectForUpdate = SplitAddress(qsPrefix + "select_for_update")
	djangoAddresses.QuerySet.SelectRelated = SplitAddress(qsPrefix + "select_related")
	djangoAddresses.QuerySet.Update = SplitAddress(qsPrefix + "update")
	djangoAddresses.QuerySet.UpdateOrCreate = SplitAddress(qsPrefix + "update_or_create")
	djangoAddresses.QuerySet.Using = SplitAddress(qsPrefix + "using")
	djangoAddresses.QuerySet.ValueAnnotation = SplitAddress(qsPrefix + "value_annotation")
	djangoAddresses.QuerySet.Values = SplitAddress(qsPrefix + "values")
	djangoAddresses.QuerySet.ValuesList = SplitAddress(qsPrefix + "values_list")
	Django.DB.Models.QuerySet = newRegType("django.db.models.query.QuerySet", constructQuerySet, nil, map[string]Value{
		"aggregate":                       nil,
		"all":                             nil,
		"annotate":                        nil,
		"as_manager":                      nil,
		"bulk_create":                     nil,
		"complex_filter":                  nil,
		"count":                           nil,
		"create":                          nil,
		"dates":                           nil,
		"datetime":                        nil,
		"db":                              nil,
		"defer":                           nil,
		"delete":                          nil,
		"distinct":                        nil,
		"earliest":                        nil,
		"exclude":                         nil,
		"exists":                          nil,
		"extra":                           nil,
		"filter":                          nil,
		"first":                           nil,
		"get":                             nil,
		"get_or_create":                   nil,
		"in_bulk":                         nil,
		"is_compatible_query_object_type": nil,
		"iterator":                        nil,
		"last":                            nil,
		"latest":                          nil,
		"none":                            nil,
		"only":                            nil,
		"order_by":                        nil,
		"ordered":                         nil,
		"prefetch_related":                nil,
		"raw":                             nil,
		"reverese":                        nil,
		"select_for_update":               nil,
		"select_related":                  nil,
		"update":                          nil,
		"update_or_create":                nil,
		"using":                           nil,
		"value_annotation":                nil,
		"values":                          nil,
		"values_list":                     nil,
	})

	mgrPrefix := "django.db.models.manager.Manager."
	djangoAddresses.Manager.Aggregate = SplitAddress(mgrPrefix + "aggregate")
	djangoAddresses.Manager.All = SplitAddress(mgrPrefix + "all")
	djangoAddresses.Manager.Annotate = SplitAddress(mgrPrefix + "annotate")
	djangoAddresses.Manager.AutoCreated = SplitAddress(mgrPrefix + "auto_created")
	djangoAddresses.Manager.BulkCreate = SplitAddress(mgrPrefix + "bulk_create")
	djangoAddresses.Manager.Check = SplitAddress(mgrPrefix + "check")
	djangoAddresses.Manager.ComplexFilter = SplitAddress(mgrPrefix + "complex_filter")
	djangoAddresses.Manager.ContributeToClass = SplitAddress(mgrPrefix + "contribute_to_class")
	djangoAddresses.Manager.Count = SplitAddress(mgrPrefix + "count")
	djangoAddresses.Manager.Create = SplitAddress(mgrPrefix + "create")
	djangoAddresses.Manager.CreationCounter = SplitAddress(mgrPrefix + "creation_counter")
	djangoAddresses.Manager.Dates = SplitAddress(mgrPrefix + "dates")
	djangoAddresses.Manager.DateTimes = SplitAddress(mgrPrefix + "datetimes")
	djangoAddresses.Manager.DB = SplitAddress(mgrPrefix + "db")
	djangoAddresses.Manager.DBManager = SplitAddress(mgrPrefix + "db_manager")
	djangoAddresses.Manager.Deconstruct = SplitAddress(mgrPrefix + "deconstruct")
	djangoAddresses.Manager.Defer = SplitAddress(mgrPrefix + "defer")
	djangoAddresses.Manager.Distinct = SplitAddress(mgrPrefix + "distinct")
	djangoAddresses.Manager.Earliest = SplitAddress(mgrPrefix + "earliest")
	djangoAddresses.Manager.Exclude = SplitAddress(mgrPrefix + "exclude")
	djangoAddresses.Manager.Exists = SplitAddress(mgrPrefix + "exists")
	djangoAddresses.Manager.Extra = SplitAddress(mgrPrefix + "extra")
	djangoAddresses.Manager.Filter = SplitAddress(mgrPrefix + "filter")
	djangoAddresses.Manager.First = SplitAddress(mgrPrefix + "first")
	djangoAddresses.Manager.FromQuerySet = SplitAddress(mgrPrefix + "from_queryset")
	djangoAddresses.Manager.Get = SplitAddress(mgrPrefix + "get")
	djangoAddresses.Manager.GetOrCreate = SplitAddress(mgrPrefix + "get_or_create")
	djangoAddresses.Manager.GetQuerySet = SplitAddress(mgrPrefix + "get_queryset")
	djangoAddresses.Manager.InBulk = SplitAddress(mgrPrefix + "in_bulk")
	djangoAddresses.Manager.Iterator = SplitAddress(mgrPrefix + "iterator")
	djangoAddresses.Manager.Last = SplitAddress(mgrPrefix + "last")
	djangoAddresses.Manager.Latest = SplitAddress(mgrPrefix + "latest")
	djangoAddresses.Manager.None = SplitAddress(mgrPrefix + "none")
	djangoAddresses.Manager.Only = SplitAddress(mgrPrefix + "only")
	djangoAddresses.Manager.OrderBy = SplitAddress(mgrPrefix + "order_by")
	djangoAddresses.Manager.PrefetchRelated = SplitAddress(mgrPrefix + "prefetch_related")
	djangoAddresses.Manager.Raw = SplitAddress(mgrPrefix + "raw")
	djangoAddresses.Manager.Reverse = SplitAddress(mgrPrefix + "reverse")
	djangoAddresses.Manager.SelectForUpdate = SplitAddress(mgrPrefix + "select_for_update")
	djangoAddresses.Manager.SelectRelated = SplitAddress(mgrPrefix + "select_related")
	djangoAddresses.Manager.Update = SplitAddress(mgrPrefix + "update")
	djangoAddresses.Manager.UpdateOrCreate = SplitAddress(mgrPrefix + "update_or_create")
	djangoAddresses.Manager.UseInMigrations = SplitAddress(mgrPrefix + "use_in_migrations")
	djangoAddresses.Manager.Using = SplitAddress(mgrPrefix + "using")
	djangoAddresses.Manager.Values = SplitAddress(mgrPrefix + "values")
	djangoAddresses.Manager.ValuesList = SplitAddress(mgrPrefix + "values_list")
	Django.DB.Models.Manager = newRegType("django.db.models.manager.Manager", constructManager, nil, map[string]Value{
		"aggregate":           nil,
		"all":                 nil,
		"annotate":            nil,
		"auto_created":        nil,
		"bulk_create":         nil,
		"check":               nil,
		"complex_filter":      nil,
		"contribute_to_class": nil,
		"count":               nil,
		"create":              nil,
		"creation_counter":    nil,
		"dates":               nil,
		"datetimes":           nil,
		"db":                  nil,
		"db_manager":          nil,
		"deconstruct":         nil,
		"defer":               nil,
		"distinct":            nil,
		"earliest":            nil,
		"exclude":             nil,
		"exists":              nil,
		"extra":               nil,
		"filter":              nil,
		"first":               nil,
		"from_queryset":       nil,
		"get":                 nil,
		"get_or_create":       nil,
		"in_bulk":             nil,
		"iterator":            nil,
		"last":                nil,
		"latest":              nil,
		"none":                nil,
		"only":                nil,
		"order_by":            nil,
		"prefetch_related":    nil,
		"raw":                 nil,
		"reverse":             nil,
		"select_for_update":   nil,
		"select_related":      nil,
		"update":              nil,
		"update_or_create":    nil,
		"use_in_migrations":   nil,
		"using":               nil,
		"values":              nil,
		"values_list":         nil,
	})

	Django.Shortcuts.GetListOr404 = newRegFunc("django.shortcuts.get_object_or_404", pyGetObjectOr404)
	Django.Shortcuts.GetObjectOr404 = newRegFunc("django.shortcuts.get_list_or_404", pyGetListOr404)
}

// ManagerInstance represents an instance of django.db.models.manager.Manager
type ManagerInstance struct {
	Element Value
}

// NewManager returns a new instance of django.dm.models.manager.Manager
func NewManager(query Value) Value {
	return ManagerInstance{query}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ManagerInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v ManagerInstance) Type() Value { return Django.DB.Models.Manager }

// Address gets the fully qualified path to this value in the import graph
func (v ManagerInstance) Address() Address { return Address{} }

// attr looks up an attribute on this value
// TODO(juan): add rest of members
func (v ManagerInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "aggregate":
		// NOTE(Juan): account for other aggregate methods
		fn := func(Args) Value {
			return NewDict(StrInstance{}, Union{[]Value{FloatInstance{}, IntInstance{}}})
		}
		return SingleResult(BoundMethod{djangoAddresses.Manager.Aggregate, fn}, v), nil

	case "all":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.All)
	case "annotate":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Annotate)
	case "auto_created":
		return SingleResult(BoundMethod{djangoAddresses.Manager.AutoCreated, func(Args) Value { return BoolInstance{} }}, v), nil
	case "bulk_create":
		return SingleResult(BoundMethod{djangoAddresses.Manager.BulkCreate, func(Args) Value { return NewList(v.Element) }}, v), nil
	case "check":
		return SingleResult(BoundMethod{djangoAddresses.Manager.Check, func(Args) Value { return BoolInstance{} }}, v), nil
	case "complex_filter":
		// TODO(juan): support Q objects
		return SingleResult(BoundMethod{djangoAddresses.Manager.ComplexFilter, func(Args) Value { return Builtins.None }}, v), nil
	case "contribute_to_class":
		return SingleResult(BoundMethod{djangoAddresses.Manager.ContributeToClass, func(Args) Value { return Builtins.None }}, v), nil
	case "count":
		return SingleResult(BoundMethod{djangoAddresses.Manager.Count, func(Args) Value { return IntInstance{} }}, v), nil
	case "create":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.Create)
	case "creation_counter":
		return SingleResult(IntInstance{}, v), nil
	case "dates":
		// TODO(juan): actually returns a query set over datetime.date objects!
		return SingleResult(BoundMethod{djangoAddresses.Manager.Dates, func(Args) Value { return NewList(nil) }}, v), nil
	case "datetimes":
		// TODO(juan): actually returns a query set over datetime.datetime objects
		return SingleResult(BoundMethod{djangoAddresses.Manager.DateTimes, func(Args) Value { return NewList(nil) }}, v), nil
	case "db":
		return SingleResult(StrInstance{}, v), nil
	case "db_manager":
		return SingleResult(BoundMethod{djangoAddresses.Manager.DBManager, func(Args) Value { return v }}, v), nil
	case "deconstruct":
		fn := func(Args) Value {
			return NewTuple(BoolConstant(true), StrInstance{}, StrInstance{}, NewList(nil), NewDict(nil, nil))
		}
		return SingleResult(BoundMethod{djangoAddresses.Manager.Deconstruct, fn}, v), nil
	case "defer":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Defer)
	case "distinct":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Distinct)
	case "earliest":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.Earliest)
	case "exclude":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Exclude)
	case "exists":
		return SingleResult(BoundMethod{djangoAddresses.Manager.Exists, func(Args) Value { return BoolInstance{} }}, v), nil
	case "extra":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Extra)
	case "filter":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Filter)
	case "first":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.First)
	case "from_queryset":
		// TODO(juan): does this cause circular dependency?
		return SingleResult(v, v), nil
	case "get":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.Get)
	case "get_or_create":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.GetOrCreate)
	case "get_queryset":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.GetQuerySet)
	case "in_bulk":
		fn := func(Args) Value {
			return NewDict(Union{[]Value{IntInstance{}}}, v.Element)
		}
		return SingleResult(BoundMethod{djangoAddresses.Manager.InBulk, fn}, v), nil
	case "iterator":
		return SingleResult(NewList(v.Element), v), nil
	case "last":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.Last)
	case "latest":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.Latest)
	case "none":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.None)
	case "only":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Only)
	case "order_by":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.OrderBy)
	case "prefetch_related":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.PrefetchRelated)
	case "raw":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Raw)
	case "reverse":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Reverse)
	case "select_for_update":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.SelectForUpdate)
	case "select_related":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.SelectRelated)
	case "update":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.Manager.Update)
	case "update_or_create":
		fn := func(Args) Value {
			return NewTuple(v.Element, BoolInstance{})
		}
		return SingleResult(BoundMethod{djangoAddresses.Manager.UpdateOrCreate, fn}, v), nil
	case "use_in_migrations":
		return SingleResult(BoolInstance{}, v), nil
	case "using":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.Manager.Using)
	case "values":
		fn := func(Args) Value {
			// TODO(juan): techinically this returns a dict representation of the underlying model.
			return NewQuerySet(NewDict(StrInstance{}, nil))
		}
		return SingleResult(BoundMethod{djangoAddresses.Manager.Values, fn}, v), nil
	case "values_list":
		fn := func(Args) Value {
			// TODO(juan): techinically this returns a list of tuples over the memebers of the underlying model
			return NewQuerySet(NewList(NewTuple(nil)))
		}
		return SingleResult(BoundMethod{djangoAddresses.Manager.ValuesList, fn}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

func (v ManagerInstance) pyModelBoundMethodAttrResult(addr Address) (AttrResult, error) {
	fn := func(Args) Value {
		return v.Element
	}
	return SingleResult(BoundMethod{addr, fn}, v), nil
}

func (v ManagerInstance) pyQuerySetBoundMethodAttrResult(addr Address) (AttrResult, error) {
	fn := func(Args) Value {
		return NewQuerySet(v.Element)
	}
	return SingleResult(BoundMethod{addr, fn}, v), nil
}

// equal determines whether this value is equal to another value
func (v ManagerInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ManagerInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// Flatten creates a flat version of this value
func (v ManagerInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Manager = &FlatManager{r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v ManagerInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltManager, v.Element)
}

// String provides a string representation of this value
func (v ManagerInstance) String() string {
	return fmt.Sprintf("django.db.models.Manager<%v>", v.Element)
}

func constructManager(args Args) Value {
	if len(args.Positional) < 1 {
		return NewManager(nil)
	}
	return NewManager(args.Positional[0])
}

// FlatManager is the representation of ManagerInstance used for serialization
type FlatManager struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatManager) Inflate(r *Inflater) Value {
	return NewManager(r.Inflate(f.Element))
}

// QuerySetInstance is an instance of a django.db.models.QuerySet<T> containing elements of value T
type QuerySetInstance struct {
	Element Value
}

// NewQuerySet returns a new instance of a django.db.models.QuerySet<T>
func NewQuerySet(elem Value) Value {
	return QuerySetInstance{elem}
}

// Kind categorizes this value as a function/type/module/instance/union/etc
func (v QuerySetInstance) Kind() Kind { return InstanceKind }

// Type gets the results of calling type() on this value in python
func (v QuerySetInstance) Type() Value { return Django.DB.Models.QuerySet }

// Address gets the fully qualified path to this value in the import graph
func (v QuerySetInstance) Address() Address { return Address{} }

// Elem gets the value that results from iterating over this value
func (v QuerySetInstance) Elem() Value { return v.Element }

// Index gets the value that results from indexing into this value at the specified index
func (v QuerySetInstance) Index(index Value, allowValueMutation bool) Value { return v.Element }

// SetIndex returns the QuerySetInstance that results from setting the element at the provided
// index to the provided value
func (v QuerySetInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	return NewQuerySet(value)
}

// attr looks up an attribute on this value
func (v QuerySetInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "aggregate":
		// NOTE(Juan): account for other aggregate methods
		fn := func(Args) Value {
			return NewDict(StrInstance{}, Union{[]Value{FloatInstance{}, IntInstance{}}})
		}
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.Aggregate, fn}, v), nil
	case "all":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.All)
	case "annotate":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Annotate)
	case "as_manager":
		fn := func(Args) Value {
			return NewManager(v)
		}
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.AsManager, fn}, v), nil
	case "bulk_create":
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.BulkCreate, func(Args) Value { return NewList(v.Element) }}, v), nil
	case "complex_filter":
		// TODO(juan): support Q objects
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.ComplexFilter, func(Args) Value { return Builtins.None }}, v), nil
	case "count":
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.Count, func(Args) Value { return IntInstance{} }}, v), nil
	case "create":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.Create)
	case "dates":
		// TODO(juan): actually returns a query set over datetime.date objects!
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Dates)
	case "datetimes":
		// TODO(juan): actually returns a query set over datetime.datetime objects
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.DateTimes)
	case "db":
		return SingleResult(StrInstance{}, v), nil
	case "defer":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Defer)
	case "delete":
		fn := func(Args) Value {
			return NewTuple(IntInstance{}, NewDict(StrInstance{}, IntInstance{}))
		}
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.Delete, fn}, v), nil
	case "distinct":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Distinct)
	case "earliest":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.Earliest)
	case "exclude":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Exclude)
	case "exists":
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.Exists, func(Args) Value { return BoolInstance{} }}, v), nil
	case "extra":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Extra)
	case "filter":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Filter)
	case "first":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.First)
	case "get":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.Get)
	case "get_or_create":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.GetOrCreate)
	case "in_bulk":
		fn := func(Args) Value {
			return NewDict(Union{[]Value{IntInstance{}}}, v.Element)
		}
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.InBulk, fn}, v), nil
	case "is_compatible_query_object_type":
		return SingleResult(BoolInstance{}, v), nil
	case "iterator":
		return SingleResult(NewList(v.Element), v), nil
	case "last":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.Last)
	case "latest":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.Latest)
	case "none":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.None)
	case "only":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Only)
	case "order_by":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.OrderBy)
	case "prefetch_related":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.PrefetchRelated)
	case "raw":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Raw)
	case "reverse":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Reverse)
	case "select_for_update":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.SelectForUpdate)
	case "select_related":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.SelectRelated)
	case "update":
		return v.pyModelBoundMethodAttrResult(djangoAddresses.QuerySet.Update)
	case "update_or_create":
		fn := func(Args) Value {
			return NewTuple(v.Element, BoolInstance{})
		}
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.UpdateOrCreate, fn}, v), nil
	case "using":
		return v.pyQuerySetBoundMethodAttrResult(djangoAddresses.QuerySet.Using)
	case "value_annotation":
		return SingleResult(BoolInstance{}, v), nil
	case "values":
		fn := func(Args) Value {
			// TODO(juan): techinically this returns a dict representation of the underlying model.
			return NewQuerySet(NewDict(StrInstance{}, nil))
		}
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.Values, fn}, v), nil
	case "values_list":
		fn := func(Args) Value {
			// TODO(juan): techinically this returns a list of tuples over the memebers of the underlying model
			return NewQuerySet(NewList(NewTuple(nil)))
		}
		return SingleResult(BoundMethod{djangoAddresses.QuerySet.ValuesList, fn}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

func (v QuerySetInstance) pyModelBoundMethodAttrResult(addr Address) (AttrResult, error) {
	fn := func(Args) Value {
		return v.Element
	}
	return SingleResult(BoundMethod{addr, fn}, v), nil
}

func (v QuerySetInstance) pyQuerySetBoundMethodAttrResult(addr Address) (AttrResult, error) {
	fn := func(Args) Value {
		return v
	}
	return SingleResult(BoundMethod{addr, fn}, v), nil
}

// equal determines wether two values are equal
func (v QuerySetInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(QuerySetInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// Flatten creates a flat version of this Value
func (v QuerySetInstance) Flatten(f *FlatValue, r *Flattener) {
	f.QuerySet = &FlatQuerySet{r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v QuerySetInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltQuerySet, v.Element)
}

// String provides a string representation of this value
func (v QuerySetInstance) String() string {
	return fmt.Sprintf("django.db.models.QuerySet<%v>", v.Element)
}

func constructQuerySet(args Args) Value {
	if len(args.Positional) < 1 {
		if model, ok := args.Keyword("model"); ok {
			return NewQuerySet(model)
		}
		return nil
	}
	return NewQuerySet(args.Positional[0])
}

// FlatQuerySet is the representation of QuerySet used for serialization
type FlatQuerySet struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatQuerySet) Inflate(r *Inflater) Value {
	return NewQuerySet(r.Inflate(f.Element))
}

// OptionsInstance represents an instance of django.db.models.options.Options
type OptionsInstance struct {
	Model Value
}

// NewOptions returns a new instance of django.db.models.options.Options
// It is okay for model to be nil (this happens when we encounter an import graph
// node of type django.db.models.options.Options)
func NewOptions(model Value) Value {
	return OptionsInstance{model}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v OptionsInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v OptionsInstance) Type() Value { return Django.DB.Models.Options.Options }

// Address gets the fully qualified path to this value in the import graph
func (v OptionsInstance) Address() Address { return Address{} }

// attr looks up an attribute on this value
// TODO(juan): add rest of memebers
func (v OptionsInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "FORAWARD_PROPERTIES", "REVERSE_PROPERTIES":
		return SingleResult(NewSet(StrInstance{}), v), nil
	case "abstract":
		return SingleResult(BoolInstance{}, v), nil
	case "add_field":
		return SingleResult(BoundMethod{djangoAddresses.Options.AddField, func(Args) Value { return Builtins.None }}, v), nil
	case "add_manager":
		return SingleResult(BoundMethod{djangoAddresses.Options.AddManager, func(Args) Value { return Builtins.None }}, v), nil
	case "app_config":
		// TODO(juan): we should return a member of the apps module
		return SingleResult(Builtins.None, v), nil
	case "app_label":
		return SingleResult(StrInstance{}, v), nil
	case "apps":
		// TODO(juan): we should return an ExternalInstance here
		return SingleResult(Builtins.None, v), nil
	case "auto_created":
		return SingleResult(BoolInstance{}, v), nil
	case "auto_field":
		// TODO(juan): we should return an ExternalInstance here
		return SingleResult(Builtins.None, v), nil
	case "base_manager":
		// TODO(juan): we should return an ExternalInstance here
		return SingleResult(Builtins.None, v), nil
	case "base_manager_name":
		return SingleResult(StrInstance{}, v), nil
	case "can_migrate":
		return SingleResult(BoundMethod{djangoAddresses.Options.CanMigrate, func(Args) Value { return BoolInstance{} }}, v), nil
	case "concrete_fields":
		// TODO(juan): correct semantics here by looking at model attributes
		return SingleResult(NewTuple(Builtins.None), v), nil
	case "concrete_model":
		return SingleResult(v.Model, v), nil
	case "contribute_to_class":
		return SingleResult(BoundMethod{djangoAddresses.Options.ContributeToClass, func(Args) Value { return Builtins.None }}, v), nil
	case "db_table":
		return SingleResult(StrInstance{}, v), nil
	case "db_tabelspace":
		return SingleResult(StrInstance{}, v), nil
	case "default_apps":
		// TODO(juan): we should return an ExternalInstance here
		return SingleResult(Builtins.None, v), nil
	case "default_manager":
		return SingleResult(NewManager(v.Model), v), nil
	case "default_manager_name":
		return SingleResult(StrInstance{}, v), nil
	case "default_permissions":
		return SingleResult(NewTuple(StrConstant("add"), StrConstant("change"), StrConstant("delete")), v), nil
	case "default_related_name":
		return SingleResult(StrInstance{}, v), nil
	case "fields":
		// TODO(juan): get fields from model
		return SingleResult(NewTuple(nil), v), nil
	case "fields_map":
		// TODO(juan): get fields from model
		return SingleResult(NewDict(StrInstance{}, nil), v), nil
	case "get_ancetor_link":
		// TODO(juan): could add more semantics here, also could return an ExternalInstance
		return SingleResult(BoundMethod{djangoAddresses.Options.GetAncestorLink, func(Args) Value { return Builtins.None }}, v), nil
	case "get_base_chain":
		// TODO(juan): could add more semantics here
		return SingleResult(BoundMethod{djangoAddresses.Options.GetBaseChain, func(Args) Value { return NewList(nil) }}, v), nil
	case "get_field":
		fn := func(args Args) Value {
			var fieldname string
			switch {
			case len(args.Positional) > 0 && len(args.Keywords) > 0:
				return Builtins.None
			case len(args.Positional) == 1:
				if str, ok := args.Positional[0].(StrConstant); ok {
					fieldname = string(str)
				}
			case len(args.Keywords) == 1:
				if str, ok := args.Keywords[0].Value.(StrConstant); ok {
					fieldname = string(str)
				}
			}
			if fieldname != "" && v.Model != nil {
				res, err := Attr(kitectx.TODO(), v.Model, fieldname)
				if err == nil && res.Value() != nil {
					return res.Value().Type()
				}
			}
			return Builtins.None
		}
		return SingleResult(BoundMethod{djangoAddresses.Options.GetField, fn}, v), nil
	case "get_fields":
		// TODO(juan): could add more semantics here
		return SingleResult(BoundMethod{djangoAddresses.Options.GetFields, func(Args) Value { return Builtins.None }}, v), nil
	case "get_latest_by":
		return SingleResult(Builtins.None, v), nil
	case "get_parent_list":
		return SingleResult(BoundMethod{djangoAddresses.Options.GetParentList, func(Args) Value { return Builtins.None }}, v), nil
	case "has_auto_field":
		return SingleResult(BoolInstance{}, v), nil
	case "index_together":
		return SingleResult(NewList(nil), v), nil
	case "installed":
		return SingleResult(BoolInstance{}, v), nil
	case "label", "label_lower":
		return SingleResult(StrInstance{}, v), nil
	case "local_concrete_fields":
		// TODO(juan): could add more semantics here
		return SingleResult(NewTuple(nil), v), nil
	case "local_fields":
		return SingleResult(NewList(nil), v), nil
	case "local_managers":
		return SingleResult(NewList(NewManager(v.Model)), v), nil
	case "local_many_to_many":
		return SingleResult(NewList(nil), v), nil
	case "managed":
		return SingleResult(BoolInstance{}, v), nil
	case "manager_inheritance_from_future":
		return SingleResult(BoolInstance{}, v), nil
	case "managers":
		return SingleResult(NewTuple(NewManager(v.Model)), v), nil
	case "managers_map":
		return SingleResult(NewDict(StrInstance{}, NewManager(v.Model)), v), nil
	case "many_to_many":
		// TODO(juan): could add more semantics here
		return SingleResult(NewTuple(nil), v), nil
	case "model":
		return SingleResult(v.Model, v), nil
	case "model_name":
		return SingleResult(StrInstance{}, v), nil
	case "object_name":
		return SingleResult(StrInstance{}, v), nil
	case "order_with_respect_to":
		return SingleResult(Builtins.None, v), nil
	case "ordering":
		return SingleResult(NewList(nil), v), nil
	case "original_attrs":
		return SingleResult(NewDict(StrInstance{}, nil), v), nil
	case "parents":
		return SingleResult(NewOrderedDict(StrInstance{}, nil), nil), nil
	case "permissions":
		return SingleResult(NewList(nil), v), nil
	case "pk":
		// TODO(juan): this should return an External<django.db.models.fields.AutoField>
		return SingleResult(Union{[]Value{IntInstance{}}}, v), nil
	case "private_fields":
		return SingleResult(NewList(nil), v), nil
	case "proxy":
		return SingleResult(BoolInstance{}, v), nil
	case "proxy_for_model":
		return SingleResult(Builtins.None, v), nil
	case "related_fkey_lookups":
		return SingleResult(NewList(nil), v), nil
	case "related_objects":
		// TODO(juan): could add more semantics here
		return SingleResult(NewTuple(nil), v), nil
	case "required_db_features":
		return SingleResult(NewList(nil), v), nil
	case "required_db_vendor":
		return SingleResult(Builtins.None, v), nil
	case "select_on_save":
		return SingleResult(BoolInstance{}, v), nil
	case "setup_pk":
		return SingleResult(BoundMethod{djangoAddresses.Options.SetupPK, func(Args) Value { return Builtins.None }}, v), nil
	case "setup_proxy":
		return SingleResult(BoundMethod{djangoAddresses.Options.SetupProxy, func(Args) Value { return Builtins.None }}, v), nil
	case "swappable":
		return SingleResult(Union{[]Value{Builtins.None, BoolInstance{}}}, v), nil
	case "swapped":
		return SingleResult(Union{[]Value{Builtins.None, BoolInstance{}}}, v), nil
	case "unique_together":
		return SingleResult(NewList(nil), v), nil
	case "verbose_name":
		return SingleResult(StrInstance{}, v), nil
	case "verbose_name_plural":
		return SingleResult(StrInstance{}, v), nil
	case "verbose_name_raw":
		return SingleResult(StrInstance{}, v), nil
	case "virtual_fields":
		return SingleResult(NewList(nil), v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines if two values are equal
func (v OptionsInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(OptionsInstance); ok {
		return equal(ctx, v.Model, u.Model)
	}
	return false
}

// Flatten creates a flat version of this value
func (v OptionsInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Options = &FlatOptions{r.Flatten(v.Model)}
}

// hash gets a unique ID for this value
func (v OptionsInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltOptions, v.Model)
}

// String provides a string representation of this value
func (v OptionsInstance) String() string {
	return fmt.Sprintf("django.db.models.options.Options<%v>", v.Model)
}

func constructOptions(args Args) Value {
	if len(args.Positional) < 1 {
		return NewOptions(nil)
	}
	return NewOptions(args.Positional[0])
}

// FlatOptions is the representation of OptionsInstance used for serialization
type FlatOptions struct {
	Model FlatID
}

// Inflate creates a value from a flat value
func (f FlatOptions) Inflate(r *Inflater) Value {
	return NewOptions(r.Inflate(f.Model))
}

// pyGetObjectOr404 returns an instance of the type passed as first parameter.
// pyGetObjectOr404(M) = M()
func pyGetObjectOr404(args Args) Value {
	if len(args.Positional) == 0 {
		return nil
	}
	var out []Value
	for _, v := range Disjuncts(kitectx.TODO(), args.Positional[0]) {
		if ctor, ok := v.(Callable); ok && v.Kind() == TypeKind {
			out = append(out, ctor.Call(Args{}))
		}
	}
	return Unite(kitectx.TODO(), out...)
}

// pyGetListOr404 returns a list of instances of the type passed as first parameter.
// pyGetListOr404(M) = [M(), ...]
func pyGetListOr404(args Args) Value {
	if len(args.Positional) == 0 {
		return nil
	}
	var out []Value
	for _, v := range Disjuncts(kitectx.TODO(), args.Positional[0]) {
		if ctor, ok := v.(Callable); ok && v.Kind() == TypeKind {
			out = append(out, ctor.Call(Args{}))
		}
	}
	return NewList(Unite(kitectx.TODO(), out...))
}
