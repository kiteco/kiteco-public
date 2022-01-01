package pythonstatic

import "github.com/kiteco/kiteco/kite-go/lang/python/pythontype"

// Comprehension represents a list/dict/set comprehension. It is _not_ a type
// because it is impossible to refer to the symbol table created by a
// comprehension.
type Comprehension struct {
	Scope *pythontype.SymbolTable
}

// String returns a string representation of a module
func (c *Comprehension) String() string {
	return "comprehension:" + c.Scope.Name.String()
}
