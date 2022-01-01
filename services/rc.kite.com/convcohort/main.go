package main

import (
	"encoding/csv"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/kiteco/kiteco/kite-go/conversion/remotecontent"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/conversion/cohort"
	"github.com/kiteco/kiteco/kite-go/conversion/country"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const (
	s3MaxmindCountryPath = "s3://kite-metrics/enrichment/maxmind/raw/country/latest/"

	locationsPath  = s3MaxmindCountryPath + "GeoLite2-Country-Locations-en/GeoLite2-Country-Locations-en.csv"
	iPv4BlocksPath = s3MaxmindCountryPath + "GeoLite2-Country-Blocks-IPv4/GeoLite2-Country-Blocks-IPv4.csv"
	iPv6BlocksPath = s3MaxmindCountryPath + "GeoLite2-Country-Blocks-IPv6/GeoLite2-Country-Blocks-IPv6.csv"
)

func main() {
	var port string
	flag.StringVar(&port, "port", ":9000", "port to listen on")
	flag.Parse()

	cioAPIKey := os.Getenv("CUSTOMER_IO_API_KEY")
	if cioAPIKey == "" {
		log.Fatal("Fatal error: CUSTOMER_IO_API_KEY environment variable required")
	}
	geolite, err := getS3GeoLite2CSVs()
	if err != nil {
		log.Fatal("Fatal: Could not load GeoLite2 CSVs", err)
	}

	cohortManager := cohort.NewManager(cioAPIKey, &http.Client{})
	countryManager, err := country.NewManager(geolite, &country.RequireEmailConfig{
		AllRequired: false,
		Countries: map[string]struct{}{
			"US": {},
			"GB": {},
		},
	})
	if err != nil {
		log.Fatal("Failed to instantiate country manager", err)
	}

	remoteContentManager := remotecontent.NewManager(remotecontent.RemoteContent{
		DashboardHeader: remotecontent.Item{
			Content: "",
			Link:    "",
		},
		DocsDashboardParagraph: remotecontent.Item{
			Content: "",
			Link:    "",
		},
	})

	r := mux.NewRouter()
	r.HandleFunc("/convcohort/.ping", handlePing)
	// cohortManager is not passed the /convcohort prefix for backward compatibility
	cohortManager.SetupRoutes(r.PathPrefix("/cohort").Subrouter())
	countryManager.SetupRoutes(r.PathPrefix("/convcohort/country").Subrouter())
	remoteContentManager.SetupRoutes(r.PathPrefix("/convcohort/remotecontent").Subrouter())

	log.Println("listening on", port)
	log.Fatalln(http.ListenAndServe(port, r))
}

func getS3GeoLite2CSVs() (country.GeoliteCSVs, error) {
	loc, err := fileutil.NewCachedReader(locationsPath)
	if err != nil {
		return country.GeoliteCSVs{}, err
	}
	ipv4, err := fileutil.NewCachedReader(iPv4BlocksPath)
	if err != nil {
		return country.GeoliteCSVs{}, err
	}
	ipv6, err := fileutil.NewCachedReader(iPv6BlocksPath)
	if err != nil {
		return country.GeoliteCSVs{}, err
	}
	return country.GeoliteCSVs{
		Locations:  *csv.NewReader(loc),
		IPv4Blocks: *csv.NewReader(ipv4),
		IPv6Blocks: *csv.NewReader(ipv6),
	}, nil
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong\n"))
}
