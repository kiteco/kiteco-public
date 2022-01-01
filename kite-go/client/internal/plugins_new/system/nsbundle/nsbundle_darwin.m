#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import "nsbundle_darwin.h"

const char* getVersion(char* path, char **err) {
    @try {
        @autoreleasepool {
            NSString *p = [NSString stringWithUTF8String:path];
            NSBundle *bundle = [NSBundle bundleWithPath:p];
            NSString *s = bundle.infoDictionary[@"CFBundleShortVersionString"];
            return [s UTF8String];
        }
    } @catch (NSException* ex) {
        *err = strdup([ex.reason UTF8String]);  // caller must free memory
        return [@"" UTF8String];
    }
}

int appRunning(char* bundleID, char **err) {
    @try {
        @autoreleasepool {
            NSString *p = [NSString stringWithUTF8String:bundleID];
            NSArray *apps = [NSRunningApplication runningApplicationsWithBundleIdentifier:p];
            return [apps count] > 0;
        }
    } @catch (NSException* ex) {
        *err = strdup([ex.reason UTF8String]);  // caller must free memory
        return 0;
    }
}
