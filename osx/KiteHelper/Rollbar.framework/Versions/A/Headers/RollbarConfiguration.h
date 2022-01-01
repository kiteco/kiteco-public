//  Copyright (c) 2018 Rollbar, Inc. All rights reserved.

#import <Foundation/Foundation.h>
#import "RollbarLevel.h"
#import "CaptureIpType.h"

@class RollbarConfig;
@class RollbarData;

//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarConfig class instead.")
@interface RollbarConfiguration : NSObject

+ (RollbarConfiguration*)configuration;

- (id)initWithLoadedConfiguration;

- (RollbarConfig *)asRollbarConfig;

#pragma mark - Persistence
- (void)_setRoot;
- (void)save;

#pragma mark - Custom data
- (NSDictionary *)customData;

#pragma mark - Rollbar project destination/endpoint
@property (nonatomic, copy) NSString *accessToken;
@property (nonatomic, copy) NSString *environment;
@property (nonatomic, copy) NSString *endpoint;

#pragma mark - Developer options
@property (nonatomic) BOOL enabled;
@property (nonatomic) BOOL transmit;
@property (nonatomic) BOOL logPayload;
@property (nonatomic, copy) NSString *logPayloadFile;

#pragma mark - HTTP proxy
@property (nonatomic) BOOL httpProxyEnabled;
@property (nonatomic, copy) NSString *httpProxy;
@property (nonatomic) NSNumber *httpProxyPort;

#pragma mark - HTTPS proxy
@property (nonatomic) BOOL httpsProxyEnabled;
@property (nonatomic, copy) NSString *httpsProxy;
@property (nonatomic) NSNumber *httpsProxyPort;

#pragma mark - Logging options
@property (nonatomic) RollbarLevel rollbarCrashLevel;
@property (nonatomic) RollbarLevel rollbarLogLevel;
@property (nonatomic) NSUInteger maximumReportsPerMinute;
@property (nonatomic) CaptureIpType captureIp;
@property (nonatomic, copy) NSString *codeVersion;
@property (nonatomic, copy) NSString *framework;
// ID to link request between client/server
@property (nonatomic, copy) NSString *requestId;

#pragma mark - Payload scrubbing options
// Fields to scrub from the payload
@property (readonly, nonatomic, strong) NSSet *scrubFields;
- (void)addScrubField:(NSString *)field;
- (void)removeScrubField:(NSString *)field;
// Fields to not scrub from the payload even if they mention among scrubFields:
@property (readonly, nonatomic, strong) NSSet *scrubSafeListFields;
- (void)addScrubSafeListField:(NSString *)field;
- (void)removeScrubSafeListField:(NSString *)field;

#pragma mark - Server
@property (nonatomic, copy) NSString *serverHost;
@property (nonatomic, copy) NSString *serverRoot;
@property (nonatomic, copy) NSString *serverBranch;
@property (nonatomic, copy) NSString *serverCodeVersion;

#pragma mark - Person/user tracking
@property (nonatomic, copy) NSString *personId;
@property (nonatomic, copy) NSString *personUsername;
@property (nonatomic, copy) NSString *personEmail;

#pragma mark - Notifier
@property (nonatomic, copy) NSString *notifierName;
@property (nonatomic, copy) NSString *notifierVersion;

#pragma mark - Telemetry:
@property (nonatomic) BOOL telemetryEnabled;
@property (nonatomic) NSInteger maximumTelemetryEvents;
@property (nonatomic) BOOL captureLogAsTelemetryEvents;
@property (nonatomic) BOOL shouldCaptureConnectivity;
@property (nonatomic) BOOL scrubViewInputsTelemetry;
@property (nonatomic, strong) NSMutableSet *telemetryViewInputsToScrub;
- (void)addTelemetryViewInputToScrub:(NSString *)input;
- (void)removeTelemetryViewInputToScrub:(NSString *)input;


#pragma mark - Payload processing callbacks

// Decides whether or not to send provided payload data. Returns true to ignore, false to send
@property (nonatomic, copy) BOOL (^checkIgnoreRollbarData)(RollbarData *rollbarData);
// Modify payload data
@property (nonatomic, copy) RollbarData *(^modifyRollbarData)(RollbarData *rollbarData);

#pragma mark - DEPRECATED Payload processing callbacks



#pragma mark - Convenience Methods

- (void)setPersonId:(NSString*)personId
           username:(NSString*)username
              email:(NSString*)email;

- (void)setServerHost:(NSString *)host
                 root:(NSString*)root
               branch:(NSString*)branch
          codeVersion:(NSString*)codeVersion;

- (void)setNotifierName:(NSString *)name
                version:(NSString *)version;

#pragma mark - DEPRECATED

// Fields to not scrub from the payload even if they mention among scrubFields:
@property (readonly, nonatomic, strong) NSSet *scrubWhitelistFields;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use scrubSafeListFields property instead.");

- (void)addScrubWhitelistField:(NSString *)field;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use addScrubSafeListField method instead.");

- (void)removeScrubWhitelistField:(NSString *)field;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use removeScrubSafeListField method instead.");

// Decides whether or not to send payload. Returns true to ignore, false to send
@property (readonly, nonatomic, copy) BOOL (^checkIgnore)(NSDictionary *payload);
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use ^checkIgnoreRollbarData property instead.");

- (void)setCheckIgnoreBlock:(BOOL (^)(NSDictionary*))checkIgnoreBlock;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use ^checkIgnoreRollbarData property instead.");

// Modify payload
@property (readonly, nonatomic, copy) void (^payloadModification)(NSMutableDictionary *payload);
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use ^modifyRollbarData property instead.");

- (void)setPayloadModificationBlock:(void (^)(NSMutableDictionary*))payloadModificationBlock;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use ^modifyRollbarData property instead.");

@property (nonatomic, copy) NSString *crashLevel;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use rollbarCrashLevel property instead.");
@property (nonatomic, copy) NSString *logLevel;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use rollbarLogLevel property instead.");

- (void)setRollbarLevel:(RollbarLevel)level;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use rollbarLogLevel property instead.");
- (RollbarLevel)getRollbarLevel;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use rollbarLogLevel property instead.");

- (void)setReportingRate:(NSUInteger)maximumReportsPerMinute;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use maximumReportsPerMinute property instead.");

- (void)setCodeFramework:(NSString *)framework;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use framework property instead.");

- (void)setCodeVersion:(NSString *)codeVersion;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use codeVersion property instead.");

- (void)setRequestId:(NSString*)requestId;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use requestId property instead.");

- (void)setCaptureIpType:(CaptureIpType)captureIp;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use captureIp property instead.");

- (void)setMaximumTelemetryData:(NSInteger)maximumTelemetryData;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use maximumTelemetryEvents property instead.");

- (void)setCaptureLogAsTelemetryData:(BOOL)captureLog;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use captureLogAsTelemetryEvents property instead.");

- (void)setCaptureConnectivityAsTelemetryData:(BOOL)captureConnectivity;
//    DEPRECATED_MSG_ATTRIBUTE("In v2, use shouldCaptureConnectivity property instead.");


@end

