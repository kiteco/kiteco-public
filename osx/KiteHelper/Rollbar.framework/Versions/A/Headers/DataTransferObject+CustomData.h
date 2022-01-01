//
//  DataTransferObject+CustomData.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-09.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

/// Adds custom data manipulation methods to a DTO
@interface DataTransferObject (CustomData)

#pragma mark - Non-safe operations

/// Add custom data value by a key
/// @param aKey the key
/// @param aValue the custom data
- (void)addKeyed:(NSString *)aKey DataTransferObject:(DataTransferObject *)aValue;

/// Add custom data value by a key
/// @param aKey the key
/// @param aValue the custom data
- (void)addKeyed:(NSString *)aKey String:(NSString *)aValue;

/// Add custom data value by a key
/// @param aKey the key
/// @param aValue the custom data
- (void)addKeyed:(NSString *)aKey Number:(NSNumber *)aValue;

/// Add custom data value by a key
/// @param aKey the key
/// @param aValue the custom data
- (void)addKeyed:(NSString *)aKey Array:(NSArray *)aValue;

/// Add custom data value by a key
/// @param aKey the key
/// @param aValue the custom data
- (void)addKeyed:(NSString *)aKey Dictionary:(NSDictionary *)aValue;

/// Add custom data value by a key
/// @param aKey the key
/// @param aValue the custom data
- (void)addKeyed:(NSString *)aKey Placeholder:(NSNull *)aValue;

#pragma mark - Safe operations

/// Tries adding a custom data by a key value
/// @param aKey the key to use
/// @param aValue the data to add
- (BOOL)tryAddKeyed:(NSString *)aKey Object:(NSObject *)aValue;

@end

NS_ASSUME_NONNULL_END
