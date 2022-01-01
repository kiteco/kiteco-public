// +build windows standalone

package liveupdates

import (
	"context"
	"time"
)

// Listener is the mock callback for updates.
type Listener func()

func checkForUpdates(showModal bool) error {
	return nil
}

func restartAndUpdate() error {
	return nil
}

func restart() error {
	return nil
}

func updateReady() bool {
	return false
}

func secondsSinceUpdateReady() int {
	return 0
}

func start(ctx context.Context, bundle string, f Listener, lastEvent func() time.Time) error {
	return nil
}

// UpdateTarget returns the update target to use for the live updater
func UpdateTarget() (string, error) {
	return "", nil
}
