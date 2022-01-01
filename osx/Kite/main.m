//
//  main.m
//  Kite
//
//  Copyright (c) 2015 Manhattan Engineering. All rights reserved.
//

#import <Cocoa/Cocoa.h>
#import "AppDelegate.h"

int main(int argc, const char * argv[]) {
    // When run from Xcode, the app crashes whenever it encounters SIGPIPE.
    // This ignores the signal to prevent this. The go runtime ignores
    // SIGPIPE as well, but it does not seem to be always effective.
    signal(SIGPIPE, SIG_IGN);
    AppDelegate *delegate = [[AppDelegate alloc] init];
    [[NSApplication sharedApplication] setDelegate:delegate];
    [NSApp run];
    return 0;
}
