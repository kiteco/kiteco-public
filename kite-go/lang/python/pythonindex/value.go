package pythonindex

// TODO(naman) unused: rm unless we decide to turn local code search back on

import (
	"strings"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	maxSourceTreeDepth = 8
	minValueCount      = 2
)

// ValueTokens groups together a base and a list of tokens. Conceptually, the
// base and tokens could be used to navigate to the value in the SourceTree
// that this ValueTokens object is relevant to.
//
// As an example, suppose we have the following files:
//
//	/Users/dhung/src/tests/graph.py
//	/Users/dhung/src/tests/import_example.py
//	/Users/dhung/src/tests/example_module/__init__.py
//	/Users/dhung/src/tests/__init__.py
//	/Users/dhung/src/tests/example.py
//
// The base will be /Users/dhung/src and the corresponding tokens are:
//
//	/Users/dhung/src/tests/graph.py                   -> "tests.graph"
//	/Users/dhung/src/tests/import_example.py          -> "tests.import_example"
//	/Users/dhung/src/tests/example_module/__init__.py -> "tests.example_module"
//	/Users/dhung/src/tests/__init__.py                -> "tests"
//	/Users/dhung/src/tests/example.py                 -> "tests.example"
//
// Note how we treat __init__.py files differently - They aren't included in
// the path but their containing folders are.
type ValueTokens struct {
	Value  pythontype.Value
	Base   string
	Tokens []string
}

func (toks *ValueTokens) ident() string {
	return strings.Join(toks.Tokens, ".")
}

func copyTokens(toks *ValueTokens) *ValueTokens {
	cp := ValueTokens{
		Value:  toks.Value,
		Base:   toks.Base,
		Tokens: make([]string, len(toks.Tokens)),
	}
	copy(cp.Tokens, toks.Tokens)
	return &cp
}

func copyTokensWithValue(toks *ValueTokens, v pythontype.Value) *ValueTokens {
	cp := copyTokens(toks)
	cp.Value = v
	return cp
}

// BasePaths takes in a list of files returns a list of file system paths that
// collectively make up the bases of all the files in the input list. In other
// words, any file in the input list is the descendant of exactly one of the
// returned base paths.
//
// This function assumes that the input paths are separated by "/", which is
// consistent with how we represent paths in the backend, regardless of the
// user's OS.
func BasePaths(fs []string) []string {
	var shortest []string
	var counter int
	seen := make(map[string]bool)
	for _, f := range fs {
		// parts will always have at least 2 elements since we assume each file
		// path begins with "/"
		parts := strings.Split(f, "/")
		if parts[len(parts)-1] == "__init__.py" {
			// An __init__.py indicates that the containing directory can also be
			// imported directly
			parts = parts[:len(parts)-1]
		}

		// Set the path of the containing directory
		path := strings.Join(parts[:len(parts)-1], "/") + "/"

		// Check against the current shortest base paths
		if len(shortest) == 0 || len(parts) < counter {
			shortest = []string{path}
			counter = len(parts)
			seen[path] = true
		} else if len(parts) == counter && !seen[path] {
			shortest = append(shortest, path)
			seen[path] = true
		}
	}
	return shortest
}

// SourceTreeTokens maps each value contained in a pythonenv SourceTree to
// the tokens that lead to the value. It returns a map that can be used to
// find the tokens associated to a value using the value's locator.
func SourceTreeTokens(t *pythonenv.SourceTree) map[string][]*ValueTokens {
	vals := make(map[string][]*ValueTokens)
	var fs []string
	for f := range t.Files {
		fs = append(fs, f)
	}
	bases := BasePaths(fs)
	for fn, mod := range t.Files {
		tokenizeFile(fn, mod, bases, vals)
	}
	return vals
}

// ValueCounter maps a value to the number of times it is referred to in a
// codebase.
type ValueCounter func(pythontype.Value) int

// IndexSourceTree creates an inverted index from a pythonenv SourceTree.
// It accepts a ValueCounter to set the count of values.
func IndexSourceTree(t *pythonenv.SourceTree, cnts ValueCounter) map[string][]*IdentCount {
	return indexTokens(SourceTreeTokens(t), cnts)
}

func newDiskmapIndex(dm *diskmap.Map, cache *lru.Cache) *index {
	return &index{
		diskIndex: &diskmapIndex{
			index: dm,
			cache: cache,
		},
	}
}

// --

