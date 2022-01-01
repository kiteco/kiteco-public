//
//  RollbarServer.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-24.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarServerConfig : DataTransferObject

#pragma mark - properties

@property (nonatomic, copy, nullable) NSString *host;
@property (nonatomic, copy, nullable) NSString *root;
@property (nonatomic, copy, nullable) NSString *branch;
@property (nonatomic, copy, nullable) NSString *codeVersion;

#pragma mark - initializers

- (instancetype)initWithHost:(nullable NSString *)host
                        root:(nullable NSString *)root
                      branch:(nullable NSString *)branch
                 codeVersion:(nullable NSString *)codeVersion;

@end

NS_ASSUME_NONNULL_END
