//  Copyright © 2018 Rollbar. All rights reserved.

#import <Foundation/Foundation.h>

#pragma mark - RollbarLevel

typedef NS_ENUM(NSUInteger, RollbarLevel) {
    RollbarInfo,
    RollbarDebug,
    RollbarWarning,
    RollbarCritical,
    RollbarError
};

#pragma mark - RollbarLevel utility

NS_ASSUME_NONNULL_BEGIN

/// RollbarLevel utility
@interface RollbarLevelUtil : NSObject

/// Converts RollbarLevel enum value to its string equivalent or default string.
/// @param value RollbarLevel enum value
+ (NSString *) RollbarLevelToString:(RollbarLevel)value;

/// Converts string value into its  RollbarLevel enum value equivalent or default enum value.
/// @param value input string
+ (RollbarLevel) RollbarLevelFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END

#pragma mark - deprecated

NSString* _Nonnull RollbarStringFromLevel(RollbarLevel level);
//DEPRECATED_MSG_ATTRIBUTE("In v2, use [RollbarLevelUtil RollbarLevelToString:...] methods instead.");

RollbarLevel RollbarLevelFromString(NSString * _Nonnull levelString);
//DEPRECATED_MSG_ATTRIBUTE("In v2, use [RollbarLevelUtil RollbarLevelFromString:...] methods instead.");

