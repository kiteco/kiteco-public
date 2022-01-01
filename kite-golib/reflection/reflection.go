package reflection

import "reflect"

// StructurallyEqual checks if the types t, u are structurally equal.
// This means that it is safe to cast from one to the other.
func StructurallyEqual(t, u reflect.Type) bool {
	if t.Kind() != u.Kind() {
		return false
	}

	// structural types
	switch t.Kind() {
	case reflect.Struct:
		numField := t.NumField()
		if numField != u.NumField() {
			return false
		}
		if t.Size() != u.Size() {
			return false
		}

		for i := 0; i < numField; i++ {
			tF := t.Field(i)
			uF := u.Field(i)
			if tF.Offset != uF.Offset {
				return false
			}
			if !StructurallyEqual(tF.Type, uF.Type) {
				return false
			}
		}
	case reflect.Array:
		if t.Len() != u.Len() {
			return false
		}
		fallthrough
	case reflect.Slice, reflect.Ptr, reflect.Chan:
		return StructurallyEqual(t.Elem(), u.Elem())
	case reflect.Interface:
		return t == u
	case reflect.Func:
		panic("StructurallyEqual not implemented for function types")
	}

	return true
}
