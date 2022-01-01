package pythonskeletons

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"

// --

// NodeTemplate is a template for a node of the provided type and kind
type NodeTemplate struct {
	Kind pythonimports.Kind
	Type *pythonimports.Node
}

// ClassDefaultAttributes is the set of attributes that are included in any class.
func ClassDefaultAttributes(builtins *pythonimports.BuiltinCache) map[string]NodeTemplate {
	return map[string]NodeTemplate{
		"__name__":         NodeTemplate{pythonimports.Object, builtins.Str},
		"__bases__":        NodeTemplate{pythonimports.Object, builtins.Tuple},
		"__class__":        NodeTemplate{pythonimports.Type, builtins.Type},
		"__delattr__":      NodeTemplate{pythonimports.Object, nil},
		"__dict__":         NodeTemplate{pythonimports.Object, builtins.Dict},
		"__doc__":          NodeTemplate{pythonimports.Object, nil},
		"__format__":       NodeTemplate{pythonimports.Descriptor, nil},
		"__getattribute__": NodeTemplate{pythonimports.Object, nil},
		"__hash__":         NodeTemplate{pythonimports.Object, nil},
		"__module__":       NodeTemplate{pythonimports.Object, builtins.Str},
		"__new__":          NodeTemplate{pythonimports.Function, nil},
		"__reduce__":       NodeTemplate{pythonimports.Descriptor, nil},
		"__reduce_ex__":    NodeTemplate{pythonimports.Descriptor, nil},
		"__repr__":         NodeTemplate{pythonimports.Object, nil},
		"__setattr__":      NodeTemplate{pythonimports.Object, nil},
		"__sizeof__":       NodeTemplate{pythonimports.Descriptor, nil},
		"__str__":          NodeTemplate{pythonimports.Object, nil},
		"__subclasshook__": NodeTemplate{pythonimports.Function, nil},
		"__weakref__":      NodeTemplate{pythonimports.Object, nil},
	}
}

// MemberFuncDefaultAttributes is the set of attributes that are included in any member function
func MemberFuncDefaultAttributes(builtins *pythonimports.BuiltinCache) map[string]NodeTemplate {
	return map[string]NodeTemplate{
		"__call__":         NodeTemplate{pythonimports.Object, nil},
		"__class__":        NodeTemplate{pythonimports.Type, builtins.Type},
		"__cmp__":          NodeTemplate{pythonimports.Object, nil},
		"__delattr__":      NodeTemplate{pythonimports.Object, nil},
		"__doc__":          NodeTemplate{pythonimports.Object, nil},
		"__format__":       NodeTemplate{pythonimports.Function, nil},
		"__func__":         NodeTemplate{pythonimports.Function, builtins.Function},
		"__get__":          NodeTemplate{pythonimports.Object, nil},
		"__getattribute__": NodeTemplate{pythonimports.Object, nil},
		"__hash__":         NodeTemplate{pythonimports.Object, nil},
		"__init__":         NodeTemplate{pythonimports.Object, nil},
		"__new__":          NodeTemplate{pythonimports.Function, nil},
		"__reduce__":       NodeTemplate{pythonimports.Function, nil},
		"__reduce_ex__":    NodeTemplate{pythonimports.Function, nil},
		"__repr__":         NodeTemplate{pythonimports.Object, nil},
		"__self__":         NodeTemplate{pythonimports.Object, nil},
		"__setattr__":      NodeTemplate{pythonimports.Object, nil},
		"__sizeof__":       NodeTemplate{pythonimports.Function, nil},
		"__str__":          NodeTemplate{pythonimports.Object, nil},
		"__subclasshook__": NodeTemplate{pythonimports.Function, nil},
		"im_class":         NodeTemplate{pythonimports.Type, builtins.Type},
		"im_func":          NodeTemplate{pythonimports.Function, builtins.Function},
		"im_self":          NodeTemplate{pythonimports.Object, nil},
	}
}

// FuncDefaultAttributes is the set of attributes that are included in any function
func FuncDefaultAttributes(builtins *pythonimports.BuiltinCache) map[string]NodeTemplate {
	return map[string]NodeTemplate{
		"__call__":         NodeTemplate{pythonimports.Object, nil},
		"__class__":        NodeTemplate{pythonimports.Type, builtins.Type},
		"__closure__":      NodeTemplate{pythonimports.Object, nil},
		"__code__":         NodeTemplate{pythonimports.Object, nil},
		"__defaults__":     NodeTemplate{pythonimports.Object, nil},
		"__delattr__":      NodeTemplate{pythonimports.Object, nil},
		"__dict__":         NodeTemplate{pythonimports.Object, builtins.Dict},
		"__doc__":          NodeTemplate{pythonimports.Object, nil},
		"__format__":       NodeTemplate{pythonimports.Function, nil},
		"__get__":          NodeTemplate{pythonimports.Object, nil},
		"__getattribute__": NodeTemplate{pythonimports.Object, nil},
		"__globals__":      NodeTemplate{pythonimports.Object, builtins.Dict},
		"__hash__":         NodeTemplate{pythonimports.Object, nil},
		"__init__":         NodeTemplate{pythonimports.Object, nil},
		"__module__":       NodeTemplate{pythonimports.Object, builtins.Str},
		"__name__":         NodeTemplate{pythonimports.Object, builtins.Str},
		"__new__":          NodeTemplate{pythonimports.Function, nil},
		"__reduce__":       NodeTemplate{pythonimports.Function, nil},
		"__reduce_ex__":    NodeTemplate{pythonimports.Function, nil},
		"__repr__":         NodeTemplate{pythonimports.Object, nil},
		"__setattr__":      NodeTemplate{pythonimports.Object, nil},
		"__sizeof__":       NodeTemplate{pythonimports.Function, nil},
		"__str__":          NodeTemplate{pythonimports.Object, nil},
		"__subclasshook__": NodeTemplate{pythonimports.Function, nil},
		"func_closure":     NodeTemplate{pythonimports.Object, nil},
		"func_code":        NodeTemplate{pythonimports.Object, nil},
		"func_defaults":    NodeTemplate{pythonimports.Object, nil},
		"func_dict":        NodeTemplate{pythonimports.Object, builtins.Dict},
		"func_doc":         NodeTemplate{pythonimports.Object, nil},
		"func_globals":     NodeTemplate{pythonimports.Object, builtins.Dict},
		"func_name":        NodeTemplate{pythonimports.Object, builtins.Str},
	}
}