func tokenizeFile(fn string, mod pythontype.Value, bases []string, vals map[string][]*ValueTokens) {
	if mod == nil {
		return
	}

	for _, base := range bases {
		if !strings.HasPrefix(fn, base) {
			continue
		}

		// Get path and name
		path := strings.Split(strings.TrimPrefix(fn, base), "/")
		name := path[len(path)-1]
		path = path[:len(path)-1]

		// Create the initial ValueToken
		toks := &ValueTokens{
			Value:  mod,
			Base:   base,
			Tokens: path,
		}

		// Recursively tokenize. This should happen only once since each
		// file name will match only one base.
		tokenize(mod, toks, name, vals, 1)
		break
	}
}

func tokenize(
	v pythontype.Value, toks *ValueTokens, name string,
	vals map[string][]*ValueTokens, depth int) {
	if v == nil || depth > maxSourceTreeDepth {
		return
	}
	switch v.Kind() {
	case pythontype.ModuleKind:
		tokenizeModule(v, toks, name, vals, depth)
	case pythontype.TypeKind:
		tokenizeType(v, toks, name, vals, depth)
	case pythontype.FunctionKind:
		tokenizeFunction(v, toks, name, vals, depth)
	}
}

func tokenizeModule(
	v pythontype.Value, toks *ValueTokens, name string,
	vals map[string][]*ValueTokens, depth int) {
	storeTokens := func() {
		if name != "__init__.py" {
			toks.Tokens = append(toks.Tokens, strings.TrimSuffix(name, ".py"))
		}
		if key := pythonenv.Locator(v); key != "" {
			vals[key] = append(vals[key], toks)
		}
	}

	// Recursively tokenize members
	switch v := v.(type) {
	case *pythontype.SourceModule:
		storeTokens()
		if v.Members == nil {
			return
		}
		for attr, member := range v.Members.Table {
			if member == nil || member.Value == nil || strings.HasPrefix(attr, "_") {
				continue
			}
			if member.Value.Kind() != pythontype.ModuleKind {
				// Ignore modules that are referenced from other modules
				for _, m := range pythontype.Disjuncts(kitectx.TODO(), member.Value) {
					tokenize(m, copyTokensWithValue(toks, m), attr, vals, depth+1)
				}
			}
		}
	case *pythontype.SourcePackage:
		// Note that we don't index the DirEntries because those are already
		// included in the SourceTree's top-level files map
		storeTokens()
		if v.Init == nil || v.Init.Members == nil {
			return
		}
		for attr, member := range v.Init.Members.Table {
			if member == nil || member.Value == nil || strings.HasPrefix(attr, "_") {
				continue
			}
			if member.Value.Kind() != pythontype.ModuleKind {
				// Ignore modules that are referenced from other modules
				for _, m := range pythontype.Disjuncts(kitectx.TODO(), member.Value) {
					tokenize(m, copyTokensWithValue(toks, m), attr, vals, depth+1)
				}
			}
		}
	}
}

func tokenizeType(
	v pythontype.Value, toks *ValueTokens, name string,
	vals map[string][]*ValueTokens, depth int) {
	toks.Tokens = append(toks.Tokens, name)
	if key := pythonenv.Locator(v); key != "" {
		vals[key] = append(vals[key], toks)
	}

	cls, ok := v.(*pythontype.SourceClass)
	if !ok {
		return
	}
	for attr, member := range cls.Members.Table {
		if member == nil || member.Value == nil || strings.HasPrefix(attr, "_") {
			continue
		}
		for _, m := range pythontype.Disjuncts(kitectx.TODO(), member.Value) {
			tokenize(m, copyTokensWithValue(toks, m), attr, vals, depth+1)
		}
	}
}

func tokenizeFunction(
	v pythontype.Value, toks *ValueTokens, name string,
	vals map[string][]*ValueTokens, depth int) {
	if key := pythonenv.Locator(v); key != "" {
		toks.Tokens = append(toks.Tokens, name)
		vals[key] = append(vals[key], toks)
	}
}

func indexTokens(vals map[string][]*ValueTokens, cnts ValueCounter) map[string][]*IdentCount {
	idx := make(map[string][]*IdentCount)
	for key, toks := range vals {
		for _, tok := range toks {
			cnt := cnts(tok.Value)
			fcnt := cnt
			if fcnt < minValueCount {
				fcnt = minValueCount
			}
			ic := &IdentCount{
				Ident:       tok.ident(),
				Count:       cnt,
				ForcedCount: fcnt,
				Locator:     key,
			}
			for _, s := range text.Uniquify(text.Lower(tok.Tokens)) {
				idx[s] = append(idx[s], ic)
			}
		}
	}
	return idx
}
