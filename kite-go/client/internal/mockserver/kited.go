package mockserver

import (
	"net/http/httptest"
	"net/url"

	"github.com/gorilla/mux"
)

// NewKitedTestServer returns a new mock instance of kited
func NewKitedTestServer() (*KitedTestServer, error) {
	router := mux.NewRouter()
	httpd := httptest.NewServer(router)

	url, err := url.Parse(httpd.URL)
	if err != nil {
		return nil, err
	}

	return &KitedTestServer{
		server: httpd,
		URL:    url,
		Router: router,
	}, nil
}

// KitedTestServer is provides a mocked
type KitedTestServer struct {
	server *httptest.Server
	URL    *url.URL
	Router *mux.Router
}

// Close releases the resources used by the kited server
func (t *KitedTestServer) Close() {
	t.server.Close()
}
