//  Copyright © 2018 Rollbar. All rights reserved.

#import <Foundation/Foundation.h>
#import "KSCrash.h"

@interface RollbarKSCrashReportSink : NSObject <KSCrashReportFilter>

- (id<KSCrashReportFilter>)defaultFilterSet;

@end
