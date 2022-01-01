// +build !standalone

#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>
#import <ServiceManagement/SMLoginItem.h>

#import "_cgo_export.h"
#import "autostart_darwin.h"

void setEnabled(bool enabled) {
    // Set login item so Kite starts on boot if enabled and does not start if disabled
    NSString* bundleName =  getBundleNameForApp(@"KiteAutostart");
    NSLog(@"autostart enabled: %hhd", enabled);
    if(!SMLoginItemSetEnabled((__bridge CFStringRef)bundleName, enabled)) {
            NSLog(@"unable to set login item %@", bundleName);
    }
}

// --

NSString* getBundlePrefix() {
    NSString *bundle = [[NSBundle mainBundle] bundleIdentifier];
    NSArray *parts = [bundle componentsSeparatedByString:@"."];

    NSRange range;
    range.location = 0;
    range.length = 2;

    NSArray *prefixParts = [parts subarrayWithRange:range];
    return [prefixParts componentsJoinedByString:@"."];
}

NSString* getBundleNameForApp(NSString* app) {
    NSString *prefix = getBundlePrefix();
    return [NSString stringWithFormat:@"%@.%@", prefix, app];
}
