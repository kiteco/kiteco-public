//
//  RollbarServer.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-12-02.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>
#import "RollbarServerConfig.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarServer : RollbarServerConfig
// Can contain any arbitrary keys. Rollbar understands the following:

#pragma mark - Properties

// Optional: cpu
// A string up to 255 characters
@property (nonatomic, copy, nullable) NSString *cpu;

#pragma mark - Initializers

- (instancetype)initWithCpu:(nullable NSString *)cpu
                       host:(nullable NSString *)host
                       root:(nullable NSString *)root
                     branch:(nullable NSString *)branch
                codeVersion:(nullable NSString *)codeVersion;

- (instancetype)initWithCpu:(nullable NSString *)cpu
               serverConfig:(nullable RollbarServerConfig *)serverConfig;

@end

NS_ASSUME_NONNULL_END
