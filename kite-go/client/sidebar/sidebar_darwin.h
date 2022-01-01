#import <Foundation/Foundation.h>

#include <stdbool.h>
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

// ActivationObserver notifies when NSWorkspaceDidActivateApplicationNotification is triggered
@interface ActivationObserver : NSObject
@end

// startObserver starts observation of NSWorkspaceDidActivateApplicationNotification
void startObserver(char **err);

// isRunning checks whether the sidebar app is current running
bool isRunning(char **err);

// appPath returns the path to the app bundle
char* appPath();

// launch launches the sidebar app bundle as a new process.
void launch(char **err);

// focus shows the sidebar window if it is hidden and brings it to the front.
void focus(char **err);

// quitSidebar closes the sidebar
void quitSidebar(char **err);

void setWasVisible(bool val);
bool wasVisible();

// helper methods to find bundle names
NSString* bundlePrefix();
NSString* bundleNameForApp(NSString* app);
