package legacy

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

const (
	importAliasThreshold        = 0.05
	minimumKeywordProbThreshold = 0.1
)

func importPackages(ctx kitectx.Context, pkg *pythonast.DottedExpr, numDots int, cb Callbacks) ProvisionResult {
	ctx.CheckAbort()

	var completions []Completion

	edges, typedPrefix := cb.PkgMembers(ctx, pkg, numDots)
	for ident, val := range edges {
		completion := Completion{
			Identifier: ident,
			Referent:   val,
			Score:      cb.ScoreImportEdge(ctx, val),
			Source:     response.TraditionalCompletionSource,
		}
		completions = append(completions, completion)
	}
	sortByScore(completions)

	return ProvisionResult{
		Handled:     true,
		Completions: completions,
		TypedPrefix: typedPrefix,
	}
}

func importSubpackages(ctx kitectx.Context, i ImportSubpackage, cb Callbacks) ProvisionResult {
	ctx.CheckAbort()

	var completions []Completion

	edges, typedPrefix := cb.Subpackages(ctx, i.pkg, i.fromDots, i.name)
	for ident, val := range edges {
		completion := Completion{
			Identifier: ident,
			Referent:   val,
			Score:      cb.ScoreImportEdge(ctx, val),
			Source:     response.TraditionalCompletionSource,
		}
		completions = append(completions, completion)
	}
	sortByScore(completions)

	return ProvisionResult{
		Handled:     true,
		Completions: completions,
		TypedPrefix: typedPrefix,
	}
}

// addAliasesToPackages replace "<package>" completions with "<package> as <alias>" completions sometimes
func addAliasesToPackages(ctx kitectx.Context, prefix pythonimports.DottedPath, pkgRes ProvisionResult, cb Callbacks) {
	ctx.CheckAbort()

	// import aliases are pro-only
	if cb.GetProduct() != licensing.Pro {
		return
	}

	for i, pkgComp := range pkgRes.Completions {
		aliases := cb.ImportAliasesForValue(ctx, pkgComp.Referent)
		var maxFraction, totalAliasFraction float64
		var maxAlias string
		for alias, fraction := range aliases {
			totalAliasFraction += fraction
			if fraction > maxFraction {
				maxFraction = fraction
				maxAlias = alias
			}
		}
		unaliasedFraction := 1. - totalAliasFraction

		// TODO(juan/wathid) proper mixing
		if maxFraction >= importAliasThreshold && maxFraction >= unaliasedFraction {
			pkgComp.Identifier = fmt.Sprintf("%s as %s", pkgComp.Identifier, maxAlias)
		}
		pkgRes.Completions[i] = pkgComp
	}
}

// importAliases returns "foo as bar" completions. Unlike with packageAsAlias, the assumption is that we are confident
// the user is typing an alias.
func importAliases(ctx kitectx.Context, in Inputs, i ImportAlias, cb Callbacks) ProvisionResult {
	ctx.CheckAbort()
	if cb.GetProduct() != licensing.Pro {
		typedPrefix := in.Buffer[int(i.module.Begin()):int(in.Cursor)]
		return ProvisionResult{
			Handled:     true,
			Completions: nil,
			TypedPrefix: string(typedPrefix),
		}
	}

	edge, aliases := cb.ImportAliases(ctx, i.module, i.alias)

	var completions []Completion
	for alias, fraction := range aliases {
		if fraction < importAliasThreshold {
			continue
		}
		completion := Completion{
			// TODO(naman)
			// this is slightly broken because e.g. the user may have typed `foo     as `
			// in which case the completion is considered invalid due to whitespace
			Identifier: fmt.Sprintf("%s as %s", i.module.Ident.Literal, alias),
			Score:      fraction,
			Referent:   edge.Child,
			Source:     response.TraditionalCompletionSource,
		}
		completions = append(completions, completion)
	}
	sortByScore(completions)

	typedPrefix := in.Buffer[int(i.module.Begin()):int(in.Cursor)]
	return ProvisionResult{
		Handled:     true,
		Completions: completions,
		TypedPrefix: string(typedPrefix),
	}
}
