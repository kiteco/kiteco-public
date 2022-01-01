package legacy

import (
	"regexp"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	// asClausePrefixRegexp should match any prefixes of '<whitespace>as<whitespace>'
	asClausePrefixRegexp = regexp.MustCompile(`^(\s+|\s+a|\s+as\s*)$`)
)

// Match returns a situation most appropriately matching the inputs. It returns nil if no such situation is available.
func Match(ctx kitectx.Context, in Inputs) Situation {
	ctx.CheckAbort()

	// Determine whether we're in the context of an import.
	for _, node := range in.UnderTrimmedCursor {
		switch t := node.(type) {
		case *pythonast.BadStmt:
			if s := matchBadImportStmt(in, t); s != nil {
				return s
			}
		case *pythonast.ImportFromStmt:
			return matchImportFrom(in, t)
		case *pythonast.ImportNameStmt:
			return matchImportName(in, t)
		}
	}

	return nil
}

// matchBadImportStmt attempts to glean an import situation from a BadStmt. It is done as to work around the fact that
// approximate parsing can be a little fragile. Currently, it only supports finding an importAlias situation in the case
// where someone is typing the "as" part of a "foo as bar" import clause.
func matchBadImportStmt(in Inputs, stmt *pythonast.BadStmt) Situation {
	// See if there's an ImportNameStmt or ImportFromStmt that was approximately parsed from the BadStmt
	var importName *pythonast.ImportNameStmt
	var importFrom *pythonast.ImportFromStmt
	for _, s := range stmt.Approximation {
		switch s := s.(type) {
		case *pythonast.ImportNameStmt:
			importName = s
		case *pythonast.ImportFromStmt:
			importFrom = s
		}
	}

	if importName != nil {
		if len(importName.Names) == 0 {
			return nil
		}
		// Find the last "foo as bar" clause that begins at or before the cursor
		var clause *pythonast.DottedAsName
		for _, c := range importName.Names {
			if in.Cursor >= int64(c.Begin()) {
				clause = c
			}
		}
		if pythonast.IsNil(clause) {
			return nil
		}
		if pythonast.IsNil(clause.External) || in.Cursor <= int64(clause.External.End()) {
			return nil
		}
		if !pythonast.IsNil(clause.Internal) && in.Cursor > int64(clause.Internal.End()) {
			return nil
		}

		if len(clause.External.Names) == 0 {
			return nil
		}
		lastName := clause.External.Names[len(clause.External.Names)-1]
		afterExternal := string(in.Buffer[lastName.End():in.Cursor])
		if !asClausePrefixRegexp.MatchString(afterExternal) {
			return nil
		}

		return ImportAlias{
			module: lastName,
			alias:  clause.Internal,
		}
	}

	if importFrom != nil {
		if len(importFrom.Names) == 0 {
			return nil
		}
		// Find the last "foo as bar" clause that begins at or before the cursor
		var clause *pythonast.ImportAsName
		for _, c := range importFrom.Names {
			if in.Cursor >= int64(c.Begin()) {
				clause = c
			}
		}
		if pythonast.IsNil(clause) {
			return nil
		}
		if pythonast.IsNil(clause.External) || in.Cursor <= int64(clause.External.End()) {
			return nil
		}
		if !pythonast.IsNil(clause.Internal) && in.Cursor > int64(clause.Internal.End()) {
			return nil
		}

		afterExternal := string(in.Buffer[clause.External.End():in.Cursor])
		if !asClausePrefixRegexp.MatchString(afterExternal) {
			return nil
		}

		return ImportAlias{module: clause.External, alias: clause.Internal}
	}

	return nil
}

func matchImportFrom(in Inputs, stmt *pythonast.ImportFromStmt) Situation {
	if in.Cursor < int64(stmt.From.End) {
		return nil
	} else if in.Cursor == int64(stmt.From.End) {
		// expr := &pythonast.NameExpr{Ident: stmt.From, Usage: pythonast.Import}
		// We have to be careful here as the prefix used for the matching will be empty, so having explicit keyword probs is important
		// return Keyword{expr: expr, keywordProbs: map[pythonscanner.Token]float32{pythonscanner.From: 1.0}}
		return nil
	}

	numDots := len(stmt.Dots)
	if pythonast.IsNil(stmt.Package) {
		if numDots == 0 || in.Cursor == int64(stmt.Dots[numDots-1].End) {
			return FromImport{pkg: nil, numDots: numDots}
		}
	} else if in.Cursor <= int64(stmt.Package.End()) {
		return FromImport{pkg: stmt.Package, numDots: numDots}
	}

	// Don't show completions if there is only whitespace after the package name.
	if pythonhelpers.UnderCursor(stmt.Package, in.TrimmedCursor) {
		return nil
	}

	var clause *pythonast.ImportAsName
	for _, c := range stmt.Names {
		if pythonhelpers.UnderCursor(c, in.TrimmedCursor) {
			clause = c
			break
		}
	}

	if pythonast.IsNil(clause) {
		if stmt.Import != nil && in.Cursor > int64(stmt.Import.End) {
			return ImportSubpackage{pkg: stmt.Package, name: nil, fromDots: numDots}
		}
		return nil
	}

	if !pythonast.IsNil(clause.Internal) && in.Cursor > int64(clause.Internal.End()) {
		return nil
	}

	if pythonast.IsNil(clause.External) || in.Cursor <= int64(clause.External.End()) {
		return ImportSubpackage{pkg: stmt.Package, name: clause.External, fromDots: numDots}
	}

	if !pythonast.IsNil(clause.External) && in.Cursor > int64(clause.External.End()) && numDots == 0 {
		// assert !pythonast.IsNil(stmt.Package), since numDots == 0
		return ImportAlias{module: clause.External, alias: clause.Internal}
	}

	return nil
}

func matchImportName(in Inputs, stmt *pythonast.ImportNameStmt) Situation {
	var clause *pythonast.DottedAsName
	for _, c := range stmt.Names {
		if pythonhelpers.UnderCursor(c, in.TrimmedCursor) {
			clause = c
			break
		}
	}

	if pythonast.IsNil(clause) {
		if in.Cursor > int64(stmt.Import.End) {
			return ImportPackage{pkg: nil}
		}
		if in.Cursor == int64(stmt.Import.End) {
			// expr := &pythonast.NameExpr{Ident: stmt.Import, Usage: pythonast.Import}
			// We have to be careful here as the prefix used for the matching will be empty, so having explicit keyword probs is important
			// return Keyword{expr: expr, keywordProbs: map[pythonscanner.Token]float32{pythonscanner.Import: 1.0}}
			return nil
		}
		return nil
	}

	if !pythonast.IsNil(clause.Internal) && in.Cursor > int64(clause.Internal.End()) {
		return nil
	}

	if !pythonast.IsNil(clause.External) {
		if in.Cursor <= int64(clause.External.End()) {
			return ImportPackage{pkg: clause.External}
		}
		return ImportAlias{
			module: clause.External.Names[len(clause.External.Names)-1],
			alias:  clause.Internal,
		}
	}

	return nil
}
