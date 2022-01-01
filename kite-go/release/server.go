package release

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// BunnyVolumeDownloadPrefix is the BunnyCDN proxy for our S3 bucket, using the "volume" plan.
const BunnyVolumeDownloadPrefix = "https://kitedownloads.b-cdn.net"

// BunnyStandardDownloadPrefix is the BunnyCDN proxy for our S3 bucket, using the "standard" plan.
const BunnyStandardDownloadPrefix = "https://kitedownloadss.b-cdn.net"

// LocalPrefix is used when running a local server for testing purposes.
const LocalPrefix = "static"

const appCastTemplateString = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0" xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle" xmlns:dc="http://purl.org/dc/elements/1.1/">
	<channel>
		<title>Kite Client Changelog</title>
		<link>{{.AppCastURL}}</link>
		<description>Most recent changes Kite client, with links to updates.</description>
		<language>en</language>
		<item>
			<title>Version {{.Version}}</title>
			<pubDate>{{.PubDate}}</pubDate>
			<enclosure url="{{.DownloadURL}}" sparkle:version="{{.Version}}" length="1623481" type="application/octet-stream" sparkle:dsaSignature="{{.DSASignature}}" />
			<sparkle:minimumSystemVersion>10.7</sparkle:minimumSystemVersion>
			<sparkle:deltas>
			{{range $delta := .Deltas}}
				<enclosure url="{{$delta.DownloadURL}}"
					sparkle:version="{{$delta.ToVersion}}"
					sparkle:deltaFrom="{{$delta.FromVersion}}"
					length="0"
					type="application/octet-stream"
					sparkle:dsaSignature="{{$delta.DSASignature}}" />
			{{end}}
			</sparkle:deltas>
		</item>
	</channel>
