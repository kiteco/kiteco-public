package clientlogs

import (
	"log"
	"net/http/httptest"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/community"
	_ "github.com/lib/pq"
)

func makeTestURL(base, endpoint string) *url.URL {
	baseURL, err := url.Parse(base)
	if err != nil {
		log.Fatal(err)
	}

	endpointURL, err := baseURL.Parse(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return endpointURL
}

func makeTestServer() (*httptest.Server, *Server, gorm.DB) {
	cdb := community.DB("postgres", "postgres://communityuser:kite@localhost/community_test?sslmode=disable")
	db, err := gorm.Open("postgres", cdb)
	if err != nil {
		log.Fatal(err)
	}
	users := community.NewUserManager(db, nil)
	auth := community.NewUserValidation(users)

	server := NewServer(auth.WrapNoBlock)
	mux := mux.NewRouter()
	server.SetupTestRoutes(mux)
	ts := httptest.NewServer(mux)

	return ts, server, cdb
}
