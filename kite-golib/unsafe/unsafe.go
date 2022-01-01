package unsafe

import (
	"reflect"
	"unsafe"
)

// StringToBytes unsafely converts s into a byte slice.
// If you modify b, then s will also be modified. This violates the
// property that strings are immutable.
func StringToBytes(s string) (b []byte) {
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sliceHeader.Data = stringHeader.Data
	sliceHeader.Len = len(s)
	sliceHeader.Cap = len(s)
	return b
}

// BytesToString Unsafely converts b into a string.
// If you modify b, then s will also be modified. This violates the
// property that strings are immutable.
func BytesToString(b []byte) (s string) {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	stringHeader.Data = sliceHeader.Data
	stringHeader.Len = len(b)
	return s
}