</rss>
`

// DefaultPlatforms specifies all the platforms handled by the release server
var DefaultPlatforms = []Platform{Mac, Windows, Linux}

type metadatas struct {
	canary    *Metadata
	nonCanary *Metadata

	deltasToCanary    []*Delta
	deltasToNonCanary []*Delta
}

// Server serves the appcast and binaries for client releases
type Server struct {
	getsCanary func(sum64 uint64, releasePercentage uint8) bool

	template        *template.Template
	metadataManager MetadataManager
	router          *mux.Router
	releaseRoot     string

	rw        sync.RWMutex
	platforms []Platform
	metadatas map[Platform]*metadatas
}

// NewServer makes a new release server with the provided
// MetadataManager, and adds the relevant routes.
func NewServer(router *mux.Router, manager MetadataManager, platforms []Platform) *Server {
	tmpl, err := template.New("appCastTemplate").Parse(appCastTemplateString)
	if err != nil {
		log.Fatal("Failed to compile appcast template")
	}

	server := &Server{
		getsCanary:      getsCanaryFromHash,
		template:        tmpl,
		metadataManager: manager,
		router:          router,
		platforms:       platforms,
		metadatas:       make(map[Platform]*metadatas),
	}

	// Kite first download URL
	router.HandleFunc("/dls/{platform}/current", server.serveLatest)
	// /dls/linux/current downloads a shell script,
	// which in turn downloads the full binary from /dls/current/kite-installer
	router.HandleFunc("/linux/{version}/kite-installer", server.serveLinuxInstaller)

	// macOS/Sparkle updater hits /appcast.xml, which points at CDN download URL
	router.HandleFunc("/appcast.xml", server.serveAppCast).Name("appcast")
	// macOS/Sparkle fallback updater hits /fallback/{version}/appcast.xml from KiteHelper
	router.HandleFunc("/fallback/{version}/appcast.xml", server.serveFallbackAppCast).Name("fallback")

	// Windows updater hits /windows/kite-app/update-check, which redirects to XML update manifest (on CDN)
	router.HandleFunc("/windows/kite-app/update-check", server.serveWinUpdateCheck)
	// XML update manifest points at /windows/{version}/KiteUpdater.exe
	router.HandleFunc("/windows/{version}/KiteUpdater.exe", server.serveWinUpdate)
	// or at /windows/{toVersion}/KitePatchUpdater{fromVersion}.exe
	router.HandleFunc("/windows/{toVersion}/KitePatchUpdater{fromVersion}-{toVersionAgain}.exe", server.serveWinDeltaUpdate)

	// Linux updater hits /linux/kite-app/update-check, which redirects to JSON update manifest (on CDN)
	router.HandleFunc("/linux/kite-app/update-check", server.serveLinuxUpdateCheck)
	// JSON update manifest points at /linux/{version}/kite-updater.sh or
	// /linux/{version}/{fromVersion}/kite-patch-updater.sh
	// which redirect to CDN download URL for the updater or patch updater package.
	router.HandleFunc("/linux/{version}/kite-updater.sh", server.serveLinuxUpdate)
	router.HandleFunc("/linux/{version}/{fromVersion}/kite-patch-updater.sh", server.serveLinuxPatchUpdate)

	if err := server.updateCache(); err != nil {
		log.Fatalln("failed to query release DB:\n", err)
	}
	go func() {
		for {
			time.Sleep(10 * time.Second)
			err := server.updateCache()
			if err != nil {
				rollbar.Error(errors.New("failed to update cache from DB"), err)
			}
		}
	}()

	return server
}

func (s *Server) updateCache() error {
	s.rw.Lock()
	defer s.rw.Unlock()

	var errs errors.Errors
	for _, p := range s.platforms {
		mds := s.metadatas[p]
		if mds == nil {
			mds = &metadatas{}
		}

		canary, err := s.metadataManager.LatestCanary(p)
		errs = errors.Append(errs, err)
		if err == nil && canary != nil {
			mds.canary = canary
			// set this nil so that the state remains consistent if the next query fails
			mds.deltasToCanary = nil
			if deltasToCanary, err := s.metadataManager.DeltasToVersion(p, canary.Version); err == nil {
				mds.deltasToCanary = deltasToCanary
			} else {
				errs = errors.Append(errs, err)
			}
		}
		nonCanary, err := s.metadataManager.LatestNonCanary(p)
		if err == nil && nonCanary != nil {
			mds.nonCanary = nonCanary
			// set this nil so that the state remains consistent if the next query fails
			mds.deltasToNonCanary = nil
			if deltasToNonCanary, err := s.metadataManager.DeltasToVersion(p, nonCanary.Version); err == nil {
				mds.deltasToNonCanary = deltasToNonCanary
			} else {
				errs = errors.Append(errs, err)
			}
		}

		s.metadatas[p] = mds
	}
	return errs
}

func (s *Server) latestCanary(p Platform) *Metadata {
	s.rw.RLock()
	defer s.rw.RUnlock()
	if _, ok := s.metadatas[p]; !ok {
		return nil
	}
	return s.metadatas[p].canary
}

func (s *Server) latestNonCanary(p Platform) *Metadata {
	s.rw.RLock()
	defer s.rw.RUnlock()
	if _, ok := s.metadatas[p]; !ok {
		return nil
	}
	return s.metadatas[p].nonCanary
}

func (s *Server) deltasToVersion(p Platform, version string) []*Delta {
	deltas, found := func() ([]*Delta, bool) {
		s.rw.RLock()
		defer s.rw.RUnlock()

		mds := s.metadatas[p]
		if mds.canary != nil && version == mds.canary.Version {
			return mds.deltasToCanary, true
		}
		if mds.nonCanary != nil && version == mds.nonCanary.Version {
			return mds.deltasToNonCanary, true
		}
		return nil, false
	}()
	if found {
		return deltas
	}
	deltas, err := s.metadataManager.DeltasToVersion(p, version)
	if err != nil {
		rollbar.Error(errors.New("failed to query deltas"), err)
		return nil
	}
	return deltas
}

func (s *Server) deltaForVersions(p Platform, from, to string) *Delta {
	deltas := s.deltasToVersion(p, to)
	for _, delta := range deltas {
		if delta.FromVersion == from {
			return delta
		}
	}
	return nil
}

func (s *Server) latestForClient(p Platform, clientID string) *Metadata {
	canary := s.latestCanary(p)
	if canary == nil {
		return s.latestNonCanary(p)
	}
	if canary.ReleasePercentage >= 100 {
		return canary
	}
	if clientID == "" {
		// only ship non-canary releases to clients with empty client IDs
		return s.latestNonCanary(p)
	}

	hash := fnv.New64()
	hash.Write([]byte(clientID))
	hash.Write([]byte(canary.Version))
	if s.getsCanary(hash.Sum64(), canary.ReleasePercentage) {
		return canary
	}

	return s.latestNonCanary(p)
}

func getsCanaryFromHash(sum64 uint64, releasePercentage uint8) bool {
	if uint8(sum64%100) < releasePercentage {
		return true
	}
	return false
}

// SetReleaseRoot sets the directory from which to server releases. This can be useful for
// development and testing.
func (s *Server) SetReleaseRoot(root string) {
	s.releaseRoot = root
	log.Printf("Serving downloads from: %s", s.releaseRoot)
}

func (s *Server) serveAppCast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	current := s.latestForClient(Mac, r.URL.Query().Get("machine-id"))
	if current == nil {
		http.Error(w, "no release found", 404)
		return
	}
	appCastURL, err := s.router.Get("appcast").URL()
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}

	s.serveAppCastHelper(w, r, current, appCastURL)
}

func (s *Server) serveFallbackAppCast(w http.ResponseWriter, r *http.Request) {
	// Prior behavior was to only serve a fallback release if the latest was Bad.
	// Since we are favoring re-releasing over marking as Bad, this endpoint is
	// deprecated and only 404s.
	http.Error(w, "no release found", 404)
}

func (s *Server) serveAppCastHelper(w http.ResponseWriter, r *http.Request, release *Metadata, appCastURL *url.URL) {
	prefix := s.getAssetPrefix(BunnyVolumeDownloadPrefix, r.Host)
	u, err := url.Parse(fmt.Sprintf("%s/mac/%s/Kite.dmg", prefix, release.Version))
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}
	downloadURL := u.String()

	deltas := s.deltasToVersion(Mac, release.Version)
	type DeltaUpdate struct {
		DownloadURL  string
		FromVersion  string
		ToVersion    string
		DSASignature string
	}
	var deltaUpdates []DeltaUpdate
	for _, d := range deltas {
		u, err := url.Parse(fmt.Sprintf("%s/mac/%s/deltaFrom/%s/Kite.delta", prefix, d.ToVersion, d.FromVersion))
		if err != nil {
			webutils.ErrorResponse(w, r, err, nil)
			return
		}
		dURL := u.String()
		deltaUpdates = append(deltaUpdates, DeltaUpdate{
			DownloadURL:  dURL,
			FromVersion:  d.FromVersion,
			ToVersion:    d.ToVersion,
			DSASignature: d.DSASignature,
		})
	}

	s.template.Execute(w, struct {
		AppCastURL   string
		DownloadURL  string
		Version      string
		DSASignature string
		PubDate      string
		Deltas       []DeltaUpdate
	}{
		AppCastURL:   appCastURL.String(),
		DownloadURL:  downloadURL,
		Version:      release.Version,
		DSASignature: release.DSASignature,
		PubDate:      release.CreatedAt.Format(time.RFC1123),
		Deltas:       deltaUpdates,
	})
}

func (s *Server) serveLatest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var platform Platform
	requestedPlatform := vars["platform"]
	switch requestedPlatform {
	case "mac":
		platform = Mac
	case "windows":
		platform = Windows
	case "linux":
		platform = Linux
	}

	latest := s.latestForClient(platform, "")
	if latest == nil {
		http.Error(w, "no release found", http.StatusNotFound)
		return
	}

	// download URL
	var dlURL string
	prefix := s.getAssetPrefix(BunnyStandardDownloadPrefix, r.Host)
	switch platform {
	case Mac:
		dlURL = fmt.Sprintf("%s/mac/%s/Kite.dmg", prefix, latest.Version)
	case Windows:
		dlURL = fmt.Sprintf("%s/windows/%s/KiteSetup.exe", prefix, latest.Version)
	case Linux:
		dlURL = fmt.Sprintf("%s/linux/%s/kite-installer.sh", prefix, latest.Version)
	}

	// validate URL
	u, err := url.Parse(dlURL)
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// serveWinUpdateCheck handles windows client update checks and serves the update info file.
func (s *Server) serveWinUpdateCheck(w http.ResponseWriter, r *http.Request) {
	// extract query params
	qvals := r.URL.Query()
	vals := make(map[string]string)
	for k, v := range qvals {
		if len(v) == 1 {
			vals[k] = v[0]
		}
	}
	// check that all query params are present
	if len(qvals) != len(vals) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// check if non-update
	if strings.ToLower(vals["will-attempt-update-if-provided"]) != "true" {
		w.Write([]byte("no update, requested no update"))
		return
	}

	// allow canary releases on windows, check if version is older
	latestVersion := s.latestForClient(Windows, vals["machine-id"])
	if latestVersion == nil {
		http.Error(w, "no release found", http.StatusNotFound)
		return
	}
	isOlder, err := olderVersion(vals["version"], latestVersion.Version)
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}

	// if older, return update info, else no update
	if isOlder {
		prefix := s.getAssetPrefix(BunnyVolumeDownloadPrefix, r.Host)
		// server update info for delta updater, if available
		if deltaUpdate := s.deltaForVersions(Windows, vals["version"], latestVersion.Version); deltaUpdate != nil {
			u, err := url.Parse(fmt.Sprintf("%s/windows/%s/deltaFrom/%s/KiteDeltaUpdateInfo.xml",
				prefix, deltaUpdate.ToVersion, deltaUpdate.FromVersion))
			if err != nil {
				webutils.ErrorResponse(w, r, err, nil)
				return
			}
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
			return
		}

		// get update info file
		u, err := url.Parse(fmt.Sprintf("%s/windows/%s/KiteUpdateInfo.xml", prefix, latestVersion.Version))
		if err != nil {
			webutils.ErrorResponse(w, r, err, nil)
			return
		}
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
		return
	}
	return
}

func (s *Server) serveWinUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version := vars["version"]
	u, err := url.Parse(fmt.Sprintf("%s/windows/%s/KiteUpdater.exe", s.getAssetPrefix(BunnyVolumeDownloadPrefix, r.Host), version))
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
	return
}

func (s *Server) serveWinDeltaUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	toVersion := vars["toVersion"]
	fromVersion := vars["fromVersion"]
	u, err := url.Parse(fmt.Sprintf("%s/windows/%s/deltaFrom/%s/KiteDeltaUpdater.exe", s.getAssetPrefix(BunnyVolumeDownloadPrefix, r.Host), toVersion, fromVersion))
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
	return
}

// serveLinuxUpdateCheck handles linux client update checks and serves the update info file.
func (s *Server) serveLinuxUpdateCheck(w http.ResponseWriter, r *http.Request) {
	// extract query params
	qvals := r.URL.Query()
	vals := make(map[string]string)
	for k, v := range qvals {
		if len(v) == 1 {
			vals[k] = v[0]
		}
	}

	// check that all query params are present
	if len(qvals) != len(vals) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	if vals["cpu_features_no_avx"] == "true" {
		// don't update users without AVX support
		return
	}

	// fixme is this needed for Linux?
	// check if non-update
	if strings.ToLower(vals["will-attempt-update-if-provided"]) != "true" {
		w.WriteHeader(http.StatusBadRequest) // fixme this is different to the windows response handler above
		w.Write([]byte("no update, requested no update"))
		return
	}

	// check if version is older
	latestVersion := s.latestForClient(Linux, vals["install-id"])
	if latestVersion == nil {
		http.Error(w, "no release found", http.StatusNotFound)
		return
	}
	isOlder, err := olderVersion(vals["version"], latestVersion.Version)
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}

	// if older, return update info, else no update
	if isOlder {
		prefix := s.getAssetPrefix(BunnyVolumeDownloadPrefix, r.Host)
		// server update info for delta updater, if available
		if deltaUpdate := s.deltaForVersions(Linux, vals["version"], latestVersion.Version); deltaUpdate != nil {
			u, err := url.Parse(fmt.Sprintf("%s/linux/%s/deltaFrom/%s/version.json",
				prefix, deltaUpdate.ToVersion, deltaUpdate.FromVersion))
			if err != nil {
				webutils.ErrorResponse(w, r, err, nil)
				return
			}
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
			return
		}
		// get update info file
		u, err := url.Parse(fmt.Sprintf("%s/linux/%s/version.json", prefix, latestVersion.Version))
		if err != nil {
			webutils.ErrorResponse(w, r, err, nil)
			return
		}
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
		return
	}
	return
}

func (s *Server) serveLinuxUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version := vars["version"]
	u, err := url.Parse(fmt.Sprintf("%s/linux/%s/%s", s.getAssetPrefix(BunnyVolumeDownloadPrefix, r.Host), version, "kite-updater.sh"))
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
	return
}

func (s *Server) serveLinuxPatchUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version := vars["version"]
	fromVersion := vars["fromVersion"]
	u, err := url.Parse(fmt.Sprintf("%s/linux/%s/deltaFrom/%s/%s", s.getAssetPrefix(BunnyVolumeDownloadPrefix, r.Host), version, fromVersion, "kite-updater.sh"))
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
	return
}

func (s *Server) serveLinuxInstaller(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	version := vars["version"]
	// "current" is an alias to retrieve the latest available version of the kite-installer binary
	if version == "current" {
		latest := s.latestForClient(Linux, "")
		if latest == nil {
			http.Error(w, "no release found", http.StatusNotFound)
			return
		}
		version = latest.Version
	}

	u, err := url.Parse(fmt.Sprintf("%s/linux/%s/%s", s.getAssetPrefix(BunnyStandardDownloadPrefix, r.Host), version, "kite-installer"))
	if err != nil {
		webutils.ErrorResponse(w, r, err, nil)
		return
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// --

// getAssetPrefix returns the input globalPrefix if not running a local server,
// otherwise it returns the local server prefix.
func (s *Server) getAssetPrefix(globalPrefix, host string) string {
	if s.releaseRoot == "" {
		return globalPrefix
	}

	// files are being served from the fs for testing
	h, err := url.Parse("//" + host)
	if err != nil {
		log.Println(err)
		return ""
	}
	prefix, err := url.Parse("http:" + h.String() + "/" + LocalPrefix)
	if err != nil {
		log.Println(err)
		return ""
	}
	return prefix.String()
}

// olderVersion is a helper for comparing windows and linux version strings.
// Returns True if left is greater version than right.
func olderVersion(v1 string, v2 string) (bool, error) {
	// format is 1.[year].[month+date].[minor], so just split by period and compare each
	v1Split := strings.Split(v1, ".")
	v2Split := strings.Split(v2, ".")

	// convert to int, return immediately if greater for each segment
	for i, v := range v1Split {
		v1i, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("invalid version string %s", v1)
			return false, err
		}
		v2i, err := strconv.Atoi(v2Split[i])
		if err != nil {
			log.Printf("invalid version string %s", v2)
			return false, err
		}
		if v1i < v2i {
			return true, nil
		}
	}
	// if none were greater, return false
	return false, nil
}
