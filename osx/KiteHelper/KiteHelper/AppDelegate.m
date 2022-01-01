//
//  AppDelegate.m
//  KiteHelper
//
//  Created by Tarak Upadhyaya on 10/5/15.
//  Copyright © 2015 Tarak Upadhyaya. All rights reserved.
//

#import "AppDelegate.h"
#import <Sparkle/SUUpdater.h>
#import <Sparkle/SUErrors.h>
@import Rollbar;

@interface AppDelegate ()

@property (weak) IBOutlet NSWindow *window;
@property (strong, nonatomic) SUUpdater *updater;
@property (strong, nonatomic) NSInvocation *updateInvocation;
@property (nonatomic) BOOL finishedLaunching;
@property (nonatomic) dispatch_source_t _timer;
@end

@implementation AppDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
    // If this is the enterprise version of KiteHelper, terminate
    if ([self isEnterprise]) {
        NSLog(@"is enterprise, finished launching");
        [NSApp terminate:self];
    }

    // If we are starting up from the .Trash folder, terminate
    NSBundle *mbundle = [NSBundle mainBundle];
    NSLog(@"starting from %@", mbundle.bundlePath);
    if ([mbundle.bundlePath containsString:@"/.Trash/"]) {
        NSLog(@"in trash, terminating");
        [NSApp terminate:self];
    }

    self.finishedLaunching = NO;

    [[NSUserDefaults standardUserDefaults] registerDefaults:@{ @"NSApplicationCrashOnExceptions": @YES }];
    // Rollbar, kited post_client_item
    [Rollbar initWithAccessToken:@"XXXXXXX"];

    NSString *kiteBundle = [self bundleNameForApp:@"Kite"];
    NSString *menubarPath = [[NSWorkspace sharedWorkspace] absolutePathForAppBundleWithIdentifier:kiteBundle];

    // Initialize sparkle updater with Kite bundle
    NSBundle *bundle = [NSBundle bundleWithPath:menubarPath];
    self.updater = [SUUpdater updaterForBundle:bundle];

    // Note: These settings (automatically checks and automatically downloads) are shared with Kite.app as well.
    // Modify with care. See comment for feedURLStringForUpdater for more details.
    [self.updater setAutomaticallyChecksForUpdates:YES];
    [self.updater setAutomaticallyDownloadsUpdates:YES];
    [self.updater setDelegate:self];
    [self.updater checkForUpdatesInBackground];

    self.finishedLaunching = YES;
}

- (void)terminateWhenFinishedHelping {
    NSLog(@"terminateWhenFinishedHelping called");
    uint64_t interval = 5 * NSEC_PER_SEC;
    uint64_t leeway = NSEC_PER_SEC;

    self._timer = dispatch_source_create(DISPATCH_SOURCE_TYPE_TIMER, 0, 0, dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0));
    dispatch_source_set_timer(self._timer, DISPATCH_TIME_NOW, interval, leeway);

    dispatch_source_set_event_handler(self._timer, ^{
        if (self.finishedLaunching == YES) {
            NSLog(@"finished helping. terminating");
            [NSApp terminate:self];
        } else {
            NSLog(@"not finished helping, retrying shortly...");
            dispatch_source_set_timer(self._timer, dispatch_time(DISPATCH_TIME_NOW, interval), interval, leeway);
        }
    });

    // Start the timer
    dispatch_resume(self._timer);
}

- (void)applicationWillTerminate:(NSNotification *)aNotification {
    NSLog(@"terminating KiteHelper");
    if (self.updateInvocation != nil) {
        // If terminating to apply an update, terminate running instances of Kite.
        NSString *kiteBundle = [self bundleNameForApp:@"Kite"];
        NSArray *runningMenuItems = [NSRunningApplication runningApplicationsWithBundleIdentifier:kiteBundle];
        for (NSRunningApplication *app in runningMenuItems) {
            NSLog(@"terminating running menubar: %@, %@", kiteBundle, app);
            [app terminate];
        }
        // Allow Sparkle enough time to update the Kite.app bundle in /Applications.
        [NSThread sleepForTimeInterval:10.0f];
    }
}

- (NSString*) bundleNameForApp: (NSString*)app {
    NSString *prefix = [self bundlePrefix];
    return [NSString stringWithFormat:@"%@.%@", prefix, app];
}


- (NSString*) bundlePrefix {
    NSString *bundle = [[NSBundle mainBundle] bundleIdentifier];
    NSArray *parts = [bundle componentsSeparatedByString:@"."];
    
    NSRange range;
    range.location = 0;
    range.length = 2;
    
    NSArray *prefixParts = [parts subarrayWithRange:range];
    return [prefixParts componentsJoinedByString:@"."];
}

- (bool) isEnterprise {
    return [[self bundlePrefix] hasPrefix:@"enterprise."];
}

#pragma mark - SUUpdaterDelegate

- (NSString *)feedURLStringForUpdater:(SUUpdater *)updater {
    // Since we set the bundle for the Sparkle updater to point to com.kite.Kite and since Sparkle stores values set by the set* functions in the user defaults based on the bundle, any settings created by the set* functions are shared between Kite and KiteHelper. We want the backup updates to check against the fallback endpoint, so we set the feed URL dynamically using this delegate method. Do NOT call setFeedURL as that will add SUFeedURL to the user's defaults for com.kite.Kite.
    NSString *feedURL = [[[NSBundle mainBundle] infoDictionary] objectForKey:@"BackupFeedURL"];
    NSLog(@"checking for backup updates at %@", feedURL);
    return feedURL;
}

- (void)updaterDidNotFindUpdate:(SUUpdater *)updater {
    NSLog(@"did not find backup update");
    // Terminate self if the updater ran and found no update.
    [self terminateWhenFinishedHelping];
}

- (BOOL)updaterShouldRelaunchApplication:(SUUpdater *)updater {
    // This gets called regardless of whether the automatic update is invoked but only relaunches the application if the invocation happened.
    return YES;
}

- (void)updater:(SUUpdater *)updater willInstallUpdateOnQuit:(SUAppcastItem *)item immediateInstallationInvocation:(NSInvocation *)invocation {
    self.updateInvocation = invocation;
    // Invoke the update to relaunch Kite
    NSLog(@"invoking the backup update");
    [invocation invoke];
}

-(void)updater:(SUUpdater *)updater didAbortWithError:(NSError *)error {
    // Don't report when no update was found.
    if ([error code] == SUNoUpdateError) {
        return;
    }
    // Don't report when there is no internet access.
    if ([[error domain] isEqual:NSURLErrorDomain]
        && [error code] == NSURLErrorNotConnectedToInternet) {
        return;
    }

    NSLog(@"updater error: %@", error);
    [NSApp terminate:self];
}
@end
