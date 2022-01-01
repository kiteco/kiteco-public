//
//  RollbarTelemetryNavigationBody.h
//  Rollbar
//
//  Created by Andrey Kornich on 2020-02-28.
//  Copyright © 2020 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#import "RollbarTelemetryBody.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarTelemetryNavigationBody : RollbarTelemetryBody

#pragma mark - Properties

@property (nonatomic, copy) NSString *from;
@property (nonatomic, copy) NSString *to;

#pragma mark - Initializers

-(instancetype)initWithFromLocation:(nonnull NSString *)from
                         toLocation:(nonnull NSString *)to
                          extraData:(nullable NSDictionary *)extraData
NS_DESIGNATED_INITIALIZER;

-(instancetype)initWithFromLocation:(nonnull NSString *)from
                         toLocation:(nonnull NSString *)to;

- (instancetype)initWithArray:(NSArray *)data
NS_DESIGNATED_INITIALIZER;

- (instancetype)initWithDictionary:(NSDictionary *)data
NS_DESIGNATED_INITIALIZER;

- (instancetype)init
NS_UNAVAILABLE;

@end

NS_ASSUME_NONNULL_END
