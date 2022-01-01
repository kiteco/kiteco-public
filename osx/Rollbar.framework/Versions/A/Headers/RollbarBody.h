//
//  RollbarBody.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-11-27.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

@class RollbarTelemetry;
@class RollbarTrace;
@class RollbarMessage;
@class RollbarCrashReport;

NS_ASSUME_NONNULL_BEGIN

@interface RollbarBody : DataTransferObject

#pragma mark - Required but mutually exclusive properties

// Required: "trace", "trace_chain", "message", or "crash_report" (exactly one)
// If this payload is a single exception, use "trace"
// If a chain of exceptions (for languages that support inner exceptions), use "trace_chain"
// If a message with no stack trace, use "message"
// If an iOS crash report, use "crash_report"

// Option 1: "trace"
@property (nonatomic, strong, nullable) RollbarTrace *trace;

// Option 2: "trace_chain"
// Used for exceptions with inner exceptions or causes
// Each element in the list should be a "trace" object, as shown above.
// Must contain at least one element.
@property (nonatomic,strong, nullable) NSArray<RollbarTrace *> *traceChain;

// Option 3: "message"
// Only one of "trace", "trace_chain", "message", or "crash_report" should be present.
// Presence of a "message" key means that this payload is a log message.
@property (nonatomic, strong, nullable) RollbarMessage *message;

// Option 4: "crash_report"
// Only one of "trace", "trace_chain", "message", or "crash_report" should be present.
@property (nonatomic, strong, nullable) RollbarCrashReport *crashReport;

#pragma mark - Optional properties

// Optional: "telemetry". Only applicable if you are sending telemetry data.
@property (nonatomic, strong, nullable) RollbarTelemetry *telemetry;

#pragma mark - Initializers

-(instancetype)initWithMessage:(nonnull NSString *)message;
-(instancetype)initWithException:(nonnull NSException *)exception;
-(instancetype)initWithError:(nonnull NSError *)error;
-(instancetype)initWithCrashReport:(nonnull NSString *)crashReport;

@end

NS_ASSUME_NONNULL_END
