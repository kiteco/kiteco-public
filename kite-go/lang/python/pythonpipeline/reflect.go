package pythonpipeline

import "reflect"

func derefType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		return derefType(t.Elem())
	default:
		return t
	}
}

// TypeName for the specified obj, if the object is a container (or pointer)
// then we return the name of the contained type, e.g. for []int we return int.
func TypeName(obj interface{}) string {
	return derefType(reflect.TypeOf(obj)).Name()
}
