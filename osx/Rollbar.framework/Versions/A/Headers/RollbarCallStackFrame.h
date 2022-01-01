//
//  RollbarCallStackFrame.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-11-27.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

@class RollbarCallStackFrameContext;

NS_ASSUME_NONNULL_BEGIN

@interface RollbarCallStackFrame : DataTransferObject

#pragma mark - Required properties

// Required: filename
// The filename including its full path.
@property (nonatomic, copy, nonnull) NSString *filename;

#pragma mark - Optional properties

// Optional: lineno
// The line number as an integer
@property (nonatomic, nullable) NSNumber *lineno;

// Optional: colno
// The column number as an integer
@property (nonatomic, nullable) NSNumber *colno;

// Optional: method
// The method or function name
@property (nonatomic, copy, nullable) NSString *method;

// Optional: code
// The line of code
@property (nonatomic, copy, nullable) NSString *code;

// Optional: class_name
// A string containing the class name.
// Used in the UI when the payload's top-level "language" key has the value "java"
@property (nonatomic, copy, nullable) NSString *className;

// Optional: context
// Additional code before and after the "code" line
@property (nonatomic, nullable) RollbarCallStackFrameContext *context;

// Optional: argspec
// List of the names of the arguments to the method/function call.
@property (nonatomic, nullable) NSArray<NSString *> *argspec;

// Optional: varargspec
// If the function call takes an arbitrary number of unnamed positional arguments,
// the name of the argument that is the list containing those arguments.
// For example, in Python, this would typically be "args" when "*args" is used.
// The actual list will be found in locals.
@property (nonatomic, nullable) NSArray<NSString *> *varargspec;

// Optional: keywordspec
// If the function call takes an arbitrary number of keyword arguments, the name
// of the argument that is the object containing those arguments.
// For example, in Python, this would typically be "kwargs" when "**kwargs" is used.
// The actual object will be found in locals.
@property (nonatomic, nullable) NSArray<NSString *> *keywordspec;

// Optional: locals
// Object of local variables for the method/function call.
// The values of variables from argspec, vararspec and keywordspec
// can be found in locals.
@property (nonatomic, nullable) NSDictionary *locals;

#pragma mark - Initializers

-(instancetype)initWithFileName:(nonnull NSString *)filename;

@end

NS_ASSUME_NONNULL_END
