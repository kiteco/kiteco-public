package editorapi

// PythonFunctionDetails contains the python specific
// function fields.
type PythonFunctionDetails struct {
	// Receiver is the `self`` or `cls` parameter, or omitted.
	Receiver *Parameter `json:"receiver,omitempty"`

	// Vararg is omitted if there are no `*args`.
	Vararg *Parameter `json:"vararg,omitempty"`

	// Kwarg is omitted if there are no `**kwargs`.
	Kwarg *Parameter `json:"kwarg,omitempty"`

	// KwargParameters are examples of parameters that have been explicity passed in
	// in to the `**kwargs` dict, omitted if there are no `**kwargs`.
	KwargParameters []*Parameter `json:"kwarg_parameters,omitempty"` // parameters passed in to the `**kwargs` dictionary

	// ReturnAnnotation is the explicit return annotation. It is currently never set.
	ReturnAnnotation Union `json:"return_annotation"`
}

// PythonParameterDetails contains the python specific
// parameter fields.
type PythonParameterDetails struct {
	// DefaultValue is the explicit default value.
	DefaultValue Union `json:"default_value"`

	// Annotation is the explicit type annotation. It is currently never set.
	Annotation Union `json:"annotation"`

	// KeywordOnly is true if this is a keyword only parameter.
	KeywordOnly bool `json:"keyword_only"`
}

// PythonTypeDetails contains the python specific
// type fields.
type PythonTypeDetails struct {
	// Bases are the base classes of the type.
	Bases []*PythonBase `json:"bases"`
	// Constructor for the type, e.g the `__init__` method.
	Constructor *FunctionDetails `json:"constructor"`
}

// PythonBase represents a base type in python.
type PythonBase struct {
	ID       ID     `json:"id"`
	Name     string `json:"name"`
	Module   string `json:"module"`
	ModuleID ID     `json:"module_id"`
}

// PythonSignatureDetails contains the python specific
// signature fields.
type PythonSignatureDetails struct {
	// Kwargs passed into a python function
	Kwargs []*ParameterExample `json:"kwargs"`
}

// PythonCallDetails contains the python specific
// `Call` fields.
type PythonCallDetails struct {
	// InKwargs is true if the argument being assigned
	// is in the `**kwargs` dict.
	InKwargs bool `json:"in_kwargs"`
}
