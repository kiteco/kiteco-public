//  Copyright © 2018 Rollbar. All rights reserved.

#import <Foundation/Foundation.h>
#import "DataTransferObject.h"

/// Models a Deployment
//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarDeployment instead.")
@interface Deployment : DataTransferObject

#pragma mark - properties

/// Rollbar project environment
@property (readonly, retain) NSString *environment;

/// Comment
@property (readonly, retain) NSString *comment;

/// Revision ID
@property (readonly, retain) NSString *revision;

/// Local user's name
@property (readonly, retain) NSString *localUsername;

/// Rollbar user's name
@property (readonly, retain) NSString *rollbarUsername;

#pragma mark - initializers

/// Designated initializer
/// @param environment Rollbar project environment
/// @param comment Comment
/// @param revision Revision ID
/// @param localUserName local user's name
/// @param rollbarUserName Rollbar user's name
- (instancetype)initWithEnvironment:(NSString *)environment
                            comment:(NSString *)comment
                           revision:(NSString *)revision
                      localUserName:(NSString *)localUserName
                    rollbarUserName:(NSString *)rollbarUserName
NS_DESIGNATED_INITIALIZER;

/// Initialize this DTO instance with valid JSON NSArray seed
/// @param data valid JSON NSArray seed
- (instancetype)initWithArray:(NSArray *)data NS_UNAVAILABLE;

/// Initialize empty DTO
- (instancetype)init NS_UNAVAILABLE;

@end
