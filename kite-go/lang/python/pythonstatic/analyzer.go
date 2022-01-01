package pythonstatic

import (
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Analyzer provides an exported interface to the propagator
type Analyzer struct {
	*helpers
}

// NewAnalyzer creates a new propagator in the given scope
func NewAnalyzer(
	rm pythonresource.Manager,
	assembly *Assembly,
	delegate PropagatorDelegate,
	trace io.Writer) *Analyzer {
	return &Analyzer{
		helpers: &helpers{
			ResourceManager: rm,
			Assembly:        assembly,
			Delegate:        delegate,
			TraceWriter:     trace,
		},
	}
}

// Module propagates a module
func (a *Analyzer) Module(ctx kitectx.Context, module *pythontype.SourceModule, node *pythonast.Module, imp Importer) {
	ctx.CheckAbort()

	p := newPropagator(ctx, module.Members, imp, module, a.helpers)
	p.propagate(node.Body)
	p.discard()
}

// Function propagates a function
func (a *Analyzer) Function(ctx kitectx.Context, fun *pythontype.SourceFunction, node *pythonast.FunctionDefStmt, imp Importer) {
	ctx.CheckAbort()

	// Here we propose types for the "self" or "cls" parameter of functions. We
	// propose the class itself as well as each of its subclasses (including
	// subclasses of subclasses, etc). This mirrors the way that the runtime
	// type of "self" in python is the derived class not the base class when you
	// call self.foo() on a derived class where "foo" was declared in the base
	// class. This is important because it is common to access derived class
	// members from base classes via "self". This does not really belong in
	// the propagator because it does not logically depend on a scope. But it
	// must happen each time we propagate a function because new base classes
	// may have been defined since last time we processed this function.
	if fun.Class != nil && len(fun.Parameters) > 0 && (fun.HasReceiver || fun.HasClassReceiver) {
		var receivers []pythontype.Value
		seen := make(map[*pythontype.SourceClass]bool)
		walkSubclasses(ctx, fun.Class, func(subclass *pythontype.SourceClass) bool {
			if seen[subclass] {
				return false
			}
			seen[subclass] = true
			if fun.HasClassReceiver {
				// for class receivers the values are direct class references
				receivers = append(receivers, subclass)
			} else {
				// for regular receivers the values are instances
				receivers = append(receivers, pythontype.SourceInstance{Class: subclass})
			}
			return true
		})
		sym := fun.Parameters[0].Symbol
		sym.Value = pythontype.Unite(ctx, append(receivers, sym.Value)...)
	}

	p := newPropagator(ctx, fun.Locals, imp, fun.Module, a.helpers)
	p.Function = fun

	// Notify delegate of all current parameter values
	for i, param := range fun.Parameters {
		if i < len(node.Parameters) {
			// must assign here because paramaters can be destructured
			p.assignExpr(node.Parameters[i].Name, param.Symbol.Value)
		}
	}
	if a.Delegate != nil {
		if node.Vararg != nil {
			if fun.Vararg == nil {
				a.Delegate.Resolved(node.Vararg.Name, pythontype.NewList(nil))
			} else {
				a.Delegate.Resolved(node.Vararg.Name, fun.Vararg.Symbol.Value)
			}
		}
		if node.Kwarg != nil {
			if fun.KwargDict == nil {
				a.Delegate.Resolved(node.Kwarg.Name, pythontype.NewDict(pythontype.StrInstance{}, nil))
			} else {
				a.Delegate.Resolved(node.Kwarg.Name, fun.KwargDict)
			}
		}
	}

	p.propagate(node.Body)
	p.discard()
}

// Lambda propagates the body of a lambda
func (a *Analyzer) Lambda(ctx kitectx.Context, lambda *pythontype.SourceFunction, node *pythonast.LambdaExpr, imp Importer) {
	ctx.CheckAbort()

	p := newPropagator(ctx, lambda.Locals, imp, lambda.Module, a.helpers)
	p.Function = lambda

	// Notify delegate of all current parameter values
	if a.Delegate != nil {
		for i, param := range lambda.Parameters {
			if i < len(node.Parameters) {
				a.Delegate.Resolved(node.Parameters[i].Name, param.Symbol.Value)
			}
		}
		if node.Vararg != nil && lambda.Vararg != nil {
			a.Delegate.Resolved(node.Vararg.Name, lambda.Vararg.Symbol.Value)
		}
		if node.Kwarg != nil && lambda.Kwarg != nil {
			a.Delegate.Resolved(node.Kwarg.Name, lambda.Kwarg.Symbol.Value)
		}
	}

	p.propagateLambdaBody(node.Body)
	p.discard()
}

// CreateModule makes module containing default members (__file, __name__, etc)
func CreateModule(name pythontype.Address, parent *pythontype.SymbolTable) *pythontype.SourceModule {
	mod := &pythontype.SourceModule{Members: pythontype.NewSymbolTable(name, parent)}
	mod.Members.Put("__file__", pythontype.StrInstance{})
	mod.Members.Put("__name__", pythontype.StrInstance{})
	mod.Members.Put("__doc__", pythontype.StrInstance{})
	mod.Members.Put("__package__", pythontype.StrInstance{})
	mod.Members.Put("__builtins__", pythontype.BuiltinModule)
	return mod
}
