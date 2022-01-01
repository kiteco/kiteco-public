package pythonstatic

import (
	"fmt"
	"path"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// ImportPath represents an import path
type ImportPath struct {
	// Origin is the path of the file containing the described import
	Origin string
	// RelativeDots is a count of the number of initial dots in an `from _ import` statement
	// e.g. from ...foo import bar has RelativeDots: 3
	RelativeDots int
	// Path is the imported dotted path for import statements;
	// for `from _ import` statements, it is the path mentioned between the `from` and `import`
	Path pythonimports.DottedPath
	// Extract is the name mentioned after the `import` in an `from _ import` statement
	// it is empty for wildcard (`*`) imports or simple import statements
	Extract string
}

func dottedParts(expr *pythonast.DottedExpr) []string {
	if expr == nil {
		return nil
	}
	var parts []string
	for _, name := range expr.Names {
		parts = append(parts, name.Ident.Literal)
	}
	return parts
}

// FindImports finds the import statements in a syntax tree
func FindImports(ctx kitectx.Context, origin string, ast *pythonast.Module) []ImportPath {
	ctx.CheckAbort()

	var imports []ImportPath
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		ctx.CheckAbort()

		if _, isexpr := n.(pythonast.Expr); isexpr {
			// we will not find import statements inside expressions
			return false
		}

		switch stmt := n.(type) {
		case *pythonast.ImportNameStmt:
			for _, clause := range stmt.Names {
				parts := dottedParts(clause.External)
				imports = append(imports, ImportPath{
					Origin: origin,
					Path:   pythonimports.NewPath(parts...),
				})
			}

		case *pythonast.ImportFromStmt:
			if stmt.Wildcard != nil {
				// `from foo import *`
				imports = append(imports, ImportPath{
					Origin:       origin,
					RelativeDots: len(stmt.Dots),
					Path:         pythonimports.NewPath(dottedParts(stmt.Package)...),
				})
				break
			}

			for _, clause := range stmt.Names {
				imports = append(imports, ImportPath{
					Origin:       origin,
					RelativeDots: len(stmt.Dots),
					Path:         pythonimports.NewPath(dottedParts(stmt.Package)...),
					Extract:      clause.External.Ident.Literal,
				})
			}
		}
		return true
	})
	return imports
}

// DependencyMap is a map from files to the files they import
type DependencyMap map[string][]string

// ComputeDependencies computes a dependency map from a list of import paths
// TODO(juan): use source tree for this
// TODO(naman): or maybe compute this during file selection and pass it in
func ComputeDependencies(files map[string][]ImportPath) DependencyMap {
	// create the set of directories
	dirs := make(map[string]struct{})
	for srcpath := range files {
		for {
			if !path.IsAbs(srcpath) {
				panic(fmt.Sprintf("ComputeDependencies got a non-absolute path: %s", srcpath))
			}
			srcpath = path.Dir(srcpath)
			dirs[srcpath] = struct{}{}
			if path.Dir(srcpath) == srcpath {
				break
			}
		}
	}

	// resolve dependencies
	deps := make(DependencyMap)
	for srcpath, imps := range files {
		// every source path must appear in map even if it has zero dependencies
		depset := make(map[string]struct{})

	outer:
		for _, imp := range imps {
			if len(imp.Path.Parts) == 0 {
				continue
			}

			// find the python root for this import path
			var root string
			if imp.RelativeDots > 0 {
				// relative path: just take N steps up the hierarchy
				root = srcpath
				for i := 0; i < imp.RelativeDots; i++ {
					root = path.Dir(root)
					if _, isdir := dirs[root]; !isdir {
						continue outer
					}
				}
			} else {
				// absolute path: search up each level of the hierarchy
				root = srcpath
				for {
					root = path.Dir(root)
					if _, isdir := dirs[root]; !isdir {
						continue outer
					}
					firstpath := path.Join(root, imp.Path.Head())
					if _, isdir := dirs[firstpath]; isdir {
						// found a matching directory
						break
					}
					if _, isfile := files[firstpath+".py"]; isfile {
						// found a matching file
						break
					}
					if path.Dir(root) == root {
						continue outer
					}
				}
			}

			// join the path components and look for a matching file
			curpath := root
			var dest string
			for _, part := range imp.Path.Parts {
				curpath = path.Join(curpath, part)

				initpath := path.Join(curpath, "__init__.py")
				if _, ispackage := files[initpath]; ispackage {
					// package found - but don't break here because nested paths should take precedence
					dest = initpath
				}

				modpath := curpath + ".py"
				if _, isfile := files[modpath]; isfile {
					// file found - break here because a file cannot contain other files
					dest = modpath
					break
				}
			}

			if dest != "" {
				depset[dest] = struct{}{}
			}
		}

		// make sure every source path appears in the map even if it has no dependencies
		deps[srcpath] = []string{}
		for dep := range depset {
			deps[srcpath] = append(deps[srcpath], dep)
		}
	}
	return deps
}

// ComputeBootstrapSequence computes an ordering of paths such that each file is, as far as possible,
// positioned after each of its dependencies.
func ComputeBootstrapSequence(deps DependencyMap) []string {
	// recursively add each path
	seen := make(map[string]bool)
	var seq []string
	var push func(srcpath string)
	push = func(srcpath string) {
		if seen[srcpath] {
			return
		}
		seen[srcpath] = true
		for _, dep := range deps[srcpath] {
			push(dep)
		}
		seq = append(seq, srcpath)
	}

	for srcpath := range deps {
		push(srcpath)
	}
	return seq
}

// bootstrap analyzes the import statements in each file and picks an initial ordering of the
// files such that each file is, as far as possible, analyzed after each of its dependencies.
func (b *Assembler) bootstrap() {
	importMap := make(map[string][]ImportPath)
	for name, f := range b.assembly.Files {
		importMap[name] = f.ASTBundle.Imports
	}

	// Compute dependency graph between files
	deps := ComputeDependencies(importMap)

	// Find an ordering of the files such that each file is processed after its dependencies
	paths := ComputeBootstrapSequence(deps)

	// Process each module
	for _, srcpath := range paths {
		if f, ok := b.assembly.Files[srcpath]; ok && f.ASTBundle.AST != nil {
			b.assembly.PropagateOrder = append(b.assembly.PropagateOrder, f.ASTBundle.AST)
		}
	}
}
