package driver

import (
	"net/http"

	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// State contains the file driver and buffer handler for a specific buffer state
// associated with a filename and editor.
type State struct {
	Filename      string
	Editor        string
	State         string
	FileDriver    core.FileDriver
	BufferHandler http.Handler
}

// Provider is an interface for querying for driver state
type Provider interface {
	Driver(ctx kitectx.Context, filename, editor, state string) (*State, bool)
	DriverFromContent(ctx kitectx.Context, filename, editor, content string, cursor int) *State
	LatestDriver(ctx kitectx.Context, unixFilepath string) *python.UnifiedDriver
}
