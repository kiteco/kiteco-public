package version

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

const char *version() {
  return [[[[NSBundle mainBundle] infoDictionary] objectForKey:@"CFBundleVersion"] UTF8String];
}
*/
import "C"

// Version returns a string representation of the current Kite version. This
// is the string that appears, for example, in the menubar item.
func Version() string {
	if v := C.GoString(C.version()); v != "" {
		return v
	}
	// fallback to the value which is also used in Kite's Info.plist file
	// the bundle version is unavailable in tests
	return "9999"
}

// IsDebugBuild returns true if this is a debug build (which is true whenever
// Kite was not built by the distribution scripts).
func IsDebugBuild() bool {
	return Version() == "9999"
}

// IsDevMode returns true if dev features should be enabled, such as the
// "Servers" submenu within the menubar item, special logging behavior on
// windows, etc.
func IsDevMode() bool {
	return IsDebugBuild()
}
