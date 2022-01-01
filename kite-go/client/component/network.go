package component

import (
	"context"
)

//NetworkManager defines the functions to query whether or not network connectivity exists
type NetworkManager interface {
	Core
	// Online returns whether or not there is network connectivity
	Online() bool

	// CheckOnline checks and returns whether or not there is network connectivity
	CheckOnline(ctx context.Context) bool

	// KitedOnline checks and returns whether or not kited has been initialized to
	// an extent where it can reliably report state to requests from clients
	KitedOnline() bool

	SetOffline(bool)
}
