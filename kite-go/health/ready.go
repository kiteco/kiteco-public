package health

import (
	"fmt"
	"net/http"
)

const (
	// ReadyEndpoint is the endpoint to use when registering ReadyHandler
	ReadyEndpoint = "/ready"
)

// ReadyHandler just responds with a string. Should be added right before
// main process loop starts in any given process.
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "let's kick the tires and light the fires, big daddy")
}
