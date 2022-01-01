package release

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

const enterpriseAppCastTemplateString = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0" xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle" xmlns:dc="http://purl.org/dc/elements/1.1/">
    <channel>
        <title>Kite Enterprise Client Changelog</title>
        <link>{{.AppCastURL}}</link>
        <description>Most recent changes Kite Enterprise client, with links to updates.</description>
        <language>en</language>
        <item>
            <title>Version {{.Version}}</title>
            <pubDate>{{.PubDate}}</pubDate>
            <enclosure url="{{.DownloadURL}}" sparkle:version="{{.Version}}" length="1623481" type="application/octet-stream" sparkle:dsaSignature="{{.DSASignature}}" />
            <sparkle:minimumSystemVersion>10.7</sparkle:minimumSystemVersion>
        </item>
    </channel>
</rss>
`

const (
	pollInterval = time.Minute
)

// EnterpriseServer serves the appcast and binaries for client releases
type EnterpriseServer struct {
	template *template.Template
	bucket   string

	// OS X
	osxm               sync.RWMutex
	osxDMG             []byte
	osxVersion         string
	osxUpdateSignature string
}

// NewEnterpriseServer makes a new release server to use with enterprise deployments
func NewEnterpriseServer(configBucket string) *EnterpriseServer {
	tmpl, err := template.New("appCastTemplate").Parse(enterpriseAppCastTemplateString)
	if err != nil {
		log.Fatal("failed to compile enterprise appcast template:", err)
	}

	ent := &EnterpriseServer{
		template: tmpl,
		bucket:   configBucket,
	}

	// Watch for client version changes on S3
	go ent.watchClientVersion()

	return ent
}

// SetupRoutes sets up release endpoints
func (s *EnterpriseServer) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/appcast.xml", s.handleAppcast)
	router.HandleFunc("/dls/mac/current", s.handleCurrent)
}

func (s *EnterpriseServer) handleAppcast(w http.ResponseWriter, r *http.Request) {
	s.osxm.RLock()
	defer s.osxm.RUnlock()

	type appcastData struct {
		AppCastURL   string
		DownloadURL  string
		Version      string
		DSASignature string
		PubDate      string
	}

	data := appcastData{
		AppCastURL:   "/release/appcast.xml",
		DownloadURL:  "/release/dls/mac/current",
		Version:      s.osxVersion,
		DSASignature: s.osxUpdateSignature,
		PubDate:      time.Now().Format(time.RFC1123),
	}

	w.Header().Set("Content-Type", "application/xml")
	s.template.Execute(w, data)
}

func (s *EnterpriseServer) handleCurrent(w http.ResponseWriter, r *http.Request) {
	s.osxm.RLock()
	defer s.osxm.RUnlock()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", "KiteEnterprise.dmg"))
	w.Write(s.osxDMG)
}

// --

func (s *EnterpriseServer) watchClientVersion() {
	s.pollClientVersion()

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	for range pollTicker.C {
		s.pollClientVersion()
	}
}

func (s *EnterpriseServer) pollClientVersion() {
	region := envutil.GetenvDefault("AWS_REGION", "us-west-1")
	sess := session.Must(session.NewSession())
	s3client := s3.New(sess, aws.NewConfig().WithRegion(region))

	delim := "/"
	prefix := "clients/osx/"
	listObjectsInput := &s3.ListObjectsInput{
		Bucket:    &s.bucket,
		Delimiter: &delim,
		Prefix:    &prefix,
	}

	listObjectsOutput, err := s3client.ListObjects(listObjectsInput)
	if err != nil {
		log.Println("error listing clients in s3:", err)
		return
	}

	var dirs []string
	for _, p := range listObjectsOutput.CommonPrefixes {
		dirs = append(dirs, *p.Prefix)
	}

	if len(dirs) == 0 {
		log.Printf("found no client versions in %s/%s", s.bucket, prefix)
		return
	}

	sort.Strings(dirs)

	latestVersionDir := dirs[len(dirs)-1]

	parts := strings.Split(latestVersionDir, "/")
	latestVersion := parts[2]

	// No change, continue...
	if latestVersion == s.osxVersion {
		return
	}

	log.Println("found version", latestVersion)

	dmgPath := filepath.Join(latestVersionDir, "KiteEnterprise.dmg")
	sigPath := filepath.Join(latestVersionDir, "update-signature")

	dmgBuf, err := getBytes(s3client, s.bucket, dmgPath)
	if err != nil {
		log.Println("error getting dmg:", err)
		return
	}

	sigBuf, err := getBytes(s3client, s.bucket, sigPath)
	if err != nil {
		log.Println("error getting dmg:", err)
		return
	}

	s.osxm.Lock()
	defer s.osxm.Unlock()
	s.osxDMG = dmgBuf
	s.osxVersion = latestVersion
	s.osxUpdateSignature = string(sigBuf)
}

func getBytes(s3client *s3.S3, bucket, key string) ([]byte, error) {
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	getObjectOutput, err := s3client.GetObject(getObjectInput)
	if err != nil {
		return nil, err
	}

	defer getObjectOutput.Body.Close()

	buf, err := ioutil.ReadAll(getObjectOutput.Body)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
