//
//  JSONSupport.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-09.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

/// JSON de/serialization protocol
//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarJSONSupport instead.")
@protocol JSONSupport <NSObject>

/// Internal JSON serializable "data store"
@property (readonly) NSMutableDictionary *jsonFriendlyData;

#pragma mark - via JSON-friendly NSData

/// Serialize into JSON-friendly NSData instance
- (NSData *)serializeToJSONData;

/// Desrialize from JSON-friendlt NSData instance
/// @param jsonData JSON-friendlt NSData instance
- (BOOL)deserializeFromJSONData:(NSData *)jsonData;

#pragma mark - via JSON string

/// Serialize into a JSON string
- (NSString *)serializeToJSONString;

/// Deserialize from a JSON string
/// @param jsonString JSON string
- (BOOL)deserializeFromJSONString:(NSString *)jsonString;

#pragma mark - Initializers

/// Initialize this DTO instance with valid JSON data string seed
/// @param jsonString valid JSON data string seed
- (instancetype)initWithJSONString:(NSString *)jsonString;

/// Initialize this DTO instance with valid JSON  NSData seed
/// @param data valid JSON NSData seed
- (instancetype)initWithJSONData:(NSData *)data;

/// Initialize this DTO instance with valid JSON NSDictionary seed
/// @param data valid JSON NSDictionary seed
- (instancetype)initWithDictionary:(NSDictionary *)data;

/// Initialize this DTO instance with valid JSON NSArray seed
/// @param data valid JSON NSArray seed
- (instancetype)initWithArray:(NSArray *)data;

@end

NS_ASSUME_NONNULL_END
