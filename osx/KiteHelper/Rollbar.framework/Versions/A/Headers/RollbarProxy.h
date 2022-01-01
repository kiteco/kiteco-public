//
//  RollbarProxy.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-24.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarProxy : DataTransferObject

#pragma mark - properties
@property (nonatomic) BOOL enabled;
@property (nonatomic, copy) NSString *proxyUrl;
@property (nonatomic) NSUInteger proxyPort;

#pragma mark - initializers

- (instancetype)initWithEnabled:(BOOL)enabled
                       proxyUrl:(NSString *)proxyUrl
                      proxyPort:(NSUInteger)proxyPort;

@end

NS_ASSUME_NONNULL_END
