//
//  RollbarDeveloperOptions.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-23.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"

NS_ASSUME_NONNULL_BEGIN

@interface RollbarDeveloperOptions : DataTransferObject

#pragma mark - properties

@property (nonatomic) BOOL enabled;
@property (nonatomic) BOOL transmit;
@property (nonatomic) BOOL logPayload;
@property (nonatomic, copy) NSString *payloadLogFile;

#pragma mark - initializers

- (instancetype)initWithEnabled:(BOOL)enabled
                       transmit:(BOOL)transmit
                     logPayload:(BOOL)logPayload
                 payloadLogFile:(NSString *)logPayloadFile;
- (instancetype)initWithEnabled:(BOOL)enabled
                       transmit:(BOOL)transmit
                     logPayload:(BOOL)logPayload;
- (instancetype)initWithEnabled:(BOOL)enabled;

@end

NS_ASSUME_NONNULL_END
