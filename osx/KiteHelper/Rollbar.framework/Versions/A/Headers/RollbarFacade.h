//  Copyright (c) 2018 Rollbar, Inc. All rights reserved.

#import <Foundation/Foundation.h>
#import "RollbarLevel.h"
#import "RollbarTelemetry.h"
#import "RollbarTelemetryType.h"

@class RollbarConfiguration;
@class RollbarNotifier;

@interface Rollbar : NSObject

#pragma mark - Class Initializers

+ (void)initWithAccessToken:(NSString*)accessToken;

+ (void)initWithAccessToken:(NSString*)accessToken
              configuration:(RollbarConfiguration*)configuration;

+ (void)initWithAccessToken:(NSString*)accessToken
              configuration:(RollbarConfiguration*)configuration
        enableCrashReporter:(BOOL)enable;

#pragma mark - Shared/global notifier

+ (RollbarNotifier*)currentNotifier;

#pragma mark - Configuration

+ (RollbarConfiguration*)currentConfiguration;

+ (void)updateConfiguration:(RollbarConfiguration*)configuration
                     isRoot:(BOOL)isRoot;



#pragma mark - New logging methods

+ (void)log:(RollbarLevel)level
    message:(NSString*)message;
+ (void)log:(RollbarLevel)level
    message:(NSString*)message
  exception:(NSException*)exception;
+ (void)log:(RollbarLevel)level
    message:(NSString*)message
  exception:(NSException*)exception
       data:(NSDictionary*)data;
+ (void)log:(RollbarLevel)level
    message:(NSString*)message
  exception:(NSException*)exception
       data:(NSDictionary*)data
    context:(NSString*)context;

+ (void)debug:(NSString*)message;
+ (void)debug:(NSString*)message
    exception:(NSException*)exception;
+ (void)debug:(NSString*)message
    exception:(NSException*)exception
         data:(NSDictionary*)data;
+ (void)debug:(NSString*)message
    exception:(NSException*)exception
         data:(NSDictionary*)data
      context:(NSString*)context;

+ (void)info:(NSString*)message;
+ (void)info:(NSString*)message
   exception:(NSException*)exception;
+ (void)info:(NSString*)message
   exception:(NSException*)exception
        data:(NSDictionary*)data;
+ (void)info:(NSString*)message
   exception:(NSException*)exception
        data:(NSDictionary*)data
     context:(NSString*)context;

+ (void)warning:(NSString*)message;
+ (void)warning:(NSString*)message
      exception:(NSException*)exception;
+ (void)warning:(NSString*)message
      exception:(NSException*)exception
           data:(NSDictionary*)data;
+ (void)warning:(NSString*)message
      exception:(NSException*)exception
           data:(NSDictionary*)data
        context:(NSString*)context;

+ (void)error:(NSString*)message;
+ (void)error:(NSString*)message
    exception:(NSException*)exception;
+ (void)error:(NSString*)message
    exception:(NSException*)exception
         data:(NSDictionary*)data;
+ (void)error:(NSString*)message
    exception:(NSException*)exception
         data:(NSDictionary*)data
      context:(NSString*)context;

+ (void)critical:(NSString*)message;
+ (void)critical:(NSString*)message
       exception:(NSException*)exception;
+ (void)critical:(NSString*)message
       exception:(NSException*)exception
            data:(NSDictionary*)data;
+ (void)critical:(NSString*)message
       exception:(NSException*)exception
            data:(NSDictionary*)data
         context:(NSString*)context;

+ (void)logCrashReport:(NSString*)crashReport;

#pragma mark - Send manually constructed JSON payload

+ (void)sendJsonPayload:(NSData*)payload;

#pragma mark - Telemetry API

+ (void)recordViewEventForLevel:(RollbarLevel)level
                        element:(NSString *)element;
+ (void)recordViewEventForLevel:(RollbarLevel)level
                        element:(NSString *)element
                      extraData:(NSDictionary *)extraData;

+ (void)recordNetworkEventForLevel:(RollbarLevel)level
                            method:(NSString *)method
                               url:(NSString *)url
                        statusCode:(NSString *)statusCode;
+ (void)recordNetworkEventForLevel:(RollbarLevel)level
                            method:(NSString *)method
                               url:(NSString *)url
                        statusCode:(NSString *)statusCode
                         extraData:(NSDictionary *)extraData;

+ (void)recordConnectivityEventForLevel:(RollbarLevel)level
                                 status:(NSString *)status;
+ (void)recordConnectivityEventForLevel:(RollbarLevel)level
                                 status:(NSString *)status
                              extraData:(NSDictionary *)extraData;

+ (void)recordErrorEventForLevel:(RollbarLevel)level
                         message:(NSString *)message;
+ (void)recordErrorEventForLevel:(RollbarLevel)level
                       exception:(NSException *)exception;
+ (void)recordErrorEventForLevel:(RollbarLevel)level
                         message:(NSString *)message
                       extraData:(NSDictionary *)extraData;

+ (void)recordNavigationEventForLevel:(RollbarLevel)level
                                 from:(NSString *)from
                                   to:(NSString *)to;
+ (void)recordNavigationEventForLevel:(RollbarLevel)level
                                 from:(NSString *)from
                                   to:(NSString *)to
                            extraData:(NSDictionary *)extraData;

+ (void)recordManualEventForLevel:(RollbarLevel)level
                         withData:(NSDictionary *)extraData;

#pragma mark - DEPRECATED old logging methods, for backward compatibility

+ (void)logWithLevel:(NSString*)level
             message:(NSString*)message;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use log:message: method instead.");

+ (void)logWithLevel:(NSString*)level
             message:(NSString*)message
                data:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use log:message:exception: method instead.");

+ (void)logWithLevel:(NSString*)level
             message:(NSString*)message
                data:(NSDictionary*)data
             context:(NSString*)context;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use log:message:exception:data:context: method instead.");

+ (void)logWithLevel:(NSString*)level
                data:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use log:message:exception:data: method instead.");

+ (void)debugWithMessage:(NSString*)message;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use debug:... methods instead.");

+ (void)debugWithMessage:(NSString*)message
                    data:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use debug:... methods instead.");

+ (void)debugWithData:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use debug:... methods instead.");


+ (void)infoWithMessage:(NSString*)message;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use info:... methods instead.");

+ (void)infoWithMessage:(NSString*)message
                   data:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use info:... methods instead.");

+ (void)infoWithData:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use info:... methods instead.");

+ (void)warningWithMessage:(NSString*)message;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use warning:... methods instead.");

+ (void)warningWithMessage:(NSString*)message
                      data:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use warning:... methods instead.");

+ (void)warningWithData:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use warning:... methods instead.");

+ (void)errorWithMessage:(NSString*)message;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use error:... methods instead.");

+ (void)errorWithMessage:(NSString*)message
                    data:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use error:... methods instead.");

+ (void)errorWithData:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use error:... methods instead.");

+ (void)criticalWithMessage:(NSString*)message;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use critical:... methods instead.");

+ (void)criticalWithMessage:(NSString*)message
                       data:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use critical:... methods instead.");

+ (void)criticalWithData:(NSDictionary*)data;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use critical:... methods instead.");

@end
