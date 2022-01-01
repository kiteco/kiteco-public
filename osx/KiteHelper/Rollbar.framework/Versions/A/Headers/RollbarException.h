//
//  RollbarException.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-11-27.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarException : DataTransferObject

#pragma mark - Properties

// Required: class
// The exception class name.
@property (nonatomic, copy, nonnull) NSString *exceptionClass;

// Optional: message
// The exception message, as a string
@property (nonatomic, copy, nullable) NSString *exceptionMessage;

// Optional: description
// An alternate human-readable string describing the exception
// Usually the original exception message will have been machine-generated;
// you can use this to send something custom
@property (nonatomic, copy, nullable) NSString *exceptionDescription;

#pragma mark - Initializers

- (instancetype)initWithExceptionClass:(nonnull NSString *)exceptionClass
                      exceptionMessage:(nullable NSString *)exceptionMessage
                  exceptionDescription:(nullable NSString *)exceptionDescription;

@end

NS_ASSUME_NONNULL_END
