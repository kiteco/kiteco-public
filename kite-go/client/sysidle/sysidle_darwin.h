// This header file is read by the cgo tool, so must only contain C99 declarations.
// The implementations of these functions are compiled by clang, so that is where
// the interface with objective-c happens.

#include <stdbool.h>

// SystemIdle returns whether or not the system is idle
bool systemIdle();
