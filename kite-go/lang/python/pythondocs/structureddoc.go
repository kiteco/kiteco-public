package pythondocs

// StructuredDoc contains parsed structural information about a module, class, method, function or variable.
type StructuredDoc struct {
	Ident           string
	Parameters      []*Parameter
	DescriptionHTML string
	ReturnType      string
}

// ParameterType is the type of a parameter in a class, method, function or exception  definition.
type ParameterType int

const (
	// UnsetParameterType is an unset parameter.
	UnsetParameterType ParameterType = iota
	// RequiredParamType is a required parameter. Example: arg
	RequiredParamType
	// OptionalParamType is an optional parameter. Example: [arg]
	OptionalParamType
	// KwParamType is a parameter with a specified default. Example: arg=None
	KwParamType
	// VarParamType is a variadic positional parameter. Example: *args
	VarParamType
	// VarKwParamType is a variadic keyword parameter. Example: **kwargs
	VarKwParamType
)

func (p ParameterType) String() string {
	switch p {
	case UnsetParameterType:
		return "Unset"
	case RequiredParamType:
		return "Required"
	case OptionalParamType:
		return "Optional"
	case KwParamType:
		return "Keyword"
	case VarParamType:
		return "Variadic positional"
	case VarKwParamType:
		return "Variadic keyword"
	}
	return "Unknown parameter"
}

// Parameter represents a parameter in a class, method or function definition.
type Parameter struct {
	Type    ParameterType
	Name    string
	Default string // only set for KwParamType

	DescriptionHTML string
}
