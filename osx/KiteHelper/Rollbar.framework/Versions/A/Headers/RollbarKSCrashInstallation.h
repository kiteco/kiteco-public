//  Copyright © 2018 Rollbar. All rights reserved.

#import <Foundation/Foundation.h>
#import "KSCrash.h"
#import "KSCrashInstallation.h"

@interface RollbarKSCrashInstallation : KSCrashInstallation

+ (instancetype)sharedInstance;
- (void)sendAllReports;

@end
