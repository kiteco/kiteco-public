#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

#include <stdbool.h>

// enable/disable autostart
void setEnabled(bool enabled);

// helper methods to find bundle names
NSString* getBundlePrefix();
NSString* getBundleNameForApp(NSString* app);
