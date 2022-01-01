#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

#include <sys/sysctl.h>
#include <utmpx.h>

#include "startup_darwin.h"

void init() {
    // We set the default value of Yes here so that we know when its set to false. This is used in
    // the darwin mode() to distinguish when plugins launch the sidebar (all the plugins set this value
    // to NO before starting Kite)
    [[NSUserDefaults standardUserDefaults] registerDefaults:@{
        @"shouldReopenSidebar": @YES,
    }];
}

// Function to determine whether the app was manually launched (by clicking
// on it in Finder, or on the Dock, or through Spotlight) vs automatically
// launched (as a login item, or by the self updater).
//
// Works by looking at the "process signature" of the parent process and
// comparing it to a whitelist which we believe covers all possible manual-
// launch cases.
bool wasManuallyLaunched() {
    // Get current process info
    ProcessSerialNumber psn;
    if (GetCurrentProcess(&psn) != noErr) {
        NSLog(@"error getting process serial number");
        return NO;
    }

    ProcessInfoRec info;
    info.processInfoLength = sizeof(ProcessInfoRec);
    info.processName = NULL;
    info.processAppRef = NULL;
    OSStatus error = GetProcessInformation(&psn, &info);
    if (error != noErr) {
        NSLog(@"Error getting process information %d", (int)error);
        return NO;
    }

    // Get info on the launching process
    NSDictionary *parentInfo = (NSDictionary*)CFBridgingRelease(
        ProcessInformationCopyDictionary(
            &info.processLauncher,
            kProcessDictionaryIncludeAllInformationMask
        )
    );

    NSString *parentBundleId = [parentInfo objectForKey:@"CFBundleIdentifier"];
    NSLog(@"Launch process bundle id: \"%@\"", parentBundleId);

    // Whitelist of bundles that the user could use to manually start the Kite.app:
    NSArray *manualStartBundleIds = @[
        @"com.apple.dt.Xcode",  // Xcode
        @"com.apple.finder",    // Finder
        @"com.apple.dock",      // Dock
        @"com.apple.Spotlight", // Spotlight
    ];
    for (NSString *manualBundleId in manualStartBundleIds) {
        if ([parentBundleId isEqualToString:manualBundleId]) {
            return YES;
        }
    }

    // Fallback to manual launched when run on Travis
    if ([parentBundleId length] == 0) {
    	NSString* travisEnv = [[[NSProcessInfo processInfo]environment]objectForKey:@"TRAVIS_OS_NAME"];
	if ([travisEnv length] > 0) {
		return YES;
	}
    }

    return NO;
}

bool shouldReopenSidebar() {
    return [[NSUserDefaults standardUserDefaults] boolForKey:@"shouldReopenSidebar"];
}

void setShouldReopenSidebar(bool val) {
    [[NSUserDefaults standardUserDefaults] setBool:val forKey:@"shouldReopenSidebar"];
}
