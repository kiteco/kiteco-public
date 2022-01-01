// +build !standalone

package throttle

import (
	"github.com/kiteco/kiteco/kite-golib/errors"
	"golang.org/x/sys/windows"
)

// https://docs.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-setpriorityclass
const belowNormalPriorityClass = 0x00004000

// SetLowPriority lowers the calling process (including all threads) priority.
func SetLowPriority() error {
	handle, err := windows.GetCurrentProcess()
	if err != nil {
		return errors.Wrapf(err, "Failed to get current process handle. Not touching priority.")
	}
	err = windows.SetPriorityClass(handle, belowNormalPriorityClass)
	return errors.Wrapf(err, "Failed to get current process handle. Not touching priority.")
}
