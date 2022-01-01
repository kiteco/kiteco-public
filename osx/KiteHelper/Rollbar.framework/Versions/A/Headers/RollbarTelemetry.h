//  Copyright (c) 2018 Rollbar, Inc. All rights reserved.

#import <Foundation/Foundation.h>
#import "RollbarLevel.h"
#import "RollbarSource.h"
#import "RollbarTelemetryType.h"
#import "RollbarTelemetryEvent.h"
#import "RollbarTelemetryBody.h"
#import "RollbarTelemetryLogBody.h"
#import "RollbarTelemetryViewBody.h"
#import "RollbarTelemetryErrorBody.h"
#import "RollbarTelemetryNavigationBody.h"
#import "RollbarTelemetryNetworkBody.h"
#import "RollbarTelemetryConnectivityBody.h"
#import "RollbarTelemetryManualBody.h"

#define NSLog(args...) [RollbarTelemetry NSLogReplacement:args];

/// RollbarTelemetry application wide "service" component
@interface RollbarTelemetry : NSObject

/// Shared service instance/singleton
+ (nonnull instancetype)sharedInstance;

/// NSLog replacement
/// @param format NSLog entry format
+ (void)NSLogReplacement:(nonnull NSString *)format, ...;

#pragma mark - Config options

/// Telemetry collection enable/disable switch
@property (readwrite, atomic) BOOL enabled;

/// Enable/disable switch for scrubbing View inputs
@property (readwrite, atomic) BOOL scrubViewInputs;

/// Set of View inputs to scrub
@property (atomic, retain, nullable) NSMutableSet *viewInputsToScrub;

/// Sets whether or not to use replacement log.
/// @param shouldCapture YES/NO flag for the log capture
- (void)setCaptureLog:(BOOL)shouldCapture;

/// Sets max number of telemetry events to capture.
/// @param dataLimit the max total events limit
- (void)setDataLimit:(NSInteger)dataLimit;

#pragma mark - Telemetry data/event recording methods

/// Records/captures a telemetry event
/// @param event a telemetry event
- (void)recordEvent:(nonnull RollbarTelemetryEvent *)event;

/// Records/captures a telemetry event
/// @param level telemetry event level
/// @param source telemetry event source
/// @param body telemetry event body
- (void)recordEventWithLevel:(RollbarLevel)level
                      source:(RollbarSource)source
                   eventBody:(nonnull RollbarTelemetryBody *)body;

/// Records/captures a telemetry event
/// @param level telemetry event level
/// @param body telemetry event body
- (void)recordEventWithLevel:(RollbarLevel)level
                   eventBody:(nonnull RollbarTelemetryBody *)body;

/// Records/captures a telemetry event
/// @param level relevant Rollbar log level
/// @param type telemetry event type
/// @param data event data
- (void)recordEventForLevel:(RollbarLevel)level
                       type:(RollbarTelemetryType)type
                       data:(nullable NSDictionary *)data;

/// Records/captures a telemetry View event
/// @param level relevant Rollbar log level
/// @param element view element
/// @param extraData event data
- (void)recordViewEventForLevel:(RollbarLevel)level
                        element:(nonnull NSString *)element
                      extraData:(nullable NSDictionary *)extraData;

/// Records/captures a telemetry Network event
/// @param level relevant Rollbar log level
/// @param method call method
/// @param url call URL
/// @param statusCode call status code
/// @param extraData event data
- (void)recordNetworkEventForLevel:(RollbarLevel)level
                            method:(nonnull NSString *)method
                               url:(nonnull NSString *)url
                        statusCode:(nonnull NSString *)statusCode
                         extraData:(nullable NSDictionary *)extraData;

/// Records/captures a telemetry Connectivity event
/// @param level relevant Rollbar log level
/// @param status connectivity status
/// @param extraData event data
- (void)recordConnectivityEventForLevel:(RollbarLevel)level
                                 status:(nonnull NSString *)status
                              extraData:(nullable NSDictionary *)extraData;

/// Records/captures a telemetry Error event
/// @param level relevant Rollbar log level
/// @param message error message
/// @param extraData event data
- (void)recordErrorEventForLevel:(RollbarLevel)level
                         message:(nonnull NSString *)message
                       extraData:(nullable NSDictionary *)extraData;

/// Records/captures a telemetry Error event
/// @param level relevant Rollbar log level
/// @param from navigation starting point
/// @param to navigation end point
/// @param extraData event data
- (void)recordNavigationEventForLevel:(RollbarLevel)level
                                 from:(nonnull NSString *)from
                                   to:(nonnull NSString *)to
                            extraData:(nullable NSDictionary *)extraData;

/// Records/captures a telemetry Manual/Custom event
/// @param level relevant Rollbar log level
/// @param extraData event data
- (void)recordManualEventForLevel:(RollbarLevel)level
                         withData:(nonnull NSDictionary *)extraData;

/// Records/captures a telemetry Log event
/// @param level relevant Rollbar log level
/// @param message log message
/// @param extraData event data
- (void)recordLogEventForLevel:(RollbarLevel)level
                       message:(nonnull NSString *)message
                     extraData:(nullable NSDictionary *)extraData;

#pragma mark - Tlemetry cache access methods

-(nonnull NSArray<RollbarTelemetryEvent*> *)getAllEvents;

/// Gets all the currently captured telemetry data/events
- (nullable NSArray *)getAllData;

/// Clears all the currently captured telemetry data/events
- (void)clearAllData;


@end
