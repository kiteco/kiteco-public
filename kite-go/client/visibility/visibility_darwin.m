#import <Foundation/Foundation.h>

bool WindowVisibleImpl(const char* appName) {
	CFStringRef match = CFStringCreateWithCString(nil, appName, kCFStringEncodingUTF8);

	// Get the list of windows
	CGWindowListOption listOptions = kCGWindowListOptionAll | kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements;
	CFArrayRef windows = CGWindowListCopyWindowInfo(listOptions, kCGNullWindowID);
	if (windows == nil) {
		NSLog(@"CGWindowListCopyWindowInfo was nil");
		return false;
	}

	// Find the position of the named window.
	CGFloat sidebarX, sidebarY;
	CFIndex sidebarIndex = -1;
	for (CFIndex i = 0; i < CFArrayGetCount(windows); i++) {
		CFDictionaryRef entry = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
		if (entry == nil) {
			NSLog(@"window %ld was nil", i);
			continue;
		}

		CFDictionaryRef boundsDict = (CFDictionaryRef)CFDictionaryGetValue(entry, kCGWindowBounds);
		if (boundsDict == nil) {
			NSLog(@"bounds for window %ld was nil", i);
			continue;
		}

		CGRect bounds;
		CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds);

		// The menubar icon also shows up as a window under the app named "Kite", but we can
		// eliminate it by checking its size.
		CFStringRef name = (CFStringRef)CFDictionaryGetValue(entry, kCGWindowOwnerName);
		if (name == nil) {
			NSLog(@"name for window %ld was nil", i);
			continue;
		}

		CFComparisonResult cmp = CFStringCompare(name, match, 0);
		if (cmp == kCFCompareEqualTo && bounds.size.height > 100) {
			sidebarIndex = i;
			sidebarX = bounds.origin.x + bounds.size.width / 2.;
			sidebarY = bounds.origin.y + bounds.size.height / 2.;
			break;
		}
	}

	if (sidebarIndex == -1) {
		CFRelease(match);
		CFRelease(windows);
		return false;
	}

	// Now determine whether the window is visible. Windows are ordered front to back so just
	// iterate through the list. Note: this ordering constraint is based on comments in Apple
	// sample code, and is not specified in the documentation.
	for (CFIndex i = 0; i < sidebarIndex; i++) {
		CFDictionaryRef entry = (CFDictionaryRef)CFArrayGetValueAtIndex(windows, i);
		if (entry == nil) {
			NSLog(@"window %ld was nil", i);
			continue;
		}

		CFDictionaryRef boundsDict = (CFDictionaryRef)CFDictionaryGetValue(entry, kCGWindowBounds);
		if (boundsDict == nil) {
			NSLog(@"bounds for window %ld was nil", i);
			continue;
		}

		// ignore the window named "Dock", which on El Capitain is the size of the whole screen
		CFStringRef name = (CFStringRef)CFDictionaryGetValue(entry, kCGWindowOwnerName);
		if (name == nil) {
			NSLog(@"name for window %ld was nil", i);
			continue;
		}
		if (CFStringCompare(name, CFSTR("Dock"), 0) == kCFCompareEqualTo) {
			continue;
		}

		CGRect bounds;
		CGRectMakeWithDictionaryRepresentation(boundsDict, &bounds);

		if (CGRectContainsPoint(bounds, CGPointMake(sidebarX, sidebarY))) {
			CFRelease(match);
			CFRelease(windows);
			return false;
		}
	}

	CFRelease(match);
	CFRelease(windows);
	return true;
}

bool windowVisible(const char* appName) {
	// make sure autoreleased objects are released before this function returns.
	@autoreleasepool {
		return WindowVisibleImpl(appName);
	}
}
