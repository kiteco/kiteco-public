package component

import "testing"

func Test_ImplementedInterfaces(t *testing.T) {
	TestImplements(t, &CountingComponent{}, Implements{
		Initializer:      true,
		EventResponser:   true,
		PluginEventer:    true,
		ProcessedEventer: true,
		Terminater:       true,
		UserAuth:         true,
		Settings:         true,
		Ticker:           true,
		Handlers:         true,
	})
}
