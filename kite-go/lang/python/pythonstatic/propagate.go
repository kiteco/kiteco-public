package pythonstatic

import (
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// typeInductionThreshold is the theshold probability for a type returned from
// type induction to be included in the union type for a function return value.
const typeInductionThreshold = 0.1

// A Production is one or more types that were assigned to a symbol
type Production struct {
	Symbol *pythontype.Symbol
	Value  pythontype.Value
}

// PropagatorDelegate contains callbacks for the propagator
type PropagatorDelegate interface {
	// Pass is called at the start of each propagation pass with the current pass (0-indexed) and the total number of passes
	Pass(current, total int)

	// Resolved is called after evaluating each AST expression. It will be
	// called even if the expression failed to evaluate, in which case value
	// will be nil. This will be called for every node in the AST that implements pythonast.Expr.
	Resolved(expr pythonast.Expr, value pythontype.Value)
}

// helpers contains data structures passed recursively to all propagators.
type helpers struct {
	ResourceManager pythonresource.Manager

	// Assembly contains the functions, classes, and modules that we have
	// extracted so far.
	Assembly *Assembly

	// Delegate contains callbacks for the propagator
	Delegate PropagatorDelegate

	// TraceWriter is the writer to which debug messages are sent, or nil to
	// turn them off.
	TraceWriter io.Writer

	// Opts are the options specified when constructing the Assembler
	Opts Options

	// CapabilityDelegate is the delegate for tracking Capabilities
	CapabilityDelegate *capabilityDelegate
}

// A propagator uses statements from an AST to propagate symbols from an
// input set to an output set
type propagator struct {
	// ctx is stored in the propagator (against guidelines) for convenience:
	// we must manually ensure it is checked frequently; we check it in evaluate and propagateStmt
	ctx kitectx.Context

	// Module that contains whatever code we are currently processing. This
	// is only used to populate the __module__ attribute of classes.
	Module *pythontype.SourceModule

	// Return is a symbol representing the return type from the current
	// function, or nil if we are not processing a function
	Return *pythontype.Symbol

	// Class is the current class if we are processing the immediate body
	// of a class, or nil if we are not processing a class.
	Class *pythontype.SourceClass

	// Function is the current function if we are processing the immediate body
	// of a function, or nil if we are not processing a function.
	Function *pythontype.SourceFunction

	// Scope is the symbol table in which name expressions will be evaluated
	// and into which new symbols will be inserted
	Scope *pythontype.SymbolTable

	// InheritedScope is the symbol table that will become the parent of any
	// descendent symbol tables created by this propagator. When processing a
	// function or module, InheritedScope = Scope. When processing a class
	// it is the symbol table of the most recent module or function.
	InheritedScope *pythontype.SymbolTable

	// Importer is responsible for resolving imports to modules (for imports
	// into both user files and the global graph).
	Importer Importer

	// helpers contains helpers contains data structures passed recursively
	// to all propagators
	h *helpers

	// comprehensionCounter is used to name the comprehension sub-scopes
	comprehensionCounter int

	// lambdaCounter is used to name the lambda sub-scopes
	lambdaCounter int
}

var propPool = &sync.Pool{
	New: func() interface{} { return &propagator{} },
}

// newPropagator creates a new propagator in the given scope
func newPropagator(ctx kitectx.Context, scope *pythontype.SymbolTable, imp Importer, module *pythontype.SourceModule, helpers *helpers) *propagator {
	ctx.CheckAbort()

	b := propPool.Get().(*propagator)
	*b = propagator{
		ctx:            ctx,
		Scope:          scope,
		InheritedScope: scope,
		Importer:       imp,
		Module:         module,
		h:              helpers,
	}
	return b
}

// discard p for reuse by newPropagator
func (b *propagator) discard() {
	propPool.Put(b)
}

func (b *propagator) trace(format string, objs ...interface{}) {
	if b.h.TraceWriter != nil {
		fmt.Fprintf(b.h.TraceWriter, format+"\n", objs...)
	}
}

func (b *propagator) producePrivate(sym *pythontype.Symbol, v pythontype.Value, private bool) {
	if sym != nil {
		b.trace("assigning %v to %v", v, sym.Name)
		sym.Private = sym.Private && private
		sym.Value = pythontype.Unite(b.ctx, sym.Value, v)
	} else {
		b.trace("attempted to assign %v to nil symbol", v)
	}
}

func (b *propagator) produce(sym *pythontype.Symbol, v pythontype.Value) {
	b.producePrivate(sym, v, false)
}

func (b *propagator) createScope(ident string) *pythontype.SymbolTable {
	// use b.Scope to create the name but use b.InheritedScope as the parent
	childName := b.Scope.Name.WithTail(ident)
	return pythontype.NewSymbolTable(childName, b.InheritedScope)
}

func (b *propagator) createClass(stmt *pythonast.ClassDefStmt) *pythontype.SourceClass {
	members := b.createScope(stmt.Name.Ident.Literal)
	members.Put("__name__", pythontype.StrConstant(stmt.Name.Ident.Literal))
	members.Put("__bases__", pythontype.NewList(pythontype.Builtins.Type))
	members.Put("__module__", b.Module)
	members.Put("__delattr__", nil)
	members.Put("__dict__", nil)
	members.Put("__doc__", pythontype.StrInstance{})
	members.Put("__format__", nil)
	members.Put("__getattribute__", nil)
	members.Put("__hash__", nil)
	members.Put("__new__", nil)
	members.Put("__reduce__", nil)
	members.Put("__reduce_ex__", nil)
	members.Put("__repr__", nil)
	members.Put("__setattr__", nil)
	members.Put("__sizeof__", nil)
	members.Put("__str__", nil)
	members.Put("__subclasshook__", nil)
	members.Put("__weakref__", nil)

	class := &pythontype.SourceClass{Members: members}

	// this should really be an attribute of instances of this class but
	// instances do not currently have their own symbol tables.
	members.Put("__class__", class)
	return class
}

func (b *propagator) createFunction(stmt *pythonast.FunctionDefStmt) *pythontype.SourceFunction {
	fun := &pythontype.SourceFunction{
		Locals: b.createScope(stmt.Name.Ident.Literal),
		Class:  b.Class,
		Module: b.Module,
	}
	fun.Return = &pythontype.Symbol{Name: fun.Locals.Name.WithTail("[return]")}

	if len(stmt.Parameters) > 0 && b.Class != nil {
		switch functionBinding(stmt) {
		case "classmethod":
			fun.HasClassReceiver = true
		case "":
			fun.HasReceiver = true
		}
	}

	for i, param := range stmt.Parameters {
		var paramName string
		if nameExpr, ok := param.Name.(*pythonast.NameExpr); ok {
			paramName = nameExpr.Ident.Literal
		} else {
			paramName = fmt.Sprintf("[param%d]", i)
		}
		fun.Parameters = append(fun.Parameters, pythontype.Parameter{
			Name:        paramName,
			Symbol:      fun.Locals.Create(paramName),
			KeywordOnly: param.KeywordOnly,
		})
	}

	if stmt.Vararg != nil {
		paramName := stmt.Vararg.Name.Ident.Literal
		fun.Vararg = &pythontype.Parameter{
			Name:   paramName,
			Symbol: fun.Locals.Create(paramName),
		}
		b.produce(fun.Vararg.Symbol, pythontype.NewList(nil))
	}

	if stmt.Kwarg != nil {
		paramName := stmt.Kwarg.Name.Ident.Literal
		fun.KwargDict = pythontype.NewKwargDict()
		fun.Kwarg = &pythontype.Parameter{
			Name:   paramName,
			Symbol: fun.Locals.Create(paramName),
		}
		b.produce(fun.Kwarg.Symbol, fun.KwargDict)
	}

	return fun
}

func (b *propagator) createLambda(expr *pythonast.LambdaExpr) *pythontype.SourceFunction {
	name := fmt.Sprintf("[lambda%d]", b.lambdaCounter)
	b.lambdaCounter++

	lambda := pythontype.SourceFunction{
		Locals: b.createScope(name),
		Module: b.Module,
	}
	lambda.Return = &pythontype.Symbol{Name: lambda.Locals.Name.WithTail("[return]")}

	for i, param := range expr.Parameters {
		var paramName string
		if nameExpr, ok := param.Name.(*pythonast.NameExpr); ok {
			paramName = nameExpr.Ident.Literal
		} else {
			paramName = fmt.Sprintf("[param%d]", i)
		}
		lambda.Parameters = append(lambda.Parameters, pythontype.Parameter{
			Name:   paramName,
			Symbol: lambda.Locals.Create(paramName),
		})
	}

	if expr.Vararg != nil {
		paramName := expr.Vararg.Name.Ident.Literal
		lambda.Vararg = &pythontype.Parameter{
			Name:   paramName,
			Symbol: lambda.Locals.Create(paramName),
		}
		b.produce(lambda.Vararg.Symbol, pythontype.NewList(nil))
	}

	if expr.Kwarg != nil {
		paramName := expr.Kwarg.Name.Ident.Literal
		lambda.Kwarg = &pythontype.Parameter{
			Name:   paramName,
			Symbol: lambda.Locals.Create(paramName),
		}
		b.produce(lambda.Kwarg.Symbol, pythontype.NewDict(pythontype.StrInstance{}, nil))
	}

	return &lambda
}

func (b *propagator) createComprehension(expr *pythonast.BaseComprehension) *Comprehension {
	name := fmt.Sprintf("[comprehension%d]", b.comprehensionCounter)
	b.comprehensionCounter++
	return &Comprehension{
		Scope: b.createScope(name),
	}
}

func (b *propagator) propagate(suite []pythonast.Stmt) {
	for _, stmt := range suite {
		b.propagateStmt(stmt)
	}
}

func (b *propagator) propagateStmt(stmt pythonast.Stmt) {
	b.ctx.CheckAbort()

	if pythonast.IsNil(stmt) {
		return
	}
	b.propagateStmtImpl(stmt)
	b.afterStmt(stmt)
}

func (b *propagator) propagateStmtImpl(stmt pythonast.Stmt) {
	switch stmt := stmt.(type) {
	case *pythonast.FunctionDefStmt:
		b.trace("\n=== Function %s ===", stmt.Name.Ident.Literal)
		b.propagateFunctionDefStmt(stmt)
	case *pythonast.ClassDefStmt:
		b.trace("\n=== Class %s ===", stmt.Name.Ident.Literal)
		b.propagateClassDefStmt(stmt)
	case *pythonast.ImportNameStmt:
		b.propagateImportNameStmt(stmt)
	case *pythonast.ImportFromStmt:
		b.propagateImportFromStmt(stmt)
	case *pythonast.ExprStmt:
		b.propagateExprStmt(stmt)
	case *pythonast.AnnotationStmt:
		b.propagateAnnotationStmt(stmt)
	case *pythonast.AssignStmt:
		b.propagateAssignStmt(stmt)
	case *pythonast.AugAssignStmt:
		b.propagateAugAssignStmt(stmt)
	case *pythonast.ReturnStmt:
		b.propagateReturnStmt(stmt)
	case *pythonast.AssertStmt:
		b.propagateAssertStmt(stmt)
	case *pythonast.IfStmt:
		b.propagateIfStmt(stmt)
	case *pythonast.ForStmt:
		b.propagateForStmt(stmt)
	case *pythonast.WhileStmt:
		b.propagateWhileStmt(stmt)
	case *pythonast.WithStmt:
		b.propagateWithStmt(stmt)
	case *pythonast.TryStmt:
		b.propagateTryStmt(stmt)
	case *pythonast.YieldStmt:
		b.propagateYieldStmt(stmt)
	case *pythonast.PrintStmt:
		b.propagatePrintStmt(stmt)
	case *pythonast.ExecStmt:
		b.propagateExecStmt(stmt)
	case *pythonast.DelStmt:
		b.propagateDelStmt(stmt)
	case *pythonast.GlobalStmt:
		b.propagateGlobalStmt(stmt)
	case *pythonast.NonLocalStmt:
		b.propagateNonLocalStmt(stmt)
	case *pythonast.RaiseStmt:
		b.propagateRaiseStmt(stmt)
	case *pythonast.BadStmt:
		b.propagateBadStmt(stmt)
	}
}

func (b *propagator) propagateAnnotationStmt(stmt *pythonast.AnnotationStmt) {
	// see https://www.python.org/dev/peps/pep-0526/#runtime-effects-of-type-annotations

	b.evaluate(stmt.Target)
	if name, ok := stmt.Target.(*pythonast.NameExpr); ok {
		// `foo: annotation` forces foo to be in the local scope
		b.Scope.LocalOrCreatePrivate(name.Ident.Literal, false)
	}

	// local variable annotations are never evaluated by Python due to performance implications
	// but it is acceptable for us to evaluate as if it were, since we want to resolve the full AST
	for _, t := range pythontype.Disjuncts(b.ctx, b.evaluate(stmt.Annotation)) {
		if t, ok := t.(pythontype.Callable); ok {
			b.assignExpr(stmt.Target, t.Call(pythontype.Args{}))
		}
	}
}

func (b *propagator) propagateFunctionDefStmt(stmt *pythonast.FunctionDefStmt) {
	// Get the function or create it.
	fun := b.h.Assembly.Functions[stmt]
	if fun == nil {
		fun = b.createFunction(stmt)
		b.h.Assembly.Functions[stmt] = fun
		b.h.Assembly.PropagateOrder = append(b.h.Assembly.PropagateOrder, stmt)
	}

	var decValues []pythontype.Value
	// Evaluate decorators
	for _, dec := range stmt.Decorators {
		decValues = append(decValues, b.evaluate(dec))
	}

	// Evaluate the return annotation, e.g. "def foo() -> str:"
	for _, t := range pythontype.Disjuncts(b.ctx, b.evaluate(stmt.Annotation)) {
		if t, ok := t.(pythontype.Callable); ok {
			b.produce(fun.Return, t.Call(pythontype.Args{}))
		}
	}

	// Evaluate defaults and annotations on parameters. We do this here in
	// the parent scope since that is what python does.
	for i, param := range stmt.Parameters {
		if !pythonast.IsNil(param.Default) {
			t := b.evaluate(param.Default)
			fun.Parameters[i].Default = t
			b.produce(fun.Parameters[i].Symbol, t)
		}
		for _, t := range pythontype.Disjuncts(b.ctx, b.evaluate(param.Annotation)) {
			if tt, ok := t.(pythontype.Callable); ok {
				b.produce(fun.Parameters[i].Symbol, tt.Call(pythontype.Args{}))
			}
		}
	}

	if stmt.Vararg != nil {
		for _, t := range pythontype.Disjuncts(b.ctx, b.evaluate(stmt.Vararg.Annotation)) {
			if tt, ok := t.(pythontype.Callable); ok {
				b.produce(fun.Vararg.Symbol, pythontype.ListInstance{Element: tt.Call(pythontype.Args{})})
			}
		}
	}
	if stmt.Kwarg != nil {
		for _, t := range pythontype.Disjuncts(b.ctx, b.evaluate(stmt.Kwarg.Annotation)) {
			if tt, ok := t.(pythontype.Callable); ok {
				b.produce(fun.Kwarg.Symbol, pythontype.DictInstance{
					Key:     pythontype.StrInstance{},
					Element: tt.Call(pythontype.Args{}),
				})
			}
		}
	}

	// update function parameters with specialized information
	doParameterHeuristics(b.ctx, fun, b)

	wrapped := pythontype.Value(fun)
	// TODO we should eventually evaluate all decorators using b.doCall,
	// but in order to maintain recall of e.g. argspecs for wrapped functions,
	// there is additional work to be done to propagate *args, **kwargs constraints based on calls,
	// and also potentially handle functools.wraps.
	// For now, we handle only properties
	for i := range decValues {
		dec := decValues[len(decValues)-i-1] // apply decorator inside-out
		_, isUpdater := dec.(pythontype.PropertyUpdater)
		if !isUpdater && !pythontype.Equal(b.ctx, dec, pythontype.Builtins.Property) {
			continue
		}

		args := pythontype.Args{
			Positional: []pythontype.Value{wrapped},
		}
		ret := b.doCall(dec, args, nil)
		if ret != nil {
			wrapped = ret
		}
		// perhaps we want to do
		// wrapped = pythontype.Unite(b.ctx, ret, wrapped)
		// if we are evaluating non-property decorators
	}

	b.assignName(stmt.Name.Ident.Literal, wrapped)
	b.afterExpr(stmt.Name, wrapped)
}

func (b *propagator) propagateClassDefStmt(stmt *pythonast.ClassDefStmt) {
	// Get the class or create it
	class := b.h.Assembly.Classes[stmt]
	if class == nil {
		class = b.createClass(stmt)
		b.h.Assembly.Classes[stmt] = class
	}

	// Evaluate base classes
	var bases []pythontype.Value
	for _, expr := range stmt.Bases() {
		if base := b.evaluate(expr); base != nil {
			b.trace("adding base %v to %v", base, class)
			bases = append(bases, base)
			if baseclass, ok := base.(*pythontype.SourceClass); ok {
				baseclass.AddSubclass(class)
			}
		}
	}
	class.Bases = bases

	// anonymous symbol table to use for class propagation, combined with the
	// symbol table for the class after propagation
	anonymous := b.createScope(stmt.Name.Ident.Literal)

	// Create a child propagator and go through the class def. Classes are evaluated
	// by python at definition time (like default values for function parameters, but
	// unlike the bodies of functions) so we do the same.
	classProp := newPropagator(b.ctx, anonymous, b.Importer, b.Module, b.h)
	// pass the class into the propagator for member function binding
	classProp.Class = class
	// classes do not expose their symbol table to other functions or classes nested within them.
	classProp.InheritedScope = b.InheritedScope
	classProp.propagate(stmt.Body)
	classProp.discard()

	// update class with meta variables if needed
	updateClass(b.ctx, class, anonymous)

	// since updateClass doesn't keep a reference to the table, we can discard it
	anonymous.Discard()

	// python does actually assign
	b.assignName(stmt.Name.Ident.Literal, class)
	b.afterExpr(stmt.Name, class)
}

func (b *propagator) propagateImportNameStmt(stmt *pythonast.ImportNameStmt) {
	for _, clause := range stmt.Names {
		if clause.External == nil || len(clause.External.Names) == 0 {
			continue
		}

		// external is the top-level package name being imported, e.g. "numpy"
		// in "import numpy.linalg as la" or "json" in "import json.special.data"
		external := clause.External.Names[0]
		root, _ := b.Importer.ImportAbs(b.ctx, external.Ident.Literal)

		b.afterExpr(external, root)

		val := root
		for _, name := range clause.External.Names[1:] {
			if val != nil {
				res, _ := pythontype.Attr(b.ctx, val, name.Ident.Literal)
				val = res.Value()
			}
			b.afterExpr(name, val)
		}

		// internal is the local name for the imported package, e.g. "la" in
		// "import numpy.linalg as la" or "json" in "import json.special.data". If
		// there is no internal alias then we simply import the top-level package
		// and assign it to the internal name, since later on the user will use the
		// full path to access members of the package. If there is an internal alias
		// then we must traverse the full import path since that is what gets assigned
		// to the internal alias.
		if clause.Internal == nil {
			b.assignNamePrivate(external.Ident.Literal, root, b.h.Opts.PrivateImports)
		} else {
			b.assignNamePrivate(clause.Internal.Ident.Literal, val, b.h.Opts.PrivateImports)
			b.afterExpr(clause.Internal, val)
		}

		// `*pythonast.DottedExpr` is an expression, so we emit for consistency, we also do this
		// with the package portion of an `*pythonast.ImportFromStmt`.
		// TODO(juan): should we emit this? If not then we should make `DottedExpr` just a node and not an expression.
		b.afterExpr(clause.External, val)
	}
}

func (b *propagator) propagateImportFromStmt(stmt *pythonast.ImportFromStmt) {
	var path []*pythonast.NameExpr
	if stmt.Package != nil {
		path = stmt.Package.Names
	}

	// Import the first component using either ImportRel (e.g. "from ..foo import bar")
	// or ImportAbs (e.g. "e.g. from foo.bar import baz")
	var pkg pythontype.Value
	if numDots := len(stmt.Dots); numDots > 0 {
		pkg, _ = b.Importer.ImportRel(numDots)
	} else if len(path) > 0 {
		pkg, _ = b.Importer.ImportAbs(b.ctx, path[0].Ident.Literal)
		b.afterExpr(path[0], pkg)
		path = path[1:]
	}

	// Traverse the rest of the path components
	for _, name := range path {
		if pkg != nil {
			res, _ := pythontype.Attr(b.ctx, pkg, name.Ident.Literal)
			pkg = res.Value()
		}
		b.afterExpr(name, pkg)
	}

	if stmt.Package != nil {
		b.afterExpr(stmt.Package, pkg)
	}

	// process wildcard import e.g from foo import *
	if stmt.Wildcard != nil && pkg != nil {
		for _, name := range dir(b.ctx, b.Importer.Global, pkg) {
			res, _ := pythontype.Attr(b.ctx, pkg, name)
			if !res.Found() {
				continue
			}
			b.assignNamePrivate(name, res.Value(), false)
		}
	}

	// Process each "foo as bar" clause
	for _, clause := range stmt.Names {
		// I do not believe clause.External can be nil here but just in case...
		if clause.External == nil {
			continue
		}

		// get the type of the symbol being imported
		external := clause.External.Ident.Literal
		var val pythontype.Value
		if pkg != nil {
			res, _ := pythontype.Attr(b.ctx, pkg, external)
			val = res.Value()
		}

		b.afterExpr(clause.External, val)

		// create a local symbol
		if clause.Internal == nil {
			b.assignNamePrivate(external, val, b.h.Opts.PrivateImports)
		} else {
			b.assignNamePrivate(clause.Internal.Ident.Literal, val, false)
			b.afterExpr(clause.Internal, val)
		}
	}
}

func (b *propagator) propagateExprStmt(stmt *pythonast.ExprStmt) {
	b.evaluate(stmt.Value)
}

// propagateLambdaBody propagates the body of a lambda as though it were a return statement.
// This is used when we are propagating values through a lambda as though the lambda were
// a function. This is different to simply evaluating a LambdaExpr, which happens in the parent
// scope and returns a reference to a function.
func (b *propagator) propagateLambdaBody(expr pythonast.Expr) {
	if b.Function == nil {
		panic("return symbol was nil when propagating the body of a lambda")
	}
	t := b.evaluate(expr)
	if t == nil {
		return
	}
	b.canReturn(t)
}

// Here we check for assert isinstance(x, type) and use it to assign a type to x
func (b *propagator) propagateAssertStmt(stmt *pythonast.AssertStmt) {
	// need to evaluate these first so that we can "see" CallExpr
	b.evaluate(stmt.Condition)
	b.evaluate(stmt.Message)

	// is the assertion a CallExpr?
	call, iscall := stmt.Condition.(*pythonast.CallExpr)
	if !iscall || call.Func == nil || len(call.Args) < 2 {
		return
	}
	if pythonast.IsNil(call.Args[0]) || pythonast.IsNil(call.Args[1]) {
		return
	}

	// is the function resolvable?
	fun := b.evaluate(call.Func)
	if fun == nil {
		return
	}

	// is the second argument resolvable?
	arg2 := b.evaluate(call.Args[1].Value)
	if arg2 == nil {
		return
	}

	// do we have isinstance or issubclass?
	if fun.Address().Equals(pythontype.Builtins.IsInstance.Address()) {
		// note that isinstance can take a tuple of types, in which case any type is acceptable
		var vs []pythontype.Value
		switch t := arg2.(type) {
		case pythontype.TupleInstance:
			for _, ti := range t.Elements {
				if ti, iscallable := ti.(pythontype.Callable); iscallable {
					vs = append(vs, ti.Call(pythontype.Args{}))
				}
			}
		case pythontype.Callable:
			vs = append(vs, t.Call(pythontype.Args{}))
		}
		b.assignExpr(call.Args[0].Value, pythontype.Unite(b.ctx, vs...))
	} else if fun.Address().Equals(pythontype.Builtins.IsSubclass.Address()) {
		var vs []pythontype.Value
		if cls, isclass := arg2.(*pythontype.SourceClass); isclass {
			// propose each possible subclass of cls as a potential value for t
			seen := make(map[*pythontype.SourceClass]bool)
			walkSubclasses(b.ctx, cls, func(subclass *pythontype.SourceClass) bool {
				if seen[subclass] {
					return false
				}
				seen[subclass] = true
				vs = append(vs, subclass)
				return true
			})
		}
		b.assignExpr(call.Args[0].Value, pythontype.Unite(b.ctx, vs...))
	}
}

func (b *propagator) propagateAssignStmt(stmt *pythonast.AssignStmt) {
	t := b.evaluate(stmt.Value)

	// loop over multiple assignments like "a = b = c = 1"
	for _, target := range stmt.Targets {
		b.assignExpr(target, t)
	}

	for _, t := range pythontype.Disjuncts(b.ctx, b.evaluate(stmt.Annotation)) {
		// we know stmt.Annotation != nil, so
		// len(stmt.Targets) == 1 && stmt.Targets[0] is not a TupleExpr
		if t, ok := t.(pythontype.Callable); ok {
			b.assignExpr(stmt.Targets[0], t.Call(pythontype.Args{}))
		}
	}
}

func (b *propagator) propagateAugAssignStmt(stmt *pythonast.AugAssignStmt) {
	t := b.evaluate(stmt.Value)
	b.assignExpr(stmt.Target, t)
}

func (b *propagator) propagateReturnStmt(stmt *pythonast.ReturnStmt) {
	if t := b.evaluate(stmt.Value); t != nil {
		b.canReturn(t)
	}
}

func (b *propagator) propagateYieldStmt(stmt *pythonast.YieldStmt) {
	if t := b.evaluate(stmt.Value); t != nil {
		b.canReturn(pythontype.NewGenerator(t))
	}
}

func (b *propagator) propagateIfStmt(stmt *pythonast.IfStmt) {
	// TODO(alex): propose bool as type for condition if it implements Mutable?
	for _, branch := range stmt.Branches {
		b.evaluate(branch.Condition) // must evaluate in order to "see" calls
		b.propagate(branch.Body)
	}
	b.propagate(stmt.Else)
}

func (b *propagator) propagateForStmt(stmt *pythonast.ForStmt) {
	sequence := b.evaluate(stmt.Iterable)
	var elem pythontype.Value
	if seq, ok := sequence.(pythontype.Iterable); ok {
		elem = seq.Elem()
	}
	b.assignExprs(stmt.Targets, elem)
	b.propagate(stmt.Body)
	b.propagate(stmt.Else)
}

func (b *propagator) propagateWhileStmt(stmt *pythonast.WhileStmt) {
	// TODO(alex): propose bool as type for condition if it implements Mutable?
	b.evaluate(stmt.Condition) // must evaluate in order to "see" calls
	b.propagate(stmt.Body)
	b.propagate(stmt.Else)
}

func (b *propagator) propagateWithStmt(stmt *pythonast.WithStmt) {
	// TODO(alex): propose bool as type for condition if it implements Mutable?
	for _, item := range stmt.Items {
		value := b.evaluate(item.Value) // must evaluate in order to "see" calls
		if item.Target != nil {
			b.assignExpr(item.Target, value)
		}
	}
	b.propagate(stmt.Body)
}

func (b *propagator) propagateTryStmt(stmt *pythonast.TryStmt) {
	// TODO(alex): propose bool as type for condition if it implements Mutable?
	b.propagate(stmt.Body)
	for _, clause := range stmt.Handlers {
		var exType pythontype.Value
		if clause.Type != nil {
			exType = b.evaluate(clause.Type) // must evaluate in order to "see" calls
		}
		if clause.Target != nil {
			var ex pythontype.Value
			if ctor, ok := exType.(pythontype.Callable); ok && exType.Kind() == pythontype.TypeKind {
				ex = ctor.Call(pythontype.Args{})
			}
			b.assignExpr(clause.Target, ex)
		}
		b.propagate(clause.Body)
	}
	b.propagate(stmt.Else)
	b.propagate(stmt.Finally)
}

func (b *propagator) propagateBadStmt(stmt *pythonast.BadStmt) {
	b.propagate(stmt.Approximation)
}

// Here we assign types to the variables used in a comprehension. Although a
// comprehension is an expression, we treat the generator part like a statement
// because it has its own symbol table.
func (b *propagator) propagateGenerator(gen *pythonast.Generator) {
	sequence := b.evaluate(gen.Iterable)
	var elem pythontype.Value
	if seq, ok := sequence.(pythontype.Iterable); ok {
		// Widen constants since destructuring a generator is (hopefully rare)
		elem = pythontype.WidenConstants(seq.Elem())
	}
	b.assignExprs(gen.Vars, elem)
	for _, filt := range gen.Filters {
		b.evaluate(filt)
	}
}

func (b *propagator) propagatePrintStmt(stmt *pythonast.PrintStmt) {
	for _, expr := range stmt.Values {
		b.evaluate(expr)
	}
	if !pythonast.IsNil(stmt.Dest) {
		b.evaluate(stmt.Dest)
	}
}

func (b *propagator) propagateExecStmt(stmt *pythonast.ExecStmt) {
	if !pythonast.IsNil(stmt.Body) {
		b.evaluate(stmt.Body)
	}
	if !pythonast.IsNil(stmt.Globals) {
		b.evaluate(stmt.Globals)
	}
	if !pythonast.IsNil(stmt.Locals) {
		b.evaluate(stmt.Locals)
	}
}

func (b *propagator) propagateDelStmt(stmt *pythonast.DelStmt) {
	for _, expr := range stmt.Targets {
		b.evaluate(expr)
	}
}

func (b *propagator) propagateGlobalStmt(stmt *pythonast.GlobalStmt) {
	// TODO(alex): mark these symbols as writeable
	for _, expr := range stmt.Names {
		b.evaluate(expr)
	}
}

func (b *propagator) propagateNonLocalStmt(stmt *pythonast.NonLocalStmt) {
	// TODO(alex): mark these symbols as writeable
	for _, expr := range stmt.Names {
		b.evaluate(expr)
	}
}

func (b *propagator) propagateRaiseStmt(stmt *pythonast.RaiseStmt) {
	if !pythonast.IsNil(stmt.Type) {
		b.evaluate(stmt.Type)
	}
	if !pythonast.IsNil(stmt.Instance) {
		b.evaluate(stmt.Instance)
	}
	if !pythonast.IsNil(stmt.Traceback) {
		b.evaluate(stmt.Traceback)
	}
}

// -- assign*

func (b *propagator) assignExpr(expr pythonast.Expr, t pythontype.Value) {
	b.evaluate(expr)
	switch expr := expr.(type) {
	case *pythonast.NameExpr:
		b.assignNameExpr(expr, t)
	case *pythonast.AttributeExpr:
		b.assignAttributeExpr(expr, t)
	case *pythonast.IndexExpr:
		subs := b.evaluateSubscripts(expr.Subscripts)
		b.assignIndexExpr(expr, subs, t)
	case *pythonast.TupleExpr:
		b.assignTupleExpr(expr, t)
	case *pythonast.ListExpr:
		b.assignListExpr(expr, t)
	}
	b.afterExpr(expr, t) // only for the reference map
}

func (b *propagator) assignExprs(exprs []pythonast.Expr, t pythontype.Value) {
	switch len(exprs) {
	case 0:
		return
	case 1:
		b.assignExpr(exprs[0], t)
	default:
		b.assignExprsDestructure(exprs, t)
	}
}

func (b *propagator) assignExprsDestructure(exprs []pythonast.Expr, t pythontype.Value) {
	switch t := t.(type) {
	case pythontype.DictInstance:
		// special case dicts since they get their keys assigned instead of their values
		for i := range exprs {
			b.assignExpr(exprs[i], t.Key)
		}
	case pythontype.Indexable:
		for i := range exprs {
			b.assignExpr(exprs[i], t.Index(pythontype.IntConstant(i), b.h.Opts.AllowValueMutation))
		}
	case pythontype.Iterable:
		elem := t.Elem()
		for i := range exprs {
			b.assignExpr(exprs[i], elem)
		}
	default:
		// must assign all expressions so that we evaluate all LHS components like
		// "self.foo" in self.foo = unknown
		for i := range exprs {
			b.assignExpr(exprs[i], nil)
		}
	}
}

func (b *propagator) assignNameExpr(expr *pythonast.NameExpr, t pythontype.Value) {
	b.assignName(expr.Ident.Literal, t)
}

func (b *propagator) assignAttributeExpr(expr *pythonast.AttributeExpr, t pythontype.Value) {
	base := b.evaluate(expr.Value)
	if base != nil {
		b.assignAttr(base, expr.Attribute.Literal, t)
	}
}

func (b *propagator) assignIndexExpr(expr *pythonast.IndexExpr, index, val pythontype.Value) {
	var vs []pythontype.Value

	if idxable, ok := b.evaluate(expr.Value).(pythontype.IndexAssignable); ok {
		// base is indexable so get the updated values for the base
		for _, ki := range pythontype.Disjuncts(b.ctx, index) {
			vs = append(vs, idxable.SetIndex(ki, val, b.h.Opts.AllowValueMutation))
		}
	} else {
		if index == nil {
			vs = append(vs, pythontype.NewList(val), pythontype.NewDict(nil, val))
		}
		for _, ki := range pythontype.Disjuncts(b.ctx, index) {
			switch ki.(type) {
			case pythontype.IntInstance, pythontype.IntConstant:
				vs = append(vs, pythontype.NewList(val))
			default:
				// widen constants for keys and values of dicts
				// to make it consistent with evaluating a dict expression
				// in which keys and values are also widened
				ki = pythontype.WidenConstants(ki)
				val = pythontype.WidenConstants(val)
				vs = append(vs, pythontype.NewDict(ki, val))
			}
		}
	}
	b.assignExpr(expr.Value, pythontype.Unite(b.ctx, vs...))
}

func (b *propagator) assignTupleExpr(expr *pythonast.TupleExpr, t pythontype.Value) {
	b.assignExprsDestructure(expr.Elts, t)
}

func (b *propagator) assignListExpr(expr *pythonast.ListExpr, t pythontype.Value) {
	b.assignExprsDestructure(expr.Values, t)
}

func (b *propagator) assignNamePrivate(name string, t pythontype.Value, private bool) {
	b.producePrivate(b.Scope.LocalOrCreatePrivate(name, private), t, private)
}

func (b *propagator) assignName(name string, t pythontype.Value) {
	b.assignNamePrivate(name, t, false)
}

func (b *propagator) assignAttr(base pythontype.Value, attr string, t pythontype.Value) {
	if base == nil {
		return
	}
	if obj, ok := base.(pythontype.Mutable); ok {
		syms := obj.AttrSymbol(b.ctx, attr, true)
		for _, sym := range syms {
			b.trace("assigning %v to %s", t, sym.Name)
			// if any of the disjuncts of the symbol value are properties, call the setter
			for _, v := range pythontype.Disjuncts(b.ctx, sym.Value) {
				if prop, ok := v.(pythontype.PropertyInstance); ok && base.Kind() == pythontype.InstanceKind {
					b.doCall(prop.FSet, pythontype.Args{
						Positional: []pythontype.Value{t},
					}, nil)
				}
			}
			// always unite the symbol value with t
			b.produce(sym, t)
		}
	} else if u, ok := base.(pythontype.Union); ok {
		for _, c := range u.Constituents {
			b.assignAttr(c, attr, t)
		}
	} else {
		b.trace("could not assign to %s attribute of immutable %T", attr, base)
	}
}

// -- helpers

func (b *propagator) canReturn(rv pythontype.Value) {
	if b.Function != nil {
		b.produce(b.Function.Return, rv)
	}
}

// afterStmt is called after the evaluator processes a statement
func (b *propagator) afterStmt(stmt pythonast.Stmt) {
	assign, isAssign := stmt.(*pythonast.AssignStmt)
	if !isAssign {
		return
	}

	if b.h.CapabilityDelegate == nil {
		return
	}
	if len(assign.Targets) != 1 {
		return
	}

	lhs, isName := assign.Targets[0].(*pythonast.NameExpr)
	if !isName {
		return
	}

	lhss := b.Scope.Find(lhs.Ident.Literal)
	if lhss == nil {
		return
	}

	switch rhs := assign.Value.(type) {
	case *pythonast.NameExpr:
		if rhss := b.Scope.Find(rhs.Ident.Literal); rhss != nil {
			b.h.CapabilityDelegate.RecordEdge(rhss, lhss)
		}
	case *pythonast.CallExpr:
		callee := b.evaluate(rhs.Func)
		if callee == nil {
			return
		}

		// get a list of functions that could be called, including class constructors
		var funcs []*pythontype.SourceFunction
		for _, t := range pythontype.Disjuncts(b.ctx, callee) {
			switch t := t.(type) {
			case *pythontype.SourceFunction:
				funcs = append(funcs, t)
			case *pythontype.SourceClass:
				for _, constructorSym := range t.AttrSymbol(b.ctx, "__init__", false) {
					for _, t := range pythontype.Disjuncts(b.ctx, constructorSym.Value) {
						if fun, ok := t.(*pythontype.SourceFunction); ok {
							funcs = append(funcs, fun)
						}
					}
				}
			}
		}

		for _, f := range funcs {
			if f.Return != nil {
				b.h.CapabilityDelegate.RecordEdge(f.Return, lhss)
			}
		}

	}
}

// afterExpr is called by the evaluator after an expression is evaluated.
func (b *propagator) afterExpr(expr pythonast.Expr, value pythontype.Value) {
	if b.h.TraceWriter != nil { // check here to avoid the pythonast.String call
		b.trace("evaluated %s -> %v", pythonast.String(expr), value)
	}
	if b.h.Delegate != nil {
		b.h.Delegate.Resolved(expr, value)
	}
}

// -- evaluate*

func (b *propagator) evaluate(expr pythonast.Expr) pythontype.Value {
	b.ctx.CheckAbort()

	if pythonast.IsNil(expr) {
		return nil
	}
	v := b.evaluateImpl(expr)

	b.afterExpr(expr, v)
	return v
}

func (b *propagator) evaluateImpl(expr pythonast.Expr) pythontype.Value {
	switch expr := expr.(type) {

	case *pythonast.NumberExpr:
		return b.evaluateNumberExpr(expr)
	case *pythonast.StringExpr:
		return b.evaluateStringExpr(expr)
	case *pythonast.ReprExpr:
		return b.evaluateReprExpr(expr)

	case *pythonast.TupleExpr:
		return b.evaluateTupleExpr(expr)
	case *pythonast.ListExpr:
		return b.evaluateListExpr(expr)
	case *pythonast.DictExpr:
		return b.evaluateDictExpr(expr)
	case *pythonast.SetExpr:
		return b.evaluateSetExpr(expr)

	case *pythonast.NameExpr:
		return b.evaluateNameExpr(expr)
	case *pythonast.AttributeExpr:
		return b.evaluateAttributeExpr(expr)
	case *pythonast.CallExpr:
		return b.evaluateCallExpr(expr)
	case *pythonast.IndexExpr:
		return b.evaluateIndexExpr(expr)
	case *pythonast.IfExpr:
		return b.evaluateIfExpr(expr)
	case *pythonast.UnaryExpr:
		return b.evaluateUnaryExpr(expr)
	case *pythonast.BinaryExpr:
		return b.evaluateBinaryExpr(expr)
	case *pythonast.LambdaExpr:
		return b.evaluateLambdaExpr(expr)

	case *pythonast.ComprehensionExpr:
		return b.evaluateComprehensionExpr(expr)
	case *pythonast.ListComprehensionExpr:
		return b.evaluateListComprehensionExpr(expr)
	case *pythonast.DictComprehensionExpr:
		return b.evaluateDictComprehensionExpr(expr)
	case *pythonast.SetComprehensionExpr:
		return b.evaluateSetComprehensionExpr(expr)

	case *pythonast.BadExpr:
		return b.evaluateBadExpr(expr)

	default:
		return nil
	}
}

func (b *propagator) evaluateCallExpr(expr *pythonast.CallExpr) pythontype.Value {
	// must evaluate everything before aborting
	callee := b.evaluate(expr.Func)
	argValues := b.evaluateArgs(expr)

	if callee == nil {
		return nil
	}

	argSymbols := make([]*pythontype.Symbol, len(expr.Args))
	for i, arg := range expr.Args {
		if name, isName := arg.Value.(*pythonast.NameExpr); isName {
			if s := b.Scope.Find(name.Ident.Literal); s != nil {
				argSymbols[i] = s
			}
		}
	}

	return b.doCall(callee, argValues, argSymbols)
}

func (b *propagator) doCall(callee pythontype.Value, args pythontype.Args, argSymbols []*pythontype.Symbol) pythontype.Value {
	out := doCallHeuristics(b.ctx, callee, args, b)

	// get a list of functions that could be called, including class constructors
	var funcs []*pythontype.SourceFunction
	for _, t := range pythontype.Disjuncts(b.ctx, callee) {
		if f, ok := t.(pythontype.Callable); ok {
			out = append(out, f.Call(args))
		}

		// Things we cannot handle simply by delegating to t.Call(...) as above:
		// - user-defined functions: we need to propagate values to parameters
		// - user-defined class: we need to propagate values to constructor parameters
		// - expressions involving "super" such as "super(self, Foo).__init__()"

		if t.Address().Equals(pythontype.Builtins.Super.Address()) {
			if b.Function != nil && b.Function.Class != nil {
				out = append(out, pythontype.NewSuper(b.Function.Class.Bases, nil))
			}
			continue
		} else if t.Address().Equals(pythontype.Builtins.Eval.Address()) {
			out = append(out, b.evaluateEvalCall(args))
			continue
		}

		switch t := t.(type) {
		case *pythontype.SourceFunction:
			funcs = append(funcs, t)
		case *pythontype.SourceClass:
			for _, constructorSym := range t.AttrSymbol(b.ctx, "__init__", false) {
				for _, t := range pythontype.Disjuncts(b.ctx, constructorSym.Value) {
					if fun, ok := t.(*pythontype.SourceFunction); ok {
						funcs = append(funcs, fun)
					}
				}
			}
		}
	}

	// propagate values to each function
	for _, fun := range funcs {
		b.trace("propagating parameters to %s", fun.Locals.Name.String())
		var offset int
		if fun.HasReceiver || fun.HasClassReceiver {
			offset = 1 // ignore the "self" parameter when passing args
		}

		// produce positional arguments
		for i, t := range args.Positional {
			if t == nil {
				continue
			}
			// if argument is past the end of the params then it goes to *args
			if i+offset < len(fun.Parameters) {
				b.produce(fun.Parameters[i+offset].Symbol, t)
			}
		}

		// produce overflow positional arguments as *args
		if fun.Vararg != nil {
			// produce vararg passed in via explicit *arg in call
			if args.Vararg != nil {
				b.produce(fun.Vararg.Symbol, args.Vararg)
			}

			// produce vararg passed in via regular positional arguments that overflowed
			if len(fun.Parameters)-offset < len(args.Positional) {
				overflowArgs := args.Positional[len(fun.Parameters)-offset:]
				b.produce(fun.Vararg.Symbol, pythontype.NewList(pythontype.Unite(b.ctx, overflowArgs...)))
			}
		}

		// produce explicitly named keyword arguments
	outer:
		for _, arg := range args.Keywords {
			// sanity-check the keyword
			if arg.Key == "" {
				continue
			}

			// try to find a matching explicit parameter
			for _, param := range fun.Parameters {
				if param.Name == arg.Key {
					b.produce(param.Symbol, arg.Value)
					continue outer
				}
			}

			// try adding to the implicit **kwargs
			if fun.KwargDict != nil {
				fun.KwargDict.Add(arg.Key, arg.Value)
			}
		}

		// produce values generated by **kwargs
		if kwargdict, ok := args.Kwarg.(*pythontype.KwargDict); ok {
		outer2:
			for argName, argVal := range kwargdict.Entries {
				// try to find an explicit parameter
				for _, param := range fun.Parameters {
					if param.Name == argName {
						b.produce(param.Symbol, argVal)
						continue outer2
					}
				}

				// try adding to the implicit **kwargs
				if fun.KwargDict != nil {
					b.trace("adding to fun.KwargDict")
					fun.KwargDict.Add(argName, argVal)
				}
			}
		}
	}

	if b.h.CapabilityDelegate != nil {
		// add edges for "simple" assignment subgraph
		for _, f := range funcs {
			for i, s := range argSymbols {
				if i >= len(f.Parameters) {
					break
				}
				if s != nil {
					b.h.CapabilityDelegate.RecordEdge(s, f.Parameters[i].Symbol)
				}
			}
		}
	}

	return pythontype.Unite(b.ctx, out...)
}

// evaluateEvalCall evaluates a call to the "eval" builtin. It takes a string argument,
// parses it, evaluates it, and returns the result.
func (b *propagator) evaluateEvalCall(args pythontype.Args) pythontype.Value {
	if len(args.Positional) == 0 {
		return nil
	}

	var opts pythonparser.Options
	var out []pythontype.Value
	for _, v := range pythontype.Disjuncts(b.ctx, args.Positional[0]) {
		str, ok := v.(pythontype.StrConstant)
		if !ok {
			continue
		}

		// make sure this doesn't blow up the analyzer by evaluating something super long
		if len(str) > 10000 {
			continue
		}

		mod, err := pythonparser.Parse(b.ctx, []byte(str), opts)
		if err != nil {
			continue
		}

		// if it is an expression then evaluate it
		if len(mod.Body) == 1 {
			if expr, ok := mod.Body[0].(*pythonast.ExprStmt); ok {
				out = append(out, b.evaluate(expr.Value))
			}
		}

		// if it is not an exprssion then execute it and ignore the value
		b.propagate(mod.Body)
	}
	return pythontype.Unite(b.ctx, out...)
}

func (b *propagator) evaluateLambdaExpr(expr *pythonast.LambdaExpr) pythontype.Value {
	// Get the lambda or create it
	lambda := b.h.Assembly.Lambdas[expr]
	if lambda == nil {
		lambda = b.createLambda(expr)
		b.h.Assembly.Lambdas[expr] = lambda
		b.h.Assembly.PropagateOrder = append(b.h.Assembly.PropagateOrder, expr)
	}

	// Evaluate default parameter values. We do this here in the parent scope
	// since that is what python does.
	for i, param := range expr.Parameters {
		if !pythonast.IsNil(param.Default) {
			t := b.evaluate(param.Default)
			b.produce(lambda.Parameters[i].Symbol, t)
		}
	}
	return lambda
}

// BaseComprehension is embedded within each of the list/set/dict compehensions
// but it does not implement pythonast.Expr on its own
func (b *propagator) evaluateBaseComprehension(expr *pythonast.BaseComprehension) (keyType, valType pythontype.Value, table *pythontype.SymbolTable) {
	comprehension := b.createComprehension(expr)
	compProp := newPropagator(b.ctx, comprehension.Scope, b.Importer, b.Module, b.h)

	table = comprehension.Scope

	// python 3 does not expose these symbols to the parent scope, whereas python 2 did
	// we do it the python 3 way, as it gets us more precise values for names that would be shadowed by the comprehension
	for _, gen := range expr.Generators {
		compProp.propagateGenerator(gen)
	}

	// evaluate the key and value expressions
	if expr.Key != nil {
		keyType = compProp.evaluate(expr.Key)
	}
	if expr.Value != nil {
		valType = compProp.evaluate(expr.Value)
	}
	compProp.discard()
	return
}

func (b *propagator) evaluateComprehensionExpr(expr *pythonast.ComprehensionExpr) pythontype.Value {
	_, elemType, table := b.evaluateBaseComprehension(expr.BaseComprehension)

	b.h.Assembly.Comprehensions[expr] = table

	return pythontype.NewGenerator(elemType)
}

func (b *propagator) evaluateListComprehensionExpr(expr *pythonast.ListComprehensionExpr) pythontype.Value {
	_, elemType, table := b.evaluateBaseComprehension(expr.BaseComprehension)

	b.h.Assembly.Comprehensions[expr] = table

	return pythontype.NewList(elemType)
}

func (b *propagator) evaluateSetComprehensionExpr(expr *pythonast.SetComprehensionExpr) pythontype.Value {
	_, elemType, table := b.evaluateBaseComprehension(expr.BaseComprehension)

	b.h.Assembly.Comprehensions[expr] = table

	return pythontype.NewSet(elemType)
}

func (b *propagator) evaluateDictComprehensionExpr(expr *pythonast.DictComprehensionExpr) pythontype.Value {
	keyType, valType, table := b.evaluateBaseComprehension(expr.BaseComprehension)

	b.h.Assembly.Comprehensions[expr] = table
	return pythontype.NewDict(keyType, valType)
}

func (b *propagator) evaluateNumberExpr(expr *pythonast.NumberExpr) pythontype.Value {
	switch expr.Number.Token {
	case pythonscanner.Int:
		// only track small numbers as constants
		n, err := strconv.ParseInt(expr.Number.Literal, 10, 64)
		if err == nil && n >= 0 && n <= 1000 {
			return pythontype.IntConstant(n)
		}
		return pythontype.IntInstance{}
	case pythonscanner.Long:
		return pythontype.IntInstance{}
	case pythonscanner.Float:
		return pythontype.FloatInstance{}
	case pythonscanner.Imag:
		return pythontype.ComplexInstance{}
	default:
		panic(fmt.Sprintf("unknown token for NumberExpr: %v", expr.Number.Token))
	}
}

func (b *propagator) evaluateStringExpr(expr *pythonast.StringExpr) pythontype.Value {
	return pythontype.StrConstant(expr.Literal())
}

func (b *propagator) evaluateReprExpr(expr *pythonast.ReprExpr) pythontype.Value {
	return pythontype.StrInstance{}
}

func (b *propagator) evaluateNameExpr(expr *pythonast.NameExpr) pythontype.Value {
	// Deal with special identifiers (these are not builtins, nor local symbols)
	ident := expr.Ident.Literal
	switch ident {
	case "None":
		return pythontype.Builtins.None
	case "True":
		return pythontype.Builtins.True
	case "False":
		return pythontype.Builtins.False
	}

	// Evaluate in local scope
	sym := b.Scope.Find(ident)
	if sym == nil {
		return nil
	}
	return sym.Value
}

func (b *propagator) evaluateAttributeExpr(expr *pythonast.AttributeExpr) pythontype.Value {
	val := b.evaluateAttributeExprImpl(expr)
	// record Capabilities for simple symbols,
	// no evaluation of the attribute expression is performed
	// TODO(juan): move this to a delegate
	if b.h.CapabilityDelegate == nil {
		return val
	}

	name, isName := expr.Value.(*pythonast.NameExpr)
	if !isName {
		return val
	}

	s := b.Scope.Find(name.Ident.Literal)
	if s == nil {
		return val
	}

	b.h.CapabilityDelegate.RecordCapability(s, Capability{Attr: expr.Attribute.Literal})

	return val
}

func (b *propagator) evaluateAttributeExprImpl(expr *pythonast.AttributeExpr) pythontype.Value {
	baseUnion := b.evaluate(expr.Value)
	if baseUnion == nil {
		return nil
	}

	var choices []pythontype.Value
	for _, base := range pythontype.Disjuncts(b.ctx, baseUnion) {
		res, _ := pythontype.Attr(b.ctx, base, expr.Attribute.Literal)
		for _, val := range pythontype.Disjuncts(b.ctx, res.Value()) {
			// TODO(naman) this isn't quite correct, as we should only call the getter if the attribute was found on base.Type().
			// For now, this is approximated by simply checking the Kind. We should eventually to generically handle the descriptor protocol.
			if prop, ok := val.(pythontype.PropertyInstance); ok && base.Kind() == pythontype.InstanceKind {
				val = b.doCall(prop.FGet, pythontype.Args{}, nil)
			}
			choices = append(choices, val)
		}
	}

	if len(choices) == 0 && expr.Usage == pythonast.Evaluate {
		return nil
	}

	return pythontype.Unite(b.ctx, choices...)
}

func (b *propagator) evaluateArgs(expr *pythonast.CallExpr) pythontype.Args {
	var args pythontype.Args
	for _, arg := range expr.Args {
		argVal := b.evaluate(arg.Value)
		if arg.Name == nil {
			args.AddPositional(argVal)
		} else {
			if name, ok := arg.Name.(*pythonast.NameExpr); ok {
				args.AddKeyword(name.Ident.Literal, argVal)
				// make sure to register this with any delegate
				// TODO(juan): we could also imagine using the formal parameter
				// for the function to add extra information here, but I think
				// we should do that as part of the explicit "backward propagation" phase.
				b.afterExpr(name, argVal)
			}
		}
	}
	if expr.Vararg != nil {
		args.HasVararg = true // in case Kwarg evaluates to nil
		args.Vararg = b.evaluate(expr.Vararg)
	}
	if expr.Kwarg != nil {
		args.HasKwarg = true // in case Kwarg evaluates to nil
		args.Kwarg = b.evaluate(expr.Kwarg)
	}
	return args
}

func (b *propagator) evaluateSubscripts(subs []pythonast.Subscript) pythontype.Value {
	var vals []pythontype.Value
	for _, sub := range subs {
		switch sub := sub.(type) {
		case *pythonast.IndexSubscript:
			vals = append(vals, b.evaluate(sub.Value))
		case *pythonast.SliceSubscript:
			// we must evaluate all expressions even if we do not use them
			if sub.Lower != nil {
				b.evaluate(sub.Lower)
			}
			if sub.Upper != nil {
				b.evaluate(sub.Upper)
			}
			if sub.Step != nil {
				b.evaluate(sub.Step)
			}
			// TODO(alex): add support for slices
			vals = append(vals, nil)
		case *pythonast.EllipsisExpr:
			// TODO(alex): add Builtins.Ellipsis
			vals = append(vals, nil)
		}
	}
	switch len(vals) {
	case 0:
		return nil
	case 1:
		return vals[0]
	default:
		return pythontype.NewTuple(vals...)
	}
}

func (b *propagator) evaluateIndexExpr(expr *pythonast.IndexExpr) pythontype.Value {
	subs := b.evaluateSubscripts(expr.Subscripts)
	if t := b.evaluate(expr.Value); t != nil {
		if seq, ok := t.(pythontype.Indexable); ok {
			return seq.Index(subs, b.h.Opts.AllowValueMutation)
		}
	}
	return nil
}

func (b *propagator) evaluateIfExpr(expr *pythonast.IfExpr) pythontype.Value {
	// must evaluate so that we see function calls etc
	b.evaluate(expr.Condition)

	var ts []pythontype.Value
	if t := b.evaluate(expr.Body); t != nil {
		ts = append(ts, t)
	}
	if t := b.evaluate(expr.Else); t != nil {
		ts = append(ts, t)
	}
	return pythontype.Unite(b.ctx, ts...)
}

func (b *propagator) evaluateUnaryExpr(expr *pythonast.UnaryExpr) pythontype.Value {
	if t := b.evaluate(expr.Value); t != nil {
		if t, ok := t.(pythontype.BoolConstant); ok {
			return computeBoolUnary(bool(t), expr.Op.Token)
		}
		return pythontype.WidenConstants(t)
	}

	// if internal type could not be resolved then make some
	// guesses based on the operator
	switch expr.Op.Token {
	case pythonscanner.Add, pythonscanner.Sub:
		return pythontype.Unite(b.ctx, pythontype.IntInstance{}, pythontype.FloatInstance{})
	case pythonscanner.BitNot:
		return pythontype.Unite(b.ctx, pythontype.IntInstance{}, pythontype.BoolInstance{})
	}
	return nil
}

func (b *propagator) evaluateBinaryExpr(expr *pythonast.BinaryExpr) pythontype.Value {
	val := b.evaluateBinaryExprImpl(expr)

	// record edge for == comparison,
	if b.h.CapabilityDelegate == nil || expr.Op.Token != pythonscanner.Eq {
		return val
	}

	lhs, isName := expr.Left.(*pythonast.NameExpr)
	if !isName {
		return val
	}

	lhss := b.Scope.Find(lhs.Ident.Literal)
	if lhss == nil {
		return val
	}

	rhs, isName := expr.Right.(*pythonast.NameExpr)
	if !isName {
		return val
	}

	rhss := b.Scope.Find(rhs.Ident.Literal)
	if rhss == nil {
		return val
	}

	b.h.CapabilityDelegate.RecordEdge(lhss, rhss)
	b.h.CapabilityDelegate.RecordEdge(rhss, lhss)

	return val
}

func (b *propagator) evaluateBinaryExprImpl(expr *pythonast.BinaryExpr) pythontype.Value {
	// must always evaluate operands
	left := b.evaluate(expr.Left)
	right := b.evaluate(expr.Right)

	// there are some operators that always / nearly always return the same thing
	switch expr.Op.Token {
	case pythonscanner.Is, pythonscanner.IsNot, pythonscanner.In, pythonscanner.NotIn:
		return pythontype.BoolInstance{}
	case pythonscanner.Truediv:
		return pythontype.FloatInstance{}
	case pythonscanner.Pct:
		return pythontype.StrInstance{}
	case pythonscanner.Eq, pythonscanner.Ne, pythonscanner.Gt, pythonscanner.Ge,
		pythonscanner.Lt, pythonscanner.Le, pythonscanner.Lg:
		return pythontype.BoolInstance{}
	case pythonscanner.Add:
		if lstr, ok := left.(pythontype.StrConstant); ok {
			if rstr, ok := right.(pythontype.StrConstant); ok {
				return lstr + rstr
			}
		}
	}

	var ts []pythontype.Value
	if left != nil {
		ts = append(ts, pythontype.WidenConstants(left))
	}
	if right != nil {
		ts = append(ts, pythontype.WidenConstants(right))
	}

	if len(ts) > 0 {
		return pythontype.Unite(b.ctx, ts...)
	}

	// if operands could not be resolved then make a guess based on the operator
	switch expr.Op.Token {
	case pythonscanner.Sub, pythonscanner.Div:
		return pythontype.Unite(b.ctx, pythontype.IntInstance{}, pythontype.FloatInstance{})
	case pythonscanner.And, pythonscanner.Or, pythonscanner.Not:
		return pythontype.BoolInstance{}
	case pythonscanner.BitAnd, pythonscanner.BitOr, pythonscanner.BitXor,
		pythonscanner.BitLshift, pythonscanner.BitRshift:
		return pythontype.IntInstance{}
	}

	return nil
}

func (b *propagator) evaluateTupleExpr(expr *pythonast.TupleExpr) pythontype.Value {
	var elts []pythontype.Value
	for _, elem := range expr.Elts {
		elts = append(elts, b.evaluate(elem))
	}
	return pythontype.NewTuple(elts...)
}

func (b *propagator) evaluateListExpr(expr *pythonast.ListExpr) pythontype.Value {
	var elemTypes []pythontype.Value
	for _, elem := range expr.Values {
		if t := b.evaluate(elem); t != nil {
			elemTypes = append(elemTypes, t)
		}
	}
	if len(elemTypes) == 0 {
		return pythontype.NewList(nil)
	}
	return pythontype.NewList(pythontype.Unite(b.ctx, elemTypes...))
}

func (b *propagator) evaluateSetExpr(expr *pythonast.SetExpr) pythontype.Value {
	var elemTypes []pythontype.Value
	for _, elem := range expr.Values {
		if t := b.evaluate(elem); t != nil {
			// widen constants since not very common to do destructuring assignments with un ordered collections
			t = pythontype.WidenConstants(t)
			elemTypes = append(elemTypes, t)
		}
	}
	return pythontype.NewSet(pythontype.Unite(b.ctx, elemTypes...))
}

func (b *propagator) evaluateDictExpr(expr *pythonast.DictExpr) pythontype.Value {
	var keytypes, valtypes []pythontype.Value
	keyMap := make(map[pythontype.ConstantValue]pythontype.Value)
	for _, item := range expr.Items {
		var keyCst pythontype.ConstantValue
		var val pythontype.Value
		if keytype := b.evaluate(item.Key); keytype != nil {
			// widen constants since not very common to do destructuring assignments with un ordered collections
			switch keytype.(type) {
			case pythontype.IntConstant, pythontype.StrConstant:
				keyCst = keytype.(pythontype.ConstantValue)
			}
			keytype = pythontype.WidenConstants(keytype)
			keytypes = append(keytypes, keytype)
		}
		if valtype := b.evaluate(item.Value); valtype != nil {
			valtype = pythontype.WidenConstants(valtype)
			val = valtype
			valtypes = append(valtypes, valtype)
		}
		if keyCst != nil {
			keyMap[keyCst] = val
		}
	}
	if len(keytypes) == 0 {
		keytypes = append(keytypes, nil)
	}
	if len(valtypes) == 0 {
		valtypes = append(valtypes, nil)
	}
	if len(keyMap) == 0 && !b.h.Opts.AllowValueMutation {
		// No need for a map as Mutation is not allowed
		keyMap = nil
	}

	return pythontype.NewDictWithMap(pythontype.Unite(b.ctx, keytypes...), pythontype.Unite(b.ctx, valtypes...), keyMap)
}

func (b *propagator) evaluateBadExpr(expr *pythonast.BadExpr) pythontype.Value {
	for _, e := range expr.Approximation {
		b.evaluate(e)
	}
	return nil
}
