package user

import (
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/search"
)

// Options contains process-wide settings and objects
// that will be shared by all user.Context objects within
// A process. API's for these objects should be goroutine safe!
type Options struct {
	Search            *search.Services // Search services for active and passive search
	Community         *community.App   // Pointer to community App
	PrintStack        bool             // Flag to enable printing stack traces on panics
	StackAll          bool             // Flag to enable including stack routines of all other goroutines
	StackSize         uint             // Sets stack size in bytes
	SegmentWriteToken string           // API key to send events to Segment
}
