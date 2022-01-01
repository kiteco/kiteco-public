//
//  HttpMethod.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-12-02.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#pragma mark - HttpMethod enum

//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod enum instead.")
typedef NS_ENUM(NSUInteger, HttpMethod) {
    Head,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Head instead."),
    Get,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Get instead."),
    Post,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Post instead."),
    Put,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Put instead."),
    Patch,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Patch instead."),
    Delete,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Delete instead."),
    Connect,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Connect instead."),
    Options,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Options instead."),
    Trace// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethod_Trace instead."),
};

#pragma mark - CaptureIpTypeUtil

NS_ASSUME_NONNULL_BEGIN

//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarHttpMethodUtil class instead.")
@interface HttpMethodUtil : NSObject

/// Convert HttpMethod to a string
/// @param value CaptureIpType value
+ (NSString *) HttpMethodToString:(HttpMethod)value;

/// Convert HttpMethod value from a string
/// @param value string representation of a CaptureIpType value
+ (HttpMethod) HttpMethodFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END
