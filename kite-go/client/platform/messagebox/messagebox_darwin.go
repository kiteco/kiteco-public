// +build !standalone

package messagebox

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#import <Cocoa/Cocoa.h>

void showAlert(const char* text, char **err) {
	@try {
		@autoreleasepool {
			NSAlert *alert = [[NSAlert alloc] init];
			[alert addButtonWithTitle:@"OK"];
			[alert addButtonWithTitle:@"Cancel"];
			[alert setMessageText:[NSString stringWithUTF8String:text]];
			[alert setAlertStyle:NSWarningAlertStyle];
			[alert runModal];
		}
	} @catch (NSException* ex) {
		*err = strdup([ex.reason UTF8String]);  // caller must free memory
	}
}

void showWarning(const char* keyStr, const char* text, const char* info, char **err) {
	dispatch_sync(dispatch_get_main_queue(), ^{
		@try {
			@autoreleasepool {
				NSString *key = [NSString stringWithUTF8String:keyStr];
				if ([[NSUserDefaults standardUserDefaults] boolForKey:key]) {
					return;
				}
				NSAlert *alert = [[NSAlert alloc] init];
				[alert setMessageText:[NSString stringWithUTF8String:text]];
				[alert setInformativeText:[NSString stringWithUTF8String:info]];
				[alert setAlertStyle:NSWarningAlertStyle];
				[alert setShowsSuppressionButton:YES];
				[[alert suppressionButton] setState:NSControlStateValueOn];
				[alert runModal];

				NSButton *button = [alert suppressionButton];
				if ([button state] == NSControlStateValueOn) {
					[[NSUserDefaults standardUserDefaults] setBool:true forKey:key];
				}
			}
		} @catch (NSException* ex) {
			*err = strdup([ex.reason UTF8String]);  // caller must free memory
		}
	});
}

*/
import "C"
import (
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

func showAlert(opts Options) error {
	var err *C.char
	defer func() {
		if err != nil {
			C.free(unsafe.Pointer(err))
		}
	}()
	C.showAlert(C.CString(opts.Text), &err)
	if err != nil {
		return errors.Errorf("caught objective-c exception in showAlert: %s", C.GoString(err))
	}
	return nil
}

func showWarning(opts Options) error {
	var err *C.char
	defer func() {
		if err != nil {
			C.free(unsafe.Pointer(err))
		}
	}()
	C.showWarning(C.CString(opts.Key), C.CString(opts.Text), C.CString(opts.Info), &err)
	if err != nil {
		return errors.Errorf("caught objective-c exception in showWarning: %s", C.GoString(err))
	}
	return nil
}
