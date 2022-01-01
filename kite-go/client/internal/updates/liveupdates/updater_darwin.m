// +build !standalone

#import "updater_darwin.h"
#import "_cgo_export.h"

#import <Foundation/Foundation.h>
#import <Sparkle/Sparkle.h>
#import <QuartzCore/QuartzCore.h>

@interface UpdateDelegate : NSObject<SUUpdaterDelegate>
@end

static SUUpdater* updater = nil;
static UpdateDelegate* delegate = nil;
static NSInvocation* curUpdate = nil;
static CFTimeInterval curUpdateTime;

@implementation UpdateDelegate
- (void)updater:(SUUpdater *)updater willInstallUpdateOnQuit:(SUAppcastItem *)item immediateInstallationInvocation:(NSInvocation *)invocation {
	curUpdate = invocation;
    curUpdateTime = CACurrentMediaTime();
	cgoWillInstallUpdateOnQuit();
}
- (NSArray *)feedParametersForUpdater:(SUUpdater *)updater sendingSystemProfile:(BOOL)sendingProfile {
    char* cMachineID = cgoGetMachineID();
    NSString* machineID = [NSString stringWithUTF8String:cMachineID];
    free(cMachineID);
    return @[@{
                 @"key" : @"machine-id",
               @"value" : machineID,
          @"displayKey" : @"machine-id",
        @"displayValue" : machineID,
    }];
}
@end

void start(const char* bundlePath, char **err) {
	@try {
		@autoreleasepool {
			NSString* path = [NSString stringWithUTF8String:bundlePath];
			delegate = [[UpdateDelegate alloc] init];
			updater = [SUUpdater updaterForBundle:[NSBundle bundleWithPath:path]];
			[updater setAutomaticallyChecksForUpdates:YES];
			[updater setAutomaticallyDownloadsUpdates:YES];
			[updater setDelegate:delegate];
		}
	} @catch (NSException* ex) {
		*err = strdup([ex.reason UTF8String]);  // caller must free memory
	}
}

void checkForUpdates(bool showModal, char **err) {
	@try {
		@autoreleasepool {
			if (updater == nil) {
				NSLog(@"CheckForUpdates called without first calling Init");
				return;
			}
			if (showModal) {
				[updater performSelectorOnMainThread:@selector(checkForUpdates:) withObject:nil waitUntilDone:NO];
			} else {
                NSString *feedURL = [[[NSBundle mainBundle] infoDictionary] objectForKey:@"SUFeedURL"];
                NSArray *runningHelperItems = [NSRunningApplication runningApplicationsWithBundleIdentifier:@"com.kite.KiteHelper"];
                if ([runningHelperItems count] > 0) {
                    NSLog(@"helper running, waiting to check for updates");
                    dispatch_after(dispatch_time(DISPATCH_TIME_NOW,
                                                 60 * NSEC_PER_SEC),
                                                 dispatch_get_main_queue(),
                                                 ^{
                        NSArray *runningHelperItems = [NSRunningApplication runningApplicationsWithBundleIdentifier:@"com.kite.KiteHelper"];
                        if ([runningHelperItems count] > 0) {
                            NSLog(@"helper still running, skipping check for updates");
                        } else {
                            NSLog(@"checking for updates at %@", feedURL);
                            [updater performSelectorOnMainThread:@selector(checkForUpdatesInBackground) withObject:nil waitUntilDone:NO];
                        }
                    });
                } else {
                    NSLog(@"checking for updates at %@", feedURL);
                    [updater performSelectorOnMainThread:@selector(checkForUpdatesInBackground) withObject:nil waitUntilDone:NO];
                }
			}
		}
	} @catch (NSException* ex) {
		*err = strdup([ex.reason UTF8String]);  // caller must free memory
	}
}

bool updateReady() {
	@autoreleasepool {
		return curUpdate != nil;
	}
}

int secondsSinceUpdateReady() {
	@autoreleasepool {
        if (curUpdate == nil) {
            return 0;
        }
        return CACurrentMediaTime() - curUpdateTime;
	}
}


void restartAndUpdate(char **err) {
	@try {
		@autoreleasepool {
			if (curUpdate == nil) {
				NSLog(@"Update called but curUpdate was nil");
				return;
			}

			[curUpdate invoke];
		}
	} @catch (NSException* ex) {
		*err = strdup([ex.reason UTF8String]);  // caller must free memory
	}
}
