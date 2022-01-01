package tensorflow

// #cgo linux LDFLAGS: -ldl
// #include <stdlib.h>
// #include <dlfcn.h>
import "C"

import (
	"log"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

func loadTensorflow() error {
	var libName = "libtensorflow.so.1.15.0"
	log.Printf("Loading tensorflow libary %s", libName)

	name := C.CString(libName)
	defer C.free(unsafe.Pointer(name))

	handle := C.dlopen(name, C.RTLD_LAZY|C.RTLD_GLOBAL)
	if handle == nil {
		err := C.dlerror()
		return errors.Errorf("error loading tensorflow: %v", C.GoString(err))
	}
	return nil
}
