package tensorflow

// #cgo LDFLAGS: -ldl
// #include <stdlib.h>
// #include <dlfcn.h>
import "C"
import (
	"log"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

func loadTensorflow() error {
	libname := "libtensorflow.1.15.0.dylib"
	log.Printf("loading tensorflow %s", libname)

	libnameCString := C.CString(libname)
	defer C.free(unsafe.Pointer(libnameCString))

	handle := C.dlopen(libnameCString, C.RTLD_LAZY|C.RTLD_GLOBAL)
	if handle == nil {
		err := C.dlerror()
		return errors.Errorf("error loading tensorflow: %s", C.GoString(err))
	}
	return nil
}
