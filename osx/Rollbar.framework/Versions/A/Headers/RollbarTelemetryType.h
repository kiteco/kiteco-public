//  Copyright (c) 2018 Rollbar, Inc. All rights reserved.

#import <Foundation/Foundation.h>

#pragma mark - RollbarTelemetryType

typedef NS_ENUM(NSUInteger, RollbarTelemetryType) {
    RollbarTelemetryLog,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTelemetryType_Log instead."),
    RollbarTelemetryView,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTelemetryType_View instead."),
    RollbarTelemetryError,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTelemetryType_Error instead."),
    RollbarTelemetryNavigation,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTelemetryType_Navigation instead."),
    RollbarTelemetryNetwork,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTelemetryType_Network instead."),
    RollbarTelemetryConnectivity,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTelemetryType_Connectivity instead."),
    RollbarTelemetryManual// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTelemetryType_Manual instead.")
};


#pragma mark - RollbarLevel utility

NS_ASSUME_NONNULL_BEGIN

/// RollbarTelemetryType utility
@interface RollbarTelemetryTypeUtil : NSObject

/// Converts RollbarTelemetryType enum value to its string equivalent or default string.
/// @param value RollbarTelemetryType enum value
+ (NSString *) RollbarTelemetryTypeToString:(RollbarTelemetryType)value;

/// Converts string value into its  RollbarTelemetryType enum value equivalent or default enum value.
/// @param value input string
+ (RollbarTelemetryType) RollbarTelemetryTypeFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END

#pragma mark - deprecated

NSString* _Nonnull RollbarStringFromTelemetryType(RollbarTelemetryType type);
//DEPRECATED_MSG_ATTRIBUTE("In v2, use [RollbarTelemetryTypeUtil RollbarTelemetryTypeToString:...] methods instead.");

