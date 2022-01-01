//
//  DeployApiCallOutcome.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-11-08.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarDeployApiCallOutcome instead.")
typedef NS_ENUM(NSInteger, DeployApiCallOutcome) {
    DeployApiCallSuccess,// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarDeployApiCall_Success instead."),
    DeployApiCallError// DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarDeployApiCall_Error instead."),
};

NS_ASSUME_NONNULL_BEGIN

/// Enum to/from NSString conversion utility
//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarDeployApiCallOutcomeUtil instead.")
@interface DeployApiCallOutcomeUtil : NSObject

/// Converts DeployApiCallOutcome value into a NSString
/// @param value DeployApiCallOutcome value to convert
+ (NSString *) DeployApiCallOutcomeToString:(DeployApiCallOutcome)value;

/// Converts NSString into a DeployApiCallOutcome value
/// @param value NSString to convert
+ (DeployApiCallOutcome) DeployApiCallOutcomeFromString:(NSString *)value;

@end

NS_ASSUME_NONNULL_END
