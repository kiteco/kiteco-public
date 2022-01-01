package kiteinstall

import (
	"strings"

	"github.com/kiteco/kiteco/kite-golib/exec"
)

// IsSystemdTimerEnabled returns true if updates are automatically downloaded and applied by a system service.
// It checks if Kite's systemd user service is active.
func IsSystemdTimerEnabled() (bool, error) {
	cmd := exec.Command("systemctl", "--user", "show", "kite-updater.timer", "--property", "ActiveState")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	return isActiveOutput(outBytes), nil
}

func isActiveOutput(outBytes []byte) bool {
	// ActiveState=active indicates an active service
	return "ActiveState=active" == strings.TrimSpace(string(outBytes))
}
