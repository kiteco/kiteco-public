//  Copyright © 2018 Rollbar. All rights reserved.

#import <Foundation/Foundation.h>
#import "DeploymentDetails.h"
#import "DataTransferObject.h"
#import "DeployApiCallOutcome.h"

NS_ASSUME_NONNULL_BEGIN

#pragma mark - DeployApiCallResult

/// Models result of Deploy API call/request
//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarDeployApiCallResult instead.")
@interface DeployApiCallResult : DataTransferObject

/// API call's outcome
@property (readonly) DeployApiCallOutcome outcome;

/// API call's result description
@property (readonly, copy, nullable) NSString *description;

/// Initialize this DTO instance with valid JSON NSDictionary seed
/// @param data valid JSON NSDictionary seed
- (instancetype)initWithDictionary:(NSDictionary *)data NS_UNAVAILABLE;

/// Initialize this DTO instance with valid JSON NSArray seed
/// @param data valid JSON NSArray seed
- (instancetype)initWithArray:(NSArray *)data NS_UNAVAILABLE;

/// Initialize empty DTO
- (instancetype)init NS_UNAVAILABLE;

/// Designated initializer
/// @param httpResponse HTTP response object
/// @param extraResponseData extra response info
/// @param error error (if any)
/// @param request corresponding HTTP request
- (instancetype)initWithResponse:(nullable NSHTTPURLResponse *)httpResponse
               extraResponseData:(nullable id)extraResponseData
                           error:(nullable NSError *)error
                      forRequest:(nonnull NSURLRequest *)request
NS_DESIGNATED_INITIALIZER;

/// Convenience initializer
/// @param httpResponse HTTP response object
/// @param data extra response data
/// @param error error (if any)
/// @param request  corresponding HTTP request
- (instancetype)initWithResponse:(nullable NSHTTPURLResponse *)httpResponse
                            data:(nullable NSData *)data
                           error:(nullable NSError *)error
                      forRequest:(nonnull NSURLRequest *)request;

@end

#pragma mark - DeploymentRegistrationResult

/// Models result of a deployment registration request
@interface DeploymentRegistrationResult : DeployApiCallResult

/// Deployment ID
@property (readonly, copy, nonnull) NSString *deploymentId;

@end

#pragma mark - DeploymentDetailsResult

/// Models result of a deployment details request
@interface DeploymentDetailsResult : DeployApiCallResult

/// Deployment details object
@property (readonly, retain, nullable) DeploymentDetails *deployment;

@end

#pragma mark - DeploymentDetailsPageResult

/// Models result of a deployment details page request
@interface DeploymentDetailsPageResult : DeployApiCallResult

/// Deployment details objects
@property (readonly, retain, nullable) NSArray<DeploymentDetails *> *deployments;

/// Deployment details page number
@property (readonly) NSUInteger pageNumber;

@end

NS_ASSUME_NONNULL_END
