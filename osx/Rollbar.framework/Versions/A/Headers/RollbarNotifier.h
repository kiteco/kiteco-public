//  Copyright (c) 2018 Rollbar, Inc. All rights reserved.

#import <Foundation/Foundation.h>

@class RollbarConfiguration;

@interface RollbarNotifier : NSObject 

/// Notifier's config object
@property (atomic, strong) RollbarConfiguration *configuration;

/// Disallowed initializer
- (instancetype)init
NS_UNAVAILABLE;

/// Designated notifier initializer
/// @param accessToken the access token
/// @param configuration the config object
/// @param isRoot the root notifier flag
- (instancetype)initWithAccessToken:(NSString*)accessToken
                      configuration:(RollbarConfiguration*)configuration
                             isRoot:(BOOL)isRoot
NS_DESIGNATED_INITIALIZER;

/// Processes persisted payloads
- (void)processSavedItems;

/// Captures a crash report
/// @param crashReport the crash report
- (void)logCrashReport:(NSString*)crashReport;

/// Captures a log entry
/// @param level Rollbar error/log level
/// @param message message
/// @param exception exception
/// @param data extra data
/// @param context extra context
- (void)log:(NSString*)level
    message:(NSString*)message
  exception:(NSException*)exception
       data:(NSDictionary*)data
    context:(NSString*)context;

/// Sends an item batch in a blocking manner.
/// @param payload an item to send
/// @param nextOffset the offset in the item queue file of the item immediately after this batch.
/// If the send is successful or the retry limit is hit, nextOffset will be saved to the queueState as the offset to use for the next batch
/// @return YES if this batch should be discarded if it was successful or a retry limit was hit. Otherwise NO is returned if this batch should be retried.
- (BOOL)sendItem:(NSDictionary*)payload
      nextOffset:(NSUInteger)nextOffset;


/// Sends a fully composed JSON payload.
/// @param payload complete Rollbar payload as JSON string
/// @return YES if successful. NO if not.
- (BOOL)sendPayload:(NSData*)payload;

/// Updates key configuration elements
/// @param accessToken the Rollbar project access token
/// @param configuration the Rollbar configuration object
/// @param isRoot the root notifier flag
- (void)updateAccessToken:(NSString*)accessToken
            configuration:(RollbarConfiguration *)configuration
                   isRoot:(BOOL)isRoot;

/// Updates key configuration elements
/// @param configuration the Rollbar configuration object
/// @param isRoot the root notifier flag
- (void)updateConfiguration:(RollbarConfiguration*)configuration
                     isRoot:(BOOL)isRoot;

/// Updates the Rollbar project access yoken
/// @param accessToken the Rollbar project access token
- (void)updateAccessToken:(NSString*)accessToken;

/// Updates allowed reporting rate
/// @param maximumReportsPerMinute the maximum allowed reports transmission rate
- (void)updateReportingRate:(NSUInteger)maximumReportsPerMinute;

@end
