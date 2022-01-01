//
//  RollbarTelemetryLogBody.h
//  Rollbar
//
//  Created by Andrey Kornich on 2020-02-28.
//  Copyright © 2020 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#import "RollbarTelemetryBody.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarTelemetryLogBody : RollbarTelemetryBody

#pragma mark - Properties

@property (nonatomic, copy) NSString *message;

#pragma mark - Initializers

-(instancetype)initWithMessage:(nonnull NSString *)message
                     extraData:(nullable NSDictionary *)extraData
NS_DESIGNATED_INITIALIZER;

-(instancetype)initWithMessage:(nonnull NSString *)message;

- (instancetype)initWithArray:(NSArray *)data
NS_DESIGNATED_INITIALIZER;

- (instancetype)initWithDictionary:(NSDictionary *)data
NS_DESIGNATED_INITIALIZER;

- (instancetype)init
NS_UNAVAILABLE;

@end

NS_ASSUME_NONNULL_END
