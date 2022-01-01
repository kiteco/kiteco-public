package pythondocs

func searchList(list []*LangEntity, search *LangEntity) bool {
	for _, le := range list {
		if le.Kind == search.Kind &&
			le.Module == search.Module &&
			le.Ident == search.Ident &&
			le.Sel == search.Sel {
			return true
		}
	}
	return false
}

func searchModule(module *Module, search *LangEntity) (bool, error) {
	switch search.Kind {
	case ModuleKind:
		return searchList([]*LangEntity{module.Documentation}, search), nil
	case ClassKind:
		return searchList(module.Classes, search), nil
	case ExceptionKind:
		return searchList(module.Exceptions, search), nil
	case FunctionKind:
		return searchList(module.Funcs, search), nil
	case MethodKind:
		return searchList(module.ClassMethods, search), nil
	case AttributeKind:
		return searchList(module.ClassAttributes, search), nil
	case VariableKind:
		return searchList(module.Vars, search), nil
	case UnknownKind:
		return searchList(module.Unknown, search), nil
	default:
		return false, errUnknownKind
	}
}
