package pythontype

// KeywordArg represents a key=value argument to a python function
type KeywordArg struct {
	Key   string
	Value Value
}

// Args represents arguments to a python function
type Args struct {
	Positional []Value
	Keywords   []KeywordArg
	Vararg     Value
	HasVararg  bool // HasVararg is because the vararg might have unknown value
	Kwarg      Value
	HasKwarg   bool // HasKwarg is because the vararg might have unknown value
}

// AddPositional adds a positional argument
func (a *Args) AddPositional(value Value) {
	a.Positional = append(a.Positional, value)
}

// AddKeyword adds a keyword argument
func (a *Args) AddKeyword(name string, value Value) {
	a.Keywords = append(a.Keywords, KeywordArg{name, value})
}

// Keyword gets the value for the specified keyword
func (a *Args) Keyword(name string) (Value, bool) {
	for _, kw := range a.Keywords {
		if kw.Key == name {
			return kw.Value, true
		}
	}
	return nil, false
}

// Positional creates positional arguments
func Positional(vs ...Value) Args {
	return Args{Positional: vs}
}
