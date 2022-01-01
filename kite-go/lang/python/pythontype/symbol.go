package pythontype

import "sync"

// Symbol represents a named data holder, such as a variable, parameter, or return value
type Symbol struct {
	Name  Address
	Value Value
	// Private indicates whether this symbol should be considered "private" to the enclosing SymbolTable/Scope; currently used only for stubs
	Private bool
}

// SymbolTable is a namespace in which symbols may be resolved
type SymbolTable struct {
	Name   Address
	Table  map[string]*Symbol
	Parent *SymbolTable
	m      sync.RWMutex
}

var symTablePool = &sync.Pool{
	New: func() interface{} { return &SymbolTable{} },
}

// NewSymbolTable creates an empty symbol table
func NewSymbolTable(name Address, parent *SymbolTable) *SymbolTable {
	t := symTablePool.Get().(*SymbolTable)
	t.Name = name
	t.Parent = parent
	if t.Table == nil {
		t.Table = make(map[string]*Symbol)
	} else {
		// clear & reuse the map
		for k := range t.Table {
			delete(t.Table, k)
		}
	}
	return t
}

// Discard the symbol table for reuse by NewSymbolTable
func (t *SymbolTable) Discard() {
	symTablePool.Put(t)
}

// Get returns the symbol for the given name
func (t *SymbolTable) Get(name string) (*Symbol, bool) {
	t.m.RLock()
	defer t.m.RUnlock()
	symbol, found := t.Table[name]
	return symbol, found
}

// Find looks up a symbol in this symbol table, then falls back to the parent symbol table
func (t *SymbolTable) Find(name string) *Symbol {
	if sym, found := t.Get(name); found {
		return sym
	}
	if t.Parent != nil {
		return t.Parent.Find(name)
	}
	return nil
}

// CreatePrivate inserts a new symbol with empty type
// TODO(naman) figure out how to avoid adding a notion of "private" symbols just for stub support
func (t *SymbolTable) CreatePrivate(name string, private bool) *Symbol {
	// when creating a symbol, python by default only searches the local
	// symbol table (unless a symbol is explicitly marked global)
	sym := &Symbol{Name: t.Name.WithTail(name), Private: private}
	t.m.Lock()
	defer t.m.Unlock()
	t.Table[name] = sym
	return sym
}

// Create inserts a new symbol with empty type
func (t *SymbolTable) Create(name string) *Symbol {
	return t.CreatePrivate(name, false)
}

// Put overwrites the type for a symbol
func (t *SymbolTable) Put(name string, v Value) *Symbol {
	sym := t.Create(name)
	sym.Value = v
	return sym
}

// LocalOrCreatePrivate looks for a symbol in this symbol table or creates it if it does not exist
// TODO(naman) figure out how to avoid adding a notion of "private" symbols just for stub support
func (t *SymbolTable) LocalOrCreatePrivate(name string, private bool) *Symbol {
	if sym, found := t.Get(name); found {
		// when creating a symbol, python by default only searches the local
		// symbol table (unless a symbol is explicitly marked global)
		return sym
	}
	return t.CreatePrivate(name, private)
}

// LocalOrCreate looks for a symbol in this symbol table or creates it if it does not exist
func (t *SymbolTable) LocalOrCreate(name string) *Symbol {
	return t.LocalOrCreatePrivate(name, false)
}

// FlatSymbol represents a flattened symbol
type FlatSymbol struct {
	Name    Address
	Value   FlatID
	Private bool
}

// FlatSymbolTable represents a flattened symbol table
type FlatSymbolTable struct {
	Parent Address
	Name   Address
	Table  []FlatSymbol
}
