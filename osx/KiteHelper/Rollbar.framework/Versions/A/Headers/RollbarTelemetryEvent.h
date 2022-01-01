//
//  RollbarTelemetryEvent.h
//  Rollbar
//
//  Created by Andrey Kornich on 2020-02-28.
//  Copyright © 2020 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#import "DataTransferObject.h"
#import "RollbarLevel.h"
#import "RollbarTelemetryType.h"
#import "RollbarSource.h"

@class RollbarTelemetryBody;

NS_ASSUME_NONNULL_BEGIN

@interface RollbarTelemetryEvent : DataTransferObject

#pragma mark - Properies
// Can contain any arbitrary keys. Rollbar understands the following:

// Required: level
// The severity level of the telemetry data. One of: "critical", "error", "warning", "info", "debug".
@property (nonatomic, readonly) RollbarLevel level;

// Required: type
// The type of telemetry data. One of: "log", "network", "dom", "navigation", "error", "manual".
@property (nonatomic, readonly) RollbarTelemetryType type;

// Required: source
// The source of the telemetry data. Usually "client" or "server".
@property (nonatomic, readonly) RollbarSource source;

// Required: timestamp_ms
// When this occurred, as a unix timestamp in milliseconds.
@property (nonatomic, readonly) NSTimeInterval timestamp; //stored in JSON as long

// Required: body
// The key-value pairs for the telemetry data point. See "body" key below.
// If type above is "log", body should contain "message" key.
// If type above is "network", body should contain "method", "url", and "status_code" keys.
// If type above is "dom", body should contain "element" key.
// If type above is "navigation", body should contain "from" and "to" keys.
// If type above is "error", body should contain "message" key.
@property (nonatomic, strong, readonly) RollbarTelemetryBody *body;

#pragma mark - Initializers

- (instancetype)initWithLevel:(RollbarLevel)level
                telemetryType:(RollbarTelemetryType)type
                       source:(RollbarSource)source;
//NS_DESIGNATED_INITIALIZER;

- (instancetype)initWithLevel:(RollbarLevel)level
                       source:(RollbarSource)source
                         body:(nonnull RollbarTelemetryBody *)body;
//NS_DESIGNATED_INITIALIZER;

- (instancetype)initWithArray:(NSArray *)data
NS_UNAVAILABLE;

- (instancetype)initWithDictionary:(NSDictionary *)data
NS_DESIGNATED_INITIALIZER;

- (instancetype)init
NS_UNAVAILABLE;

#pragma mark - Class utility

+ (nullable RollbarTelemetryBody *)createTelemetryBodyWithType:(RollbarTelemetryType)type
                                                          data:(nullable NSDictionary *)data;

@end

NS_ASSUME_NONNULL_END
