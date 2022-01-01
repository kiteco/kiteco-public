package main

import (
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/curation/titleparser"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func makeTestApp() (*App, gorm.DB) {
	codeExampleDB := curation.GormDB("sqlite3", ":memory:")
	authDB := curation.GormDB("postgres", "postgres://communityuser:kite@localhost/community_test?sslmode=disable")

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(staticfs, "templates", nil)

	titleValidator := &titleparser.TitleValidator{}
	// Note we pass in an empty string for the reference server -- that's used
	// only by client-side javascript at the moment, so it shouldn't matter
	// for tests.
	app := NewApp(AppOptions{
		CodeExampleDB:  codeExampleDB,
		AuthDB:         authDB,
		Templates:      templates,
		TitleValidator: titleValidator,
		DefaultUnified: true,
	})
	if err := app.Migrate(); err != nil {
		log.Fatalln(err)
	}
	return app, authDB
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

func makeTestServer() (*httptest.Server, *App, gorm.DB) {
	app, authDB := makeTestApp()
	mux := mux.NewRouter()
	app.SetupRoutes(mux)
	ts := httptest.NewServer(mux)
	return ts, app, authDB
}

func makeTestClient() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	return &http.Client{Jar: jar}
}
