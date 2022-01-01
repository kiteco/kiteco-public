// +build !windows,!standalone

package throttle

import (
	"github.com/kiteco/kiteco/kite-golib/errors"
	"golang.org/x/sys/unix"
)

// SetLowPriority lowers the calling process (including all threads) priority.
func SetLowPriority() error {
	if err := unix.Setpgid(0, 0); err != nil {
		return errors.Wrapf(err, "Failed to move process to own process group. Not touching process group priority.")
	}
	err := unix.Setpriority(unix.PRIO_PGRP, 0, 9)
	return errors.WrapfOrNil(err, "Failed to touch process group priority.")
	// We may want to also call ioprio_set,
	// but unclear if that's a good idea yet.
}
