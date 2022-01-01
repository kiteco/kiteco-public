package localfiles

import (
	"log"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/community"
	_ "github.com/lib/pq"
)

func makeInMemoryTestStore() *ContentStore {
	fdb := FileDB("postgres", "postgres://localfilesuser:kite@localhost/localfiles_test")
	_, err := fdb.Exec("DROP TABLE IF EXISTS file")
	if err != nil {
		log.Fatalf("error dropping file table: %s", err)
	}

	opts := ContentStoreOptions{
		Type:            InMemoryContentStore,
		MaxWriteRetries: 3,
	}
	store, err := NewContentStore(opts, fdb)
	if err != nil {
		log.Fatalf("error creating content store: %s", err.Error())
	}

	err = store.Migrate()
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Fatalf("error creating content store: %s", err.Error())
		}
	}

	return store
}

func makeTestURL(base, endpoint string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		log.Fatal(err)
	}

	endpointURL, err := baseURL.Parse(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return endpointURL.String()
}

func makeTestServer() (*httptest.Server, *ContentStore, *Server) {
	store := makeInMemoryTestStore()

	db, err := gorm.Open("postgres", store.FileDB)
	if err != nil {
		log.Fatal(err)
	}
	users := community.NewUserManager(db, nil)
	auth := community.NewUserValidation(users)
	server := NewServer(store, auth)

	mux := mux.NewRouter()
	server.SetupRoutes(mux)
	ts := httptest.NewServer(mux)

	return ts, store, server
}
