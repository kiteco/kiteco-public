package legacy

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Situation represents an editing situation in which we think the user is in due to the state of the AST, buffer,
// and cursor. The Situation is responsible for deciding whether it matches the current state, and for providing
// and scoring the relevant completions.
type Situation interface {
	Name() string
	// Provide attempts to provide ranked completions for a set of inputs.
	Provide(ctx kitectx.Context, in Inputs, cb Callbacks, prefetch bool) ProvisionResult
}

// ProvisionResult contains information that was returned by the situation for the context, including the returned
// completions.
type ProvisionResult struct {
	// Handled is set to true if the situation is able to provide completions. This way, we can add new situations
	// and implement logic for Matching before implementing logic for Provide.
	Handled     bool
	Completions []Completion
	// TypedPrefix is the prefix typed by the user before the cursor that should be replaced by the completion
	TypedPrefix string
	// TypedSuffix is the suffix typed by the user (or editor) after the cursor that should be replaced by the completion
	TypedSuffix string
}

// ---------

// ImportPackage represents a completion situation of the form:
// import foo$
type ImportPackage struct {
	pkg *pythonast.DottedExpr // node can be nil if the user has not yet started typing the package name
}

// Name implements Situation
func (ImportPackage) Name() string { return "ImportPackage" }

// Provide implements Situation
func (i ImportPackage) Provide(ctx kitectx.Context, in Inputs, cb Callbacks, prefetch bool) ProvisionResult {
	ctx.CheckAbort()

	packages := importPackages(ctx, i.pkg, 0, cb)

	// Attempt package-as-alias completions if possible
	var prefix pythonimports.DottedPath
	if i.pkg != nil {
		prefix = pythonimports.NewDottedPath(i.pkg.Join()).Predecessor()
	}
	addAliasesToPackages(ctx, prefix, packages, cb)
	return packages
}

// --------

// FromImport represents a completion situation of the form:
// from foo$
type FromImport struct {
	pkg     *pythonast.DottedExpr // node can be nil if the user has not yet started typing the package name
	numDots int
}

// Name implements Situation
func (FromImport) Name() string { return "FromImport" }

// Provide implements Situation
func (i FromImport) Provide(ctx kitectx.Context, in Inputs, cb Callbacks, prefetch bool) ProvisionResult {
	return importPackages(ctx, i.pkg, i.numDots, cb)
}

// --------

// ImportSubpackage represents a completion situation in which a name is being imported from a package, namely:
// from foo import bar$
type ImportSubpackage struct {
	pkg      *pythonast.DottedExpr
	name     *pythonast.NameExpr // This can be nil if the user has not begun typing the name yet
	fromDots int                 // The number of dots in the ImportFromStmt
}

// Name implements Situation
func (ImportSubpackage) Name() string { return "ImportSubpackage" }

// Provide implements Situation
func (i ImportSubpackage) Provide(ctx kitectx.Context, in Inputs, cb Callbacks, prefetch bool) ProvisionResult {
	subpackages := importSubpackages(ctx, i, cb)

	// Attempt subpackage-as-alias completions if possible
	var prefix pythonimports.DottedPath
	if i.pkg != nil {
		prefix = pythonimports.NewDottedPath(i.pkg.Join())
	}
	addAliasesToPackages(ctx, prefix, subpackages, cb)
	return subpackages
}

// --------

// ImportAlias represents a completion situation in which a package is being imported as a local alias:
// import foo as bar$, or
// from foo import bar as baz$
type ImportAlias struct {
	module *pythonast.NameExpr // the NameExpr referencing the module being imported (i.e. before the `as`)
	alias  *pythonast.NameExpr // The partially typed alias
}

// Name implements Situation
func (ImportAlias) Name() string { return "ImportAlias" }

// Provide implements Situation
func (i ImportAlias) Provide(ctx kitectx.Context, in Inputs, cb Callbacks, prefetch bool) ProvisionResult {
	return importAliases(ctx, in, i, cb)
}
