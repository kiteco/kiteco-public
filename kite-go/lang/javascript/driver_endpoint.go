package javascript

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-golib/status"
)

// DriverEndpoint handles serving the editor endpoints for a specific file.
type DriverEndpoint struct {
	driver *Driver
	router *mux.Router
}

// NewDriverEndpoint creates a new driver endpoint for the given file driver.
func NewDriverEndpoint(driver *Driver) *DriverEndpoint {
	d := DriverEndpoint{
		driver: driver,
		router: mux.NewRouter(),
	}
	r := d.router.PathPrefix("/api/buffer/{editor}/{filename}/{state}/").Subrouter()

	r.HandleFunc("/tokens", status.RecordStatusCode(handleNotImplemented, tokensStatusCode))
	r.HandleFunc("/hover", status.RecordStatusCode(handleNotImplemented, hoverStatusCode))
	r.HandleFunc("/callee", status.RecordStatusCode(handleNotImplemented, calleeStatusCode))

	return &d
}

// ServeHTTP implements http.Handler
func (d *DriverEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.router.ServeHTTP(w, r)
}

// handleNotImplemented always responds with a 501 not implemented status code
func handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
