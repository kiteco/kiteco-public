//
//  RollbarScrubbingOptions.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-24.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarScrubbingOptions : DataTransferObject

#pragma mark - properties

@property (nonatomic) BOOL enabled;
// Fields to scrub from the payload
@property (nonatomic, strong) NSArray *scrubFields;
// Fields to not scrub from the payload even if they mention among scrubFields:
@property (nonatomic, strong) NSArray *safeListFields;


#pragma mark - initializers

- (instancetype)initWithEnabled:(BOOL)enabled
                    scrubFields:(NSArray *)scrubFields
                 safeListFields:(NSArray *)safeListFields;
- (instancetype)initWithScrubFields:(NSArray *)scrubFields
                     safeListFields:(NSArray *)safeListFields;
- (instancetype)initWithScrubFields:(NSArray *)scrubFields;

#pragma mark - DEPRECATED

// Fields to not scrub from the payload even if they mention among scrubFields:
@property (nonatomic, strong) NSArray *whitelistFields;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use safeListFields property instead.");

- (instancetype)initWithEnabled:(BOOL)enabled
                    scrubFields:(NSArray *)scrubFields
                whitelistFields:(NSArray *)whitelistFields;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use initWithEnabled:scrubFields:safeListFields: method instead.");

- (instancetype)initWithScrubFields:(NSArray *)scrubFields
                    whitelistFields:(NSArray *)whitelistFields;
//DEPRECATED_MSG_ATTRIBUTE("In v2, use initWithEnabled:safeListFields: method instead.");

@end

NS_ASSUME_NONNULL_END
