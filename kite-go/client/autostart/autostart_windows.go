// +build !standalone

package autostart

import (
	"github.com/kiteco/kiteco/kite-go/client/internal/reg"
)

func setEnabled(enabled bool) error {
	if enabled {
		return reg.UpdateHKCURun()
	}
	return reg.RemoveHKCURun()
}
