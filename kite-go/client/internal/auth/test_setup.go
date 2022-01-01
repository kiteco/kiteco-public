package auth

import (
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/visibility"
)

// SetupWithAuthDefaults configures a default auth client and permissions manager
// both and all other components passed as argument are registered in the Kited mock server as request Handlers
func SetupWithAuthDefaults(t *mockserver.TestClientServer, components ...component.Core) error {
	visibility.Clear()

	authClient := NewTestClient(300 * time.Millisecond)

	err := t.SetupWithCustomAuthClient(authClient, components...)
	if err != nil {
		return err
	}

	// make sure that all HTTP responses were closed by our components
	t.AddCleanupAction(func() {
		unclosed := authClient.getOpenConnections()
		if unclosed > 0 {
			panic(fmt.Sprintf("auth client has unclosed connections: %d", unclosed))
		}
	})

	authClient.SetTarget(t.Backend.URL)
	return nil
}
