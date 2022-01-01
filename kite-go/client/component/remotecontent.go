package component

import "github.com/kiteco/kiteco/kite-go/conversion/remotecontent"

// RemoteContentManager ...
type RemoteContentManager interface {
	RemoteContent() remotecontent.RemoteContent
}
