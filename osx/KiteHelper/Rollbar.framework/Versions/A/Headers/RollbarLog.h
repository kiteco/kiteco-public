//
//  RollbarLog.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-11-12.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

/// Rollbar equivalent of NSLog(...) that also allows capture of the log messages as part of RollbarTelemetry data.
/// NOTE:    When this SDK is used from Objective-C codebase NSLog(...) messages get captured by the SDK.
///        However, when used from Swift codebase, all the NSLog(...) calls in Swift need to be replaced
///        with similar RollbarLog(...) calls to be captured as extra telemetry data.
/// @param format format of the logged message
void RollbarLog(NSString *format, ...);
