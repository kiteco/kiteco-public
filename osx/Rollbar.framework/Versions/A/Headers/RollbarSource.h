//
//  RollbarSource.h
//  Rollbar
//
//  Created by Andrey Kornich on 2020-02-28.
//  Copyright © 2020 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#pragma mark - RollbarSource enum

typedef NS_ENUM(NSUInteger, RollbarSource) {
    
    Client,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarSource_Client instead."),
    Server// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarSource_Server instead."),
};

#pragma mark - RollbarSourceUtil

NS_ASSUME_NONNULL_BEGIN

@interface RollbarSourceUtil : NSObject

/// Convert RollbarSource to a string
/// @param value RollbarSource value
+ (NSString *) RollbarSourceToString:(RollbarSource)value;

/// Convert RollbarSource value from a string
/// @param value string representation of a RollbarSource value
+ (RollbarSource) RollbarSourceFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END
