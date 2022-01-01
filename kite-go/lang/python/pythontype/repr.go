package pythontype

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Reprs returns a list of string representations of the disjuncts of `val`. It may be empty if a repr could not be computed.
func Reprs(ctx kitectx.Context, val Value, rm pythonresource.Manager, last bool) []string {
	ctx.CheckAbort()

	if val == nil {
		return nil
	}

	var reprs []string
	ctx.WithCallLimit(100, func(ctx kitectx.CallContext) error {
		val = translate(ctx, val, rm)
		reprs = reprsImpl(ctx, val, last)
		return nil
	})
	for i := range reprs {
		if strings.HasPrefix(reprs[i], "builtins.") {
			reprs[i] = strings.TrimPrefix(reprs[i], "builtins.")
		}
	}
	return reprs
}

func reprsImpl(ctx kitectx.CallContext, val Value, last bool) []string {
	ctx.CheckAbort()

	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case ExternalRoot:
		return nil
	case External:
		// for externals, return the canonical name, or an equivalent path if the
		// canonical name is empty
		if last {
			return []string{val.Symbol().PathLast()}
		}
		return []string{val.Symbol().PathString()}
	case ExternalInstance:
		// for external instances, return the type name, or an equivalent path if the
		// type name if it is empty
		if last {
			return []string{val.TypeExternal.Symbol().PathLast()}
		}
		return []string{val.TypeExternal.Symbol().PathString()}
	case Union:
		// for unions, apply this function to each disjunct, and uniquify
		strSet := make(map[string]struct{})
		for _, disjunct := range Disjuncts(ctx.Context, val) {
			for _, repr := range reprsImpl(ctx.Call(), disjunct, last) {
				strSet[repr] = struct{}{}
			}
		}
		var strs []string
		for str := range strSet {
			strs = append(strs, str)
		}
		sort.Strings(strs)
		return strs
	case StrConstant:
		return []string{fmt.Sprintf("\"%s\"", string(val))}
	case IntConstant:
		return []string{strconv.FormatInt(int64(val), 10)}
	case FloatConstant:
		return []string{strconv.FormatFloat(float64(val), 'f', -1, 64)}
	case ComplexConstant:
		return []string{fmt.Sprintf("%v", val)}
	case BoolConstant:
		if bool(val) {
			return []string{"True"}
		}
		return []string{"False"}
	case NoneConstant:
		return []string{"None"}
	}

	// for non-external and non-union nodes, break it down by kind
	switch val.Kind() {
	case ModuleKind:
		if mod, ok := val.(*SourceModule); ok {
			// Source modules have no path
			return []string{strings.TrimSuffix(path.Base(mod.Address().File), ".py")}
		}
		if mod, ok := val.(*SourcePackage); ok {
			// Source modules have no path
			return []string{strings.TrimSuffix(path.Base(mod.Address().File), ".py")}
		}
	case InstanceKind:
		// for instances, return the type
		return reprsImpl(ctx.Call(), val.Type(), last)
	default:
		// for types and functions, return the path within the module
		// or empty string if the address is Nil
		if last {
			return []string{val.Address().Path.Last()}
		}
		return []string{val.Address().Path.String()}
	}
	return nil
}
