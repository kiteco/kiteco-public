//
//  Persistent.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-09.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

/// A protocol adding support for file-persistence
//DEPRECATED_MSG_ATTRIBUTE("In v2, use RollbarPersistent instead.")
@protocol Persistent <NSObject>

/// Save to a file
/// @param filePath file path to save to
- (BOOL)saveToFile:(NSString *)filePath;

/// Load object state/data from a file
/// @param filePath file path to load from
- (BOOL)loadFromFile:(NSString *)filePath;

@end

NS_ASSUME_NONNULL_END
