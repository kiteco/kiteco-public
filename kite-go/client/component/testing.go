package component

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// Implements contains boolean flags for interfaces we expect to be implemented in a component. It is used
// by TestImplements in tests to ensure we don't accidentally remove or add implementations.
type Implements struct {
	Initializer      bool
	UserAuth         bool
	Terminater       bool
	PluginEventer    bool
	ProcessedEventer bool
	NetworkEventer   bool
	KitedEventer     bool
	EventResponser   bool
	Settings         bool
	Handlers         bool
	Ticker           bool
}

// TestImplements tests whether the provided object implements all expected interfaces.
func TestImplements(t *testing.T, obj interface{}, expected Implements) {
	c, ok := obj.(Core)
	require.True(t, ok, "expected component to implement component.Core")

	_, ok = obj.(Initializer)
	require.Equalf(t, expected.Initializer, ok, "unexpected implementation mismatch for component.Initializer in %s", c.Name())

	_, ok = obj.(UserAuth)
	require.Equalf(t, expected.UserAuth, ok, "unexpected implementation mismatch for component.UserAuth in %s", c.Name())

	_, ok = obj.(Terminater)
	require.Equalf(t, expected.Terminater, ok, "unexpected implementation mismatch for component.Terminater in %s", c.Name())

	_, ok = obj.(PluginEventer)
	require.Equalf(t, expected.PluginEventer, ok, "unexpected implementation mismatch for component.PluginEventer in %s", c.Name())

	_, ok = obj.(ProcessedEventer)
	require.Equalf(t, expected.ProcessedEventer, ok, "unexpected implementation mismatch for component.ProcessedEventer in %s", c.Name())

	_, ok = obj.(NetworkEventer)
	require.Equalf(t, expected.NetworkEventer, ok, "unexpected implementation mismatch for component.NetworkEventer in %s", c.Name())

	_, ok = obj.(KitedEventer)
	require.Equalf(t, expected.KitedEventer, ok, "unexpected implementation mismatch for component.KitedEventer is %s", c.Name())

	_, ok = obj.(Settings)
	require.Equalf(t, expected.Settings, ok, "unexpected implementation mismatch for component.Settings in %s", c.Name())

	_, ok = obj.(Handlers)
	require.Equalf(t, expected.Handlers, ok, "unexpected implementation mismatch for component.Handlers in %s", c.Name())

	_, ok = obj.(Ticker)
	require.Equalf(t, expected.Ticker, ok, "unexpected implementation mismatch for component.Ticker in %s", c.Name())
}

// TestFlushComponents flushes all of the components, which implement component.TestFlusher
func TestFlushComponents(ctx context.Context, components ...Core) {
	for _, c := range components {
		if s, ok := c.(TestFlusher); ok {
			// start TestFlush in a goroutine to let all components work in parallel while sharing the same context
			s.TestFlush(ctx)
		}
	}
}
