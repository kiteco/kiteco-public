//
//  RollbarData.h
//  Rollbar
//
//  Created by Andrey Kornich on 2019-10-10.
//  Copyright © 2019 Rollbar. All rights reserved.
//

#import "DataTransferObject.h"
#import "RollbarLevel.h"
#import "RollbarAppLanguage.h"

@class RollbarBody;
@class RollbarPerson;
@class RollbarRequest;
@class RollbarServer;
@class RollbarClient;
@class RollbarModule;

NS_ASSUME_NONNULL_BEGIN

@interface RollbarData : DataTransferObject

// Required: environment
// The name of the environment in which this occurrence was seen.
// A string up to 255 characters. For best results, use "production" or "prod" for your
// production environment.
// You don't need to configure anything in the Rollbar UI for new environment names;
// we'll detect them automatically.
@property (nonatomic, copy) NSString *environment;

// Required: body
// The main data being sent. It can either be a message, an exception, or a crash report.
@property (nonatomic, nonnull) RollbarBody *body;

// Optional: level
// The severity level. One of: "critical", "error", "warning", "info", "debug"
// Defaults to "error" for exceptions and "info" for messages.
// The level of the *first* occurrence of an item is used as the item's level.
@property (nonatomic) RollbarLevel level; //optional. default: error for exceptions and info for messages.

// Optional: timestamp
// When this occurred, as a unix timestamp.
@property (nonatomic) NSTimeInterval timestamp; //stored in JSON as long

// Optional: code_version
// A string, up to 40 characters, describing the version of the application code
// Rollbar understands these formats:
// - semantic version (i.e. "2.1.12")
// - integer (i.e. "45")
// - git SHA (i.e. "3da541559918a808c2402bba5012f6c60b27661c")
// If you have multiple code versions that are relevant, those can be sent inside "client" and "server"
// (see those sections below)
// For most cases, just send it here.
@property (nonatomic, copy) NSString *codeVersion;

// Optional: platform
// The platform on which this occurred. Meaningful platform names:
// "browser", "android", "ios", "flash", "client", "heroku", "google-app-engine"
// If this is a client-side event, be sure to specify the platform and use a post_client_item access token.
@property (nonatomic, copy, nullable) NSString *platform;

// Optional: language
// The name of the language your code is written in.
// This can affect the order of the frames in the stack trace. The following languages set the most
// recent call first - 'ruby', 'javascript', 'php', 'java', 'objective-c', 'lua'
// It will also change the way the individual frames are displayed, with what is most consistent with
// users of the language.
@property (nonatomic) RollbarAppLanguage language;

// Optional: framework
// The name of the framework your code uses
@property (nonatomic, copy, nullable) NSString *framework;

// Optional: context
// An identifier for which part of your application this event came from.
// Items can be searched by context (prefix search)
// For example, in a Rails app, this could be `controller#action`.
// In a single-page javascript app, it could be the name of the current screen or route.
@property (nonatomic, copy, nullable) NSString *context;

// Optional: request
// Data about the request this event occurred in.
@property (nonatomic, nullable) RollbarRequest *request;

// Optional: person
// The user affected by this event. Will be indexed by ID, username, and email.
// People are stored in Rollbar keyed by ID. If you send a multiple different usernames/emails for the
// same ID, the last received values will overwrite earlier ones.
@property (nonatomic, nullable) RollbarPerson *person;

// Optional: server
// Data about the server related to this event.
@property (nonatomic, nullable) RollbarServer *server;

// Optional: client
// Data about the client device this event occurred on.
// As there can be multiple client environments for a given event (i.e. Flash running inside
// an HTML page), data should be namespaced by platform.
@property (nonatomic, nullable) RollbarClient *client;

// Optional: custom
// Any arbitrary metadata you want to send. "custom" itself should be an object.
@property (nonatomic, nullable) NSObject<JSONSupport> *custom;

// Optional: fingerprint
// A string controlling how this occurrence should be grouped. Occurrences with the same
// fingerprint are grouped together. See the "Grouping" guide for more information.
// Should be a string up to 40 characters long; if longer than 40 characters, we'll use its SHA1 hash.
// If omitted, we'll determine this on the backend.
@property (nonatomic, copy, nullable) NSString *fingerprint;

// Optional: title
// A string that will be used as the title of the Item occurrences will be grouped into.
// Max length 255 characters.
// If omitted, we'll determine this on the backend.
@property (nonatomic, copy, nullable) NSString *title;

// Optional: uuid
// A string, up to 36 characters, that uniquely identifies this occurrence.
// While it can now be any latin1 string, this may change to be a 16 byte field in the future.
// We recommend using a UUID4 (16 random bytes).
// The UUID space is unique to each project, and can be used to look up an occurrence later.
// It is also used to detect duplicate requests. If you send the same UUID in two payloads, the second
// one will be discarded.
// While optional, it is recommended that all clients generate and provide this field
@property (nonatomic, nullable) NSUUID *uuid;

// Optional: notifier
// Describes the library used to send this event.
@property (nonatomic, nullable) RollbarModule *notifier;

#pragma mark - initialization

-(instancetype)initWithEnvironment:(nonnull NSString *)environment
                              body:(nonnull RollbarBody *)body;

@end

NS_ASSUME_NONNULL_END
