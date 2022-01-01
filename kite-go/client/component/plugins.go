package component

// PluginsManager ...
type PluginsManager interface {
	InstalledEditors() map[string]struct{}
	JetbrainsInstalledProductIDs() []string
}
