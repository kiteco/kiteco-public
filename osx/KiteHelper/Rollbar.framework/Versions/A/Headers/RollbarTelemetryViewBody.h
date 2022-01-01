//
//  RollbarTelemetryViewBody.h
//  Rollbar
//
//  Created by Andrey Kornich on 2020-02-28.
//  Copyright © 2020 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

#import "RollbarTelemetryBody.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarTelemetryViewBody : RollbarTelemetryBody

#pragma mark - Properties

@property (nonatomic, copy) NSString *element;

#pragma mark - Initializers

-(instancetype)initWithElement:(nonnull NSString *)element
                     extraData:(nullable NSDictionary *)extraData
NS_DESIGNATED_INITIALIZER;

-(instancetype)initWithElement:(nonnull NSString *)element;

- (instancetype)initWithArray:(NSArray *)data
NS_DESIGNATED_INITIALIZER;

- (instancetype)initWithDictionary:(NSDictionary *)data
NS_DESIGNATED_INITIALIZER;

- (instancetype)init
NS_UNAVAILABLE;

@end

NS_ASSUME_NONNULL_END
