//
//  RollbarAppLanguage.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-12-16.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

typedef NS_ENUM(NSUInteger, RollbarAppLanguage) {
    ObjectiveC,
    ObjectiveCpp,
    Swift,
    C,
    Cpp,
};

/// Utility class aiding with RollbarAppLanguage conversions
@interface RollbarAppLanguageUtil : NSObject

/// Convert RollbarAppLanguage to a string
/// @param value RollbarAppLanguage value
+ (NSString *) RollbarAppLanguageToString:(RollbarAppLanguage)value;

/// Convert RollbarAppLanguage value from a string
/// @param value string representation of a RollbarAppLanguage value
+ (RollbarAppLanguage) RollbarAppLanguageFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END
