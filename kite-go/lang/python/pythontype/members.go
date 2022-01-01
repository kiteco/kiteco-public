package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// Members (very roughly) approximates Python's builtin dir function, producing a map of member attributes to values
func Members(ctx kitectx.Context, rm pythonresource.Manager, val Value) map[string]Value {
	val = Translate(ctx, val, rm)
	if val == nil {
		return nil
	}
	members := make(map[string]Value)

	// TODO note that we do a very rough, incomplete appoximation of MRO here;
	// it's open work to figure out a clean abstraction for MRO behavior that we
	// can reuse wherever necessary

	// First get type members (if available)
	typ := val.Type()
	if typ != nil {
		typeVal := Translate(ctx, typ, rm)
		ctx.WithCallLimit(100, func(ctx kitectx.CallContext) error {
			addValueMembers(ctx, rm, typeVal, members)
			return nil
		})
	}

	ctx.WithCallLimit(100, func(ctx kitectx.CallContext) error {
		addValueMembers(ctx, rm, val, members)
		return nil
	})

	return members
}

func addValueMembers(ctx kitectx.CallContext, rm pythonresource.Manager, val Value, members map[string]Value) {
	if ctx.AtCallLimit() {
		return
	}

	switch v := val.(type) {

	// Global
	case ExternalRoot:
		for _, pkg := range rm.Pkgs() {
			if sym, err := rm.PathSymbol(pythonimports.NewPath(pkg)); err == nil {
				member := NewExternal(sym, rm)
				members[pkg] = Unite(ctx.Context, members[pkg], member)
			}
		}
	case External:
		sym := v.Symbol()
		for _, base := range rm.Bases(sym) {
			addValueMembers(ctx.Call(), rm, NewExternal(base, rm), members)
		}

		attrs, _ := rm.Children(sym)
		for _, attr := range attrs {
			if membersSym, err := rm.ChildSymbol(sym, attr); err == nil {
				member := NewExternal(membersSym, rm)
				members[attr] = Unite(ctx.Context, members[attr], member)
			}
		}
	case ExternalInstance:
		// nothing to do, as as Instance cannot be the type of any Value,
		// and the type is already handled in `completions` above

	case ExternalReturnValue:
		// nothing to do here

	// Source
	case *SourceClass:
		for _, base := range v.Bases {
			base = Translate(ctx.Context, base, rm)
			if base == nil {
				continue
			}
			addValueMembers(ctx.Call(), rm, base, members)
		}

		for attr, member := range v.Members.Table {
			members[attr] = Unite(ctx.Context, members[attr], member.Value)
		}
	case *SourceModule:
		for attr, member := range v.Members.Table {
			members[attr] = Unite(ctx.Context, members[attr], member.Value)
		}
	case *SourcePackage:
		for attr, member := range v.DirEntries.Table {
			members[attr] = Unite(ctx.Context, members[attr], member.Value)
		}

		if v.Init != nil {
			addValueMembers(ctx.Call(), rm, v.Init, members)
		}
	case *SourceFunction:
		// pretend functions don't have members
	case SourceInstance:
		// nothing to do, as as Instance cannot be the type of any Value,
		// and the type is already handled in `completions` above
	case PropertyInstance:
		// ignore; unclear how to handle this for now TODO(naman)

	// Constant
	case ConstantValue:
		// pretend constants don't have members

	// Union
	case Union:
		for _, v := range v.Constituents {
			addValueMembers(ctx.Call(), rm, v, members)
		}

	default:
		rollbar.Error(fmt.Errorf("addValueMembers got unexpected type"), fmt.Sprintf("%T", v))
	}
}
