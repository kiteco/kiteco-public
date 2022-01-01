package symbolcounts

// Counts represents counts for a given symbol by context in which the symbol and its descendants appeared.
type Counts struct {
	Import    int
	Name      int
	Attribute int
	Expr      int

	// ImportThis counts the number of times this symbol (and not its descendants) appears in an import context.
	ImportThis int

	// ImportAliases maintains counts of aliases that this symbol was imported as. It is not mutually exclusive
	// with the other categories.
	// ex. if there is one occurrence of "import foo as bar", then the Counts for foo will be
	// {Import: 1, ImportAliases: {"bar": 1}}.
	ImportAliases map[string]int
}

// Scorer represents a function that returns a score given a set of counts for a symbol.
type Scorer func(Counts) int

// NewCounts returns an empty set of Counts.
func NewCounts() Counts {
	return Counts{
		ImportAliases: make(map[string]int),
	}
}

// Add returns a new Counts struct containing the counts that are a sum of s and other.
func (s Counts) Add(other Counts) Counts {
	newAliases := make(map[string]int)
	for alias, count := range s.ImportAliases {
		newAliases[alias] += count
	}
	for alias, count := range other.ImportAliases {
		newAliases[alias] += count
	}

	return Counts{
		Import:        s.Import + other.Import,
		Name:          s.Name + other.Name,
		Attribute:     s.Attribute + other.Attribute,
		Expr:          s.Expr + other.Expr,
		ImportThis:    s.ImportThis + other.ImportThis,
		ImportAliases: newAliases,
	}
}

// IncorporateChild incorporates the counts of a child symbol into s. The import alias counts are not included
// because they are specific to the exact symbol being imported.
func (s Counts) IncorporateChild(child Counts) Counts {
	return Counts{
		Import:        s.Import + child.Import,
		Name:          s.Name + child.Name,
		Attribute:     s.Attribute + child.Attribute,
		Expr:          s.Expr + child.Expr,
		ImportThis:    s.ImportThis,
		ImportAliases: s.ImportAliases,
	}
}

// Sum returns the sum of all the count types.
// ImportAliases and ImportThis are not counted because they are a subset of import counts.
func (s Counts) Sum() int {
	return s.Import + s.Name + s.Attribute + s.Expr
}

// Empty returns true if all the counts in s are zero.
func (s Counts) Empty() bool {
	return s.Import == 0 && s.Name == 0 && s.Attribute == 0 && s.Expr == 0 && s.ImportThis == 0 && len(s.ImportAliases) == 0
}

// TopLevelCounts describes a set of counts of symbols for a given top-level module,
// along with counts for the module itself.
type TopLevelCounts struct {
	TopLevel string
	Count    Counts
	Symbols  []*SymbolCounts
}

// SymbolCounts describes counts for a particular symbol in the resource manager.
type SymbolCounts struct {
	Path  string
	Count Counts
}
