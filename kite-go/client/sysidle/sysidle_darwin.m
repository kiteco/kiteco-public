#include <ApplicationServices/ApplicationServices.h>

const int IDLE_THRESHOLD = 10 * 60; // 10 minutes in seconds

bool SystemIdleImpl() {
    CFTimeInterval timeSinceLastEvent = CGEventSourceSecondsSinceLastEventType(kCGEventSourceStateHIDSystemState, kCGAnyInputEventType);
    return timeSinceLastEvent >= IDLE_THRESHOLD;
}

bool systemIdle() {
	// make sure autoreleased objects are released before this function returns.
	@autoreleasepool {
		return SystemIdleImpl();
	}
}
