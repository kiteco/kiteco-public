//
//  main.m
//  KiteAutostart
//
//  Created by Tarak Upadhyaya on 10/5/15.
//  Copyright © 2015 Tarak Upadhyaya. All rights reserved.
//

#import <Cocoa/Cocoa.h>
#import "AppDelegate.h"

int main(int argc, const char * argv[]) {
    AppDelegate *delegate = [[AppDelegate alloc] init];
    [[NSApplication sharedApplication] setDelegate:delegate];
    [NSApp run];
    return 0;
}
