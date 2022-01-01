//
//  AppDelegate.m
//  Kite
//
//  Copyright (c) 2015 Manhattan Engineering. All rights reserved.
//

#import "AppDelegate.h"
#import "Constants.h"
#import <ServiceManagement/SMLoginItem.h>
#import <Sparkle/SUUpdater.h>
#import "libkited.h"
@import Rollbar;



@interface AppDelegate () {
  dispatch_source_t _timer;
  bool _initialized;
}

@end

@implementation AppDelegate

- (id)init {
  self = [super init];
  if (self) {
    _initialized = false;
  }
  return self;
}

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
    NSString* helper = [self bundleNameForApp:@"KiteHelper"];

    // Set login item so updater starts on boot
    if(!SMLoginItemSetEnabled((__bridge CFStringRef)helper, true)) {
        NSLog(@"unable to set login item %@", helper);
    }

    // Terminate any running KiteHelper processes indescriminatly. We will relaunch it afterwards. This is needed
    // to ensure any old KiteHelpers (that won't die) don't stick around after this release is shipped.
    NSArray *runningHelperItems = [NSRunningApplication runningApplicationsWithBundleIdentifier:helper];
    for (NSRunningApplication *app in runningHelperItems) {
        [app terminate];
    }

    // Remove Helper from launchd
    [self removeHelperFromLaunchd];

    // Restart the Helper. It will terminate itself.
    NSString *helperPath = [[NSWorkspace sharedWorkspace] absolutePathForAppBundleWithIdentifier:helper];
    NSLog(@"starting helper at: %@", helperPath);
    BOOL started = [[NSWorkspace sharedWorkspace] launchApplication:helperPath];
    NSLog(@"started: %hhd", started);

    // Add Helper to launchd
    [self addHelperToLaunchd];

    // default preferences must be set early (before anybody tries to read prefs)
    NSUserDefaults *prefs = [NSUserDefaults standardUserDefaults];

    // Clear out SUFeedURL preference, which we were foolishly setting earlier:
    [prefs removeObjectForKey:@"SUFeedURL"];

    // setup some environment variables
    kiteSetEnv("HOME", (char*)[NSHomeDirectory() UTF8String]);
    kiteSetEnv("KITE_UPDATE_TARGET", (char*)[[[NSBundle bundleForClass:[self class]] bundlePath] UTF8String]);
    kiteSetEnv("KITE_CONFIGURATION", (char*)[CONFIGURATION UTF8String]);

    // initialize libkited
    if (!kiteInitialize()) {
        NSLog(@"unable to initialize libkited");
        [NSApp terminate:self];
    }

    kiteConnect();

    [[NSUserDefaults standardUserDefaults] registerDefaults:@{ @"NSApplicationCrashOnExceptions": @YES }];
    // Rollbar, kited post_client_item
    [Rollbar initWithAccessToken:@"XXXXXXX"];
    RollbarConfiguration *config = [Rollbar currentConfiguration];
    [config setCheckIgnoreRollbarData:^BOOL(RollbarData *rollbarData) {
        if ([rollbarData.body.crashReport.rawCrashReport containsString:@"SIGPIPE"]) {
            return true;
        }
        return false;
    }];

    // Add notifications for poweroff
    NSNotificationCenter *notificationCenter = [[NSWorkspace sharedWorkspace] notificationCenter];
    [notificationCenter addObserver: self
            selector: @selector(appWillShutOff:)
            name: NSWorkspaceWillPowerOffNotification object: NULL];

    // Be ready to install any updates if the sidebar quits
    [notificationCenter addObserver:self
                           selector:@selector(didTerminateApp:)
                               name:NSWorkspaceDidTerminateApplicationNotification
                             object:nil];

    _initialized = true;
}

- (void) appWillShutOff: (NSNotification *) notification {
    // track when the machine is about to shut off
    NSLog(@"at appWillShutOff");
}

- (void)didTerminateApp:(NSNotification*)notification {
    NSString *sidebar = [self bundleNameForApp:@"KiteApp"];
    if ([[[notification userInfo] objectForKey:@"NSApplicationBundleIdentifier"] isEqualToString:sidebar]) {
        NSLog(@"at didTerminateApp (sidebar terminated)");
    }
}

- (void)applicationWillTerminate:(NSNotification *)aNotification {
    NSLog(@"at applicationWillTerminate");

    // Track whether the sidebar was running when kite was shut down
    kiteTrackSidebarVisibility();

    // We have to call this here because this event cannot be subscribed to via notification center. In particular,
    // this gets called when a Sparkle update is invoked, and is the only way to really make sure we shut down the sidebar.
    kiteStopSidebar();

    if (kiteUpdateReady()) {
        // terminate helper if next launch will be a new version
        NSArray *runningHelperItems = [NSRunningApplication runningApplicationsWithBundleIdentifier:@"com.kite.KiteHelper"];
        for (NSRunningApplication *app in runningHelperItems) {
            NSLog(@"terminating KiteHelper: %@", app);
            [app terminate];
        }
        // remove helper from launchd so the new version will be added on restart
        [self removeHelperFromLaunchd];
    }
}

- (void)removeHelperFromLaunchd {
    NSTask *task = [[NSTask alloc] init];
    [task setLaunchPath: @"/bin/launchctl"];
    NSArray *arguments = [NSArray arrayWithObjects: @"remove", @"com.kite.KiteHelper", nil];
    [task setArguments:arguments];
    [task launch];
    [task waitUntilExit];
    int status = [task terminationStatus];
    if (status == 0) {
        NSLog(@"removed KiteHelper from launchd");
    } else {
        NSLog(@"remove KiteHelper service failed: %d", status);
    }
}

- (void)addHelperToLaunchd {
    NSTask *task = [[NSTask alloc] init];
    [task setLaunchPath: @"/bin/launchctl"];
    NSString *plistPath = [@"~/Library/LaunchAgents/com.kite.KiteHelper.plist" stringByExpandingTildeInPath];
    NSArray *arguments = [NSArray arrayWithObjects: @"load", @"-w", plistPath, nil];
    [task setArguments:arguments];
    [task launch];
    [task waitUntilExit];
    int status = [task terminationStatus];
    if (status == 0) {
        NSLog(@"added KiteHelper to launchd");
    } else {
        [Rollbar error:@"KiteHelperServiceError" exception:nil data:@{@"status": [NSNumber numberWithInt:status]}];
        NSLog(@"load KiteHelper service failed: %d", status);
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

@end
