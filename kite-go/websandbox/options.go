package websandbox

import (
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
)

// Options contains process-wide settings and objects
type Options struct {
	Services            *python.Services
	IDCCCompleteOptions driver.Options
	SandboxRecordMode   bool
}
