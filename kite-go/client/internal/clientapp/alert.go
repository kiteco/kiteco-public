package clientapp

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/client/platform/machine"
)

var (
	machineIDMessage = `Kite was unable to read its Machine ID from the registry. Please try reinstalling Kite. Kite will now exit.`
)

// Alert shows an alert UI for the given error.
func Alert(err error) {
	log.Println("alert:", err)
	switch err {
	case ErrPortInUse:
		// don't show a message
		// https://github.com/kiteco/issue-tracker/issues/197
	case machine.ErrNoMachineID:
		platform.ShowAlert(machineIDMessage)
	default:
		platform.ShowAlert(err.Error())
	}
}
