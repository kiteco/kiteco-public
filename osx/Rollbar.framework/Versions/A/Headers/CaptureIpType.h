//
//  CaptureIpType.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-15.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#pragma mark - CaptureIpType enum

//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarCaptureIpType enum instead.")
typedef NS_ENUM(NSUInteger, CaptureIpType) {
    CaptureIpFull,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarCaptureIpType_Full instead."),
    CaptureIpAnonymize,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarCaptureIpType_Anonymize instead."),
    CaptureIpNone// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarCaptureIpType_None instead.")
};

#pragma mark - CaptureIpTypeUtil

NS_ASSUME_NONNULL_BEGIN

/// Utility class aiding with CaptureIpType conversions
//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarCaptureIpTypeUtil class instead.")
@interface CaptureIpTypeUtil : NSObject

/// Convert CaptureIpType to a string
/// @param value CaptureIpType value
+ (NSString *) CaptureIpTypeToString:(CaptureIpType)value;

/// Convert CaptureIpType value from a string
/// @param value string representation of a CaptureIpType value
+ (CaptureIpType) CaptureIpTypeFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END
