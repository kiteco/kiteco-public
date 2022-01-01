package process

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#include <stdlib.h>
#include "process_darwin.h"
*/
import "C"
import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// GetRunningApplications returns a list of ids of the currently running applications
// If the retrieval of the list failed, then an error is returned
func GetRunningApplications() (List, error) {
	var err *C.char
	defer func() {
		if err != nil {
			C.free(unsafe.Pointer(err))
		}
	}()

	var applicationsLength C.int
	applications := C.getRunningApplications(&applicationsLength, &err)
	if applications == nil {
		if err != nil {
			return nil, fmt.Errorf("caught Objective-C exception in GetRunningApplications(): %s", C.GoString(err))
		}
		return nil, fmt.Errorf("unknown error in GetRunningApplications()")
	}

	defer C.free(unsafe.Pointer(applications))

	var result []Process
	size := int(applicationsLength)
	for i := 0; i < size; i++ {
		cString := C.getElement(applications, C.int(i))
		if cString != nil {
			line := C.GoString(cString)

			entries := strings.Split(line, "|")
			if len(entries) == 3 {
				pid, err := strconv.Atoi(entries[0])
				if err != nil {
					pid = -1
				}

				result = append(result, Process{
					Pid:            pid,
					BundleID:       entries[1],
					BundleLocation: entries[2],
				})
			}

			C.free(unsafe.Pointer(cString))
		}
	}
	return result, nil
}
