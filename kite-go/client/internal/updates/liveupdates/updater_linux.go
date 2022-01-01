// +build !standalone

package liveupdates

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kiteco/kiteco/kite-golib/kiteinstall"
)

const updateCheckInterval = 1 * time.Hour

// Listener is the linux callback for updates.
type Listener func()

func checkForUpdates(showModal bool) error {
	return nil
}

func restartAndUpdate() error {
	return nil
}

func restart() error {
	log.Println("restarting kited to apply an update")
	// kite-update is treating exit code 10 as the signal to restart
	os.Exit(10)
	return nil
}

func updateReady() bool {
	return false
}

func secondsSinceUpdateReady() int {
	return 0
}

// UpdateTarget returns the update target to use for the live updater
func UpdateTarget() (string, error) {
	return "", nil
}

func start(ctx context.Context, bundle string, f Listener, lastEvent func() time.Time) error {
	// don't start the update loop if the kite-updater service is enabled
	if enabled, _ := kiteinstall.IsSystemdTimerEnabled(); enabled {
		return nil
	}

	go func() {
		// has to be shorter than the idle timeout and update interval
		nextUpdate := time.NewTicker(5 * time.Minute)
		defer nextUpdate.Stop()

		var lastUpdateCheckTimestamp time.Time
		var updateTimestamp time.Time
		updateAvailable := false

		for {
			select {
			case <-ctx.Done():
				return

			case <-nextUpdate.C:
				// skip update check if the service is already handling it
				if serviceEnabled, _ := kiteinstall.IsSystemdTimerEnabled(); serviceEnabled {
					log.Println("skipping update check because updater service is enabled")
					continue
				}

				if updateAvailable {
					if time.Since(lastEvent()) >= idleUpdateTimeout {
						_ = restart()
					} else if int(time.Since(updateTimestamp).Seconds()) > forceUpdateThreshold {
						_ = restart()
					}
				} else if time.Since(lastUpdateCheckTimestamp) >= updateCheckInterval {
					lastUpdateCheckTimestamp = time.Now()
					if checkAndDownloadUpdate() {
						updateAvailable = true
						updateTimestamp = time.Now()
					}
				}
			}
		}
	}()

	return nil
}

// checkAndDownloadUpdate returns true if a new update has been successfully downloaded and unpacked
// it blocks until the update cmd returned,
// i.e. until either the update has been downloaded or the current status is known
func checkAndDownloadUpdate() bool {
	cmdPath, err := os.Executable()
	if err != nil {
		return false
	}

	// kite-update is installed in the same dir as kited
	updaterPath := filepath.Join(filepath.Dir(cmdPath), "kite-update")

	cmd := exec.Command(updaterPath, "self-update")
	_, err = cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Println("updater command terminated with exit code", exitErr.ExitCode())
		}
		return false
	}

	log.Println("successful fetched and unpacked update. Waiting for idle state to restart.")
	return true
}
