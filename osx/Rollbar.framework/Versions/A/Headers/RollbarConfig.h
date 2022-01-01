//
//  RollbarConfig.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-11.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"
#import "CaptureIpType.h"
#import "RollbarLevel.h"

@class RollbarDestination;
@class RollbarDeveloperOptions;
@class RollbarProxy;
@class RollbarScrubbingOptions;
@class RollbarServerConfig;
@class RollbarPerson;
@class RollbarModule;
@class RollbarTelemetryOptions;
@class RollbarLoggingOptions;

NS_ASSUME_NONNULL_BEGIN

@interface RollbarConfig : DataTransferObject

#pragma mark - properties
@property (nonatomic, strong) RollbarDestination *destination;
@property (nonatomic, strong) RollbarDeveloperOptions *developerOptions;
@property (nonatomic, strong) RollbarLoggingOptions *loggingOptions;
@property (nonatomic, strong) RollbarProxy *httpProxy;
@property (nonatomic, strong) RollbarProxy *httpsProxy;
@property (nonatomic, strong) RollbarScrubbingOptions *dataScrubber;
@property (nonatomic, strong) RollbarServerConfig *server;
@property (nonatomic, strong) RollbarPerson *person;
@property (nonatomic, strong) RollbarModule *notifier;
@property (nonatomic, strong) RollbarTelemetryOptions *telemetry;

#pragma mark - Custom data
@property (nonatomic, strong) NSDictionary *customData;


#pragma mark - Payload Content Related
// Payload content related:
// ========================
// Decides whether or not to send payload. Returns true to ignore, false to send
//@property (nonatomic, copy) BOOL (^checkIgnore)(NSDictionary *payload);
// Modify payload
//@property (nonatomic, copy) void (^payloadModification)(NSMutableDictionary *payload);

#pragma mark - Convenience Methods (remove from here and only keep them within RollbarConfiguration)
- (void)setPersonId:(NSString*)personId
           username:(NSString*)username
              email:(NSString*)email;
- (void)setServerHost:(NSString *)host
                 root:(NSString*)root
               branch:(NSString*)branch
          codeVersion:(NSString*)codeVersion;
- (void)setNotifierName:(NSString *)name
                version:(NSString *)version;


@end

NS_ASSUME_NONNULL_END
