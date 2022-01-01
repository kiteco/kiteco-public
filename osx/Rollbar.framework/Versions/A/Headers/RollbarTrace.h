//
//  RollbarTrace.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-11-27.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

@class RollbarCallStackFrame;
@class RollbarException;

NS_ASSUME_NONNULL_BEGIN

@interface RollbarTrace : DataTransferObject

#pragma mark - Properties

// Required: frames
// A list of stack frames, ordered such that the most recent call is last in the list.
@property (nonatomic, nonnull) NSArray<RollbarCallStackFrame *> *frames;

// Required: exception
// An object describing the exception instance.
@property (nonatomic, nonnull) RollbarException *exception;

#pragma mark - Initializers

-(instancetype)initWithRollbarException:(nonnull RollbarException *)exception
                 rollbarCallStackFrames:(nonnull NSArray<RollbarCallStackFrame *> *)frames;

-(instancetype)initWithException:(nonnull NSException *)exception;

@end

NS_ASSUME_NONNULL_END
