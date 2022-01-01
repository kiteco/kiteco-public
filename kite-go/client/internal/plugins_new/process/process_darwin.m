#include <Foundation/Foundation.h>
#include <Cocoa/Cocoa.h>

#include "process_darwin.h"

/*
* Returns a list of bundle ids of running applications. If an application does not have a bundle ID, then
* the corresponding element is nil.
* The caller is responsible to free the memory allocated at the char** pointer
* and the memory allocated by its elements.
* If an error is returned via **error the caller is responsible to release the memory allocated by it.
*
* Return format: <pid>|<bundle identifier>|<bundle location path>
* Entries are optional.
*/
char** getRunningApplications(int *resultLen, char** err) {
    @try {
        @autoreleasepool {
            NSArray<NSRunningApplication *> *applications = NSWorkspace.sharedWorkspace.runningApplications;
            int len = applications.count;
            char** result = calloc(sizeof(char*), len);

            int i = 0;
            for (NSRunningApplication* app in applications) {
                int pid = app.processIdentifier;
                NSString* id = app.bundleIdentifier ? app.bundleIdentifier : @"";
                NSString *bundlePath = app.bundleURL != nil && app.bundleURL.fileURL ? app.bundleURL.path : @"";

                NSString *line = [NSString stringWithFormat:@"%d|%@|%@", pid, id, bundlePath];
                result[i++] = strdup(line.UTF8String);
            }

            *resultLen = len;
            return result;
        }
    } @catch (NSException* ex) {
        *err = strdup([ex.reason UTF8String]);  // caller must free memory
        return nil;
    }
}

char* getElement(char** v, int index) {
    return v[index];
}