//
//  RollbarModule.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-25.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarModule : DataTransferObject

#pragma mark - properties

// Optional: name
// Name of the library
@property (nonatomic, copy, nullable) NSString *name;

// Optional: version
// Library version string
@property (nonatomic, copy, nullable) NSString *version;

#pragma mark - initializers

- (instancetype)initWithName:(nullable NSString *)name
                     version:(nullable NSString *)version;

- (instancetype)initWithName:(nullable NSString *)name;

@end

NS_ASSUME_NONNULL_END
