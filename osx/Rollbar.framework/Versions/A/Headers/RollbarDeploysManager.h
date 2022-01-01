//  Copyright © 2018 Rollbar. All rights reserved.

#import <Foundation/Foundation.h>
#import "RollbarDeploysProtocol.h"

/// Rollbar Deploys Manager (a facade client to the Rollbar Deploy APIs)
@interface RollbarDeploysManager : NSObject <RollbarDeploysProtocol> {
}

/// Designated initializer
/// @param writeAccessToken write AccessToken
/// @param readAccessToken read AccessToken
/// @param deploymentRegistrationObserver deployment registration observer
/// @param deploymentDetailsObserver deployment details observer
/// @param deploymentDetailsPageObserver deployment details page observer
- (instancetype)initWithWriteAccessToken:(NSString *)writeAccessToken
                         readAccessToken:(NSString *)readAccessToken
          deploymentRegistrationObserver:(NSObject<DeploymentRegistrationObserver>*)deploymentRegistrationObserver
               deploymentDetailsObserver:(NSObject<DeploymentDetailsObserver>*)deploymentDetailsObserver
           deploymentDetailsPageObserver:(NSObject<DeploymentDetailsPageObserver>*)deploymentDetailsPageObserver
NS_DESIGNATED_INITIALIZER;

@end
