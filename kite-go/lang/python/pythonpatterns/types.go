package pythonpatterns

import (
	"fmt"
	"strings"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// ExternalKind is a simplified form of pythontype.Kind
type ExternalKind string

const (
	// ExternalReturnValue represents a value that was returned from an external function (kind is unknown)
	ExternalReturnValue = ExternalKind("ExternalReturnValueKind")
	// ExternalInstance represents a value that is an instance of an external type (kind is instance)
	ExternalInstance = ExternalKind("ExternalInstanceKind")
	// External represents a value that is an external, (kind can be retrieved from the resource manager)
	External = ExternalKind("External")
)

// Symbol with extra metadata that is suitable for serializtion
type Symbol struct {
	Dist keytypes.Distribution
	Path pythonimports.DottedPath
	Kind ExternalKind
}

// Hash for the symbol
func (s Symbol) Hash() pythonimports.Hash {
	parts := strings.Join([]string{
		s.Dist.Name,
		s.Dist.Version,
		s.Path.String(),
	}, ":")

	return pythonimports.Hash(spooky.Hash64([]byte(parts)))
}

// String ...
func (s Symbol) String() string {
	return fmt.Sprintf("%s.%s", s.Dist.String(), s.Path.String())
}

// Nil symbol
func (s Symbol) Nil() bool {
	return s.Dist == keytypes.Distribution{} && s.Path.Empty() && s.Kind == ""
}

// StrCount maps strings to counts
type StrCount map[string]int

// ExprSummary summarizes a set of expressions seen in python source code,
// each summary is uniquely identified by the set of "symbols" that its value
// is mapped to.
// NOTE: this set of symbols comes from the resolver itself, NOT from an aggregation process in the pipeline.
type ExprSummary struct {
	Syms  []Symbol
	Count int

	// SrcStrs contains source code snippets and counts for how often each snippet was
	// seen in the context of this expression
	SrcStrs StrCount

	// ASTTypes contains the types of the ast nodes that comprised the expression
	// we track this separately from SrcStrs (even though we can in theory parse the src strs)
	// because we filter the src strs to keep the memory manageable
	ASTTypes StrCount
}

// ArgSummary summarizes an argument in a call pattern
type ArgSummary []ExprSummary

// Call pattern for a function
type Call struct {
	Count      int
	Positional []ArgSummary
	Keyword    map[string]ArgSummary
	Hashes     StrCount
}

// Validate the call with the provided arg spec
func (c Call) Validate(as *pythonimports.ArgSpec) error {
	specArgs := as.Args
	if len(specArgs) > 0 && (specArgs[0].Name == "self" || specArgs[0].Name == "cls") {
		specArgs = as.Args[1:]
	}
	if len(c.Positional) > len(specArgs) && as.Vararg == "" {
		return fmt.Errorf("too may positional args %d, max is %d, and no *vararg parameter", len(c.Positional), len(specArgs))
	}

	for kw := range c.Keyword {
		if as.Kwarg != "" {
			// have **kwarg parameter so we cannot verify keyword names
			break
		}

		var found bool
		for _, arg := range specArgs {
			if arg.Name == kw {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid keyword %s and no **kwarg parameter", kw)
		}
	}

	// check to make sure we do not pass a value to the same paramter twice
	for i := 0; i < len(c.Positional); i++ {
		if i < len(specArgs) {
			name := specArgs[i].Name
			if _, ok := c.Keyword[name]; ok {
				return fmt.Errorf("passed parameter %s (%d) multiple times", name, i)
			}
		}
	}

	return nil
}

// Calls groups together call patterns for a particular function
type Calls struct {
	Func  Symbol
	Calls []Call
}
