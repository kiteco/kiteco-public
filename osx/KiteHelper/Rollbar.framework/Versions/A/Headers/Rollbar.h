//
//  Rollbar.h
//  Rollbar
//
//  Created by Andrey Kornich on 2020-01-17.
//  Copyright © 2020 Rollbar. All rights reserved.
//

#ifndef Rollbar_h
#define Rollbar_h

#import <Foundation/Foundation.h>

//#if TARGET_OS_IOS
//#import <UIKit/UIKit.h>
//#endif

//#if TARGET_OS_MACOS
//#import <Cocoa/Cocoa.h>
//#endif

//! Project version number for Rollbar.framework.
FOUNDATION_EXPORT double RollbarVersionNumber;

//! Project version string for Rollbar.framework.
FOUNDATION_EXPORT const unsigned char RollbarVersionString[];

// In this header, you should import all the public headers of your framework using statements like
// #import <Rollbar/PublicHeader.h>

// KSCrash dependencies::
#import <Rollbar/KSCrash.h>
#import <Rollbar/KSCrashInstallation.h>
#import <Rollbar/KSCrashReportFilterBasic.h>
#import <Rollbar/KSCrashReportFilterAppleFmt.h>
#import <Rollbar/KSCrashReportWriter.h>
#import <Rollbar/KSCrashReportFilter.h>
#import <Rollbar/KSCrashMonitorType.h>

// Notifier API:
//#import <Rollbar/Rollbar.h>
#import <Rollbar/RollbarFacade.h>
#import <Rollbar/RollbarNotifier.h>
#import <Rollbar/RollbarConfiguration.h>
#import <Rollbar/RollbarLog.h>
#import <Rollbar/RollbarKSCrashInstallation.h>
#import <Rollbar/RollbarKSCrashReportSink.h>
#import <Rollbar/RollbarTelemetry.h>

// DTO Abstraction:
#import <Rollbar/RollbarDTOAbstraction.h>
#import <Rollbar/JSONSupport.h>
#import <Rollbar/Persistent.h>
#import <Rollbar/DataTransferObject.h>
#import <Rollbar/DataTransferObject+CustomData.h>
#import <Rollbar/RollbarDTOAbstraction.h>

// Configuration DTOs:
#import <Rollbar/CaptureIpType.h>
#import <Rollbar/RollbarLevel.h>
#import <Rollbar/RollbarConfig.h>
#import <Rollbar/RollbarDestination.h>
#import <Rollbar/RollbarDeveloperOptions.h>
#import <Rollbar/RollbarProxy.h>
#import <Rollbar/RollbarScrubbingOptions.h>
#import <Rollbar/RollbarServer.h>
#import <Rollbar/RollbarPerson.h>
#import <Rollbar/RollbarModule.h>
#import <Rollbar/RollbarTelemetryOptions.h>
#import <Rollbar/RollbarLoggingOptions.h>

// Payload DTOs:
#import <Rollbar/RollbarPayloadDTOs.h>
#import <Rollbar/TriStateFlag.h>
#import <Rollbar/HttpMethod.h>
#import <Rollbar/RollbarSource.h>
#import <Rollbar/RollbarAppLanguage.h>
#import <Rollbar/RollbarPayload.h>
#import <Rollbar/RollbarData.h>
#import <Rollbar/RollbarBody.h>
#import <Rollbar/RollbarMessage.h>
#import <Rollbar/RollbarTrace.h>
#import <Rollbar/RollbarCallStackFrame.h>
#import <Rollbar/RollbarCallStackFrameContext.h>
#import <Rollbar/RollbarException.h>
#import <Rollbar/RollbarCrashReport.h>
#import <Rollbar/RollbarConfig.h>
#import <Rollbar/RollbarServerConfig.h>
#import <Rollbar/RollbarDestination.h>
#import <Rollbar/RollbarDeveloperOptions.h>
#import <Rollbar/RollbarProxy.h>
#import <Rollbar/RollbarScrubbingOptions.h>
#import <Rollbar/RollbarRequest.h>
#import <Rollbar/RollbarPerson.h>
#import <Rollbar/RollbarModule.h>
#import <Rollbar/RollbarTelemetryOptions.h>
#import <Rollbar/RollbarLoggingOptions.h>
#import <Rollbar/RollbarServer.h>
#import <Rollbar/RollbarClient.h>
#import <Rollbar/RollbarJavascript.h>

#import <Rollbar/RollbarTelemetryType.h>
#import <Rollbar/RollbarTelemetryBody.h>
#import <Rollbar/RollbarTelemetryLogBody.h>
#import <Rollbar/RollbarTelemetryViewBody.h>
#import <Rollbar/RollbarTelemetryErrorBody.h>
#import <Rollbar/RollbarTelemetryNavigationBody.h>
#import <Rollbar/RollbarTelemetryNetworkBody.h>
#import <Rollbar/RollbarTelemetryConnectivityBody.h>
#import <Rollbar/RollbarTelemetryManualBody.h>

#import <Rollbar/RollbarTelemetryEvent.h>



// Deploys API:
#import <Rollbar/RollbarDeploys.h>
#import <Rollbar/RollbarDeploysProtocol.h>
#import <Rollbar/RollbarDeploysManager.h>

// Deploys DTOs:
#import <Rollbar/RollbarDeploysDTOs.h>
#import <Rollbar/DeployApiCallOutcome.h>
#import <Rollbar/DeployApiCallResult.h>
#import <Rollbar/Deployment.h>
#import <Rollbar/DeploymentDetails.h>

#endif /* Rollbar_h */
