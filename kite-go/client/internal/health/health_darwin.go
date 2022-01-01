package health

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#include <dispatch/dispatch.h>

bool IsResponsive() {
	dispatch_semaphore_t semaphore = dispatch_semaphore_create(0L);
	dispatch_time_t deadline = dispatch_time(DISPATCH_TIME_NOW, 5 * NSEC_PER_SEC);

	dispatch_async(dispatch_get_main_queue(), ^{
		// as soon as the main thread responds, just unlock the semaphore
		dispatch_semaphore_signal(semaphore);
	});

	if (dispatch_semaphore_wait(semaphore, deadline)) {
		// timed out
		return false;
	}

	return true;
}

*/
import "C"

// IsResponsive checks whether the cocoa event loop is responsive
func IsResponsive() bool {
	return bool(C.IsResponsive())
}
