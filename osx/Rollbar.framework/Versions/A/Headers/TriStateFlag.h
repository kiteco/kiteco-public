//
//  TriStateFlag.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-12-02.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#pragma mark - TriStateFlag enum

//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTriStateFlag enum instead.")
typedef NS_ENUM(NSUInteger, TriStateFlag) {
    None, //DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTriStateFlag_None instead."),
    On, //DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTriStateFlag_On instead."),
    Off //DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTriStateFlag_Off instead.")
};

#pragma mark - TriStateFlagUtil

NS_ASSUME_NONNULL_BEGIN

/// Utility class aiding with TriStateFlag conversions
//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarTriStateFlagUtil class instead.")
@interface TriStateFlagUtil : NSObject

/// Convert TriStateFlag to a string
/// @param value TriStateFlag value
+ (NSString *) TriStateFlagToString:(TriStateFlag)value;

/// Convert TriStateFlag value from a string
/// @param value string representation of a TriStateFlag value
+ (TriStateFlag) TriStateFlagFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END
