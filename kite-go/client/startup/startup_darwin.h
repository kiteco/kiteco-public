#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

#include <stdbool.h>

void init();
bool wasManuallyLaunched();
void setShouldReopenSidebar(bool val);
bool shouldReopenSidebar();