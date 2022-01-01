//
//  RollbarPerson.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-25.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarPerson : DataTransferObject

#pragma mark - properties

// Required: id
// A string up to 40 characters identifying this user in your system.
@property (nonatomic, copy, nonnull) NSString *ID;

// Optional: username
// A string up to 255 characters
@property (nonatomic, copy, nullable) NSString *username;

// Optional: email
// A string up to 255 characters
@property (nonatomic, copy, nullable) NSString *email;

#pragma mark - initializers

- (instancetype)initWithID:(nonnull NSString *)ID
                  username:(nullable NSString *)username
                     email:(nullable NSString *)email;
- (instancetype)initWithID:(nonnull NSString *)ID
                  username:(nullable NSString *)username;
- (instancetype)initWithID:(nonnull NSString *)ID
                     email:(nullable NSString *)email;
- (instancetype)initWithID:(nonnull NSString *)ID;

@end

NS_ASSUME_NONNULL_END
