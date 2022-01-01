package internal

import "context"

// AdditionalIPluginDs is for Plugin managers, which support additional ids
type AdditionalIPluginDs interface {
	AdditionalIDs() []string
}

// InstalledProductIDs ...
type InstalledProductIDs interface {
	InstalledProductIDs(context.Context) []string
}
