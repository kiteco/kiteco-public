// This header file is read by the cgo tool, so must only contain C99 declarations.
// The implementations of these functions are compiled by clang, so that is where
// the interface with objective-c happens.

#include <stdbool.h>

// start initializes the updater and starts listening for updates. It calls the
// relevant cgo callback when a new update is available.
void start(const char* bundlePath, char **err);

// checkForUpdates checks for available updates. If showAlert is true then it
// presents a modal to show the results, otherwise it does not present any UI.
// In either case, listeners will be alerted if there is an update available.
void checkForUpdates(bool showModal, char **err);

// updateReady returns true if an update is ready to install
bool updateReady();

// secondsSinceUpdate returns how many seconds have passed since the update was ready. If no update is ready, it returns 0.
int secondsSinceUpdateReady();

// restartAndUpdate installs the pending update
void restartAndUpdate(char **err);
