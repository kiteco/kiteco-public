package updates

import (
	"github.com/kiteco/kiteco/kite-go/client/component"
)

// Manager defines the interface to access update information
type Manager interface {
	component.Core
	UpdateReady() bool
	CheckForUpdates(showModal bool)
	ReadyChan() chan struct{}
}
