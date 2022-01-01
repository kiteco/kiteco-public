package capture

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	bucketName = "kite-client-capture"
	s3URL      = "https://s3.console.aws.amazon.com/s3/object"
)

var region = aws.USWest

// Server provides an endpoint for posting client runtime capture and
// uploading it to S3
type Server struct {
	auth      func(http.HandlerFunc) http.HandlerFunc
	keyPrefix string // used to separate production and dev logs
}

// NewServer returns a new server
func NewServer(auth func(http.HandlerFunc) http.HandlerFunc) *Server {
	keyPrefix := "prod"
	release := os.Getenv("RELEASE")

	// Check if release is not set or is "test-instance" (set on test-N machines)
	if release == "" || release == "test-instance" {
		keyPrefix = "dev"
	}

	return &Server{
		auth:      auth,
		keyPrefix: keyPrefix,
	}
}

// SetupRoutes prepares handlers for the capture api in the provided router.
func (s *Server) SetupRoutes(mux *mux.Router) {
	mux.Handle("/capture", s.auth(s.handlePostCapture)).Methods("POST")
}

// handlePostCapture handles uploading the provided capture files to s3
func (s *Server) handlePostCapture(w http.ResponseWriter, r *http.Request) {
	s.postCaptureImpl(w, r, true)
}

func (s *Server) postCaptureImpl(w http.ResponseWriter, r *http.Request, upload bool) {
	defer r.Body.Close()

	key, err := s.createKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if upload {
		contents, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading body of capture for %s: %v\n", key, err)
			http.Error(w, "error reading body", http.StatusInternalServerError)
			return
		}

		auth, err := aws.GetAuth("", "", "", time.Time{})
		if err != nil {
			log.Printf("error authenticating with AWS: %v\n", err)
			http.Error(w, "error authenticating with AWS", http.StatusInternalServerError)
			return
		}

		// this is transmitted as a raw string (gzipped) so that we can
		// open the resulting log files directly in the web browser for S3
		if err := s3.New(auth, region).Bucket(bucketName).PutHeader(key, contents, map[string][]string{
			"Content-Type":     []string{"text/plain"},
			"Content-Encoding": []string{"gzip"},
		}, s3.Private); err != nil {
			log.Printf("error uploading capture to s3 for %s: %v\n", key, err)
			http.Error(w, "error uploading to s3", http.StatusInternalServerError)
		}
	}

	link, err := url.Parse(s3URL)
	if err != nil {
		http.Error(w, "error parsing s3 url", http.StatusInternalServerError)
	}
	link.Path = path.Join(link.Path, bucketName, key)

	data := struct {
		URL string `json:"url"`
	}{
		URL: link.String(),
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) createKey(r *http.Request) (string, error) {
	var installid string
	user := community.GetUser(r)
	if user == nil {
		installid = r.URL.Query().Get("installid")
		if installid == "" {
			log.Printf("unable to get user or installid for request\n")
			return "", errors.Errorf("user must be authenticated or installid provided")
		}
	} else {
		installid = fmt.Sprintf("%d", user.ID)
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		log.Printf("empty file name when posting logs for user %s\n", installid)
		return "", errors.Errorf("empty file name")
	}

	machine := r.URL.Query().Get("machineid")
	if machine == "" {
		log.Printf("empty machineid when posting capture for user %s\n", installid)
		return "", errors.Errorf("empty machineid")
	}

	timestamp := r.URL.Query().Get("timestamp")
	if timestamp == "" {
		log.Printf("empty timestamp when posting capture for user %s\n", installid)
		return "", errors.Errorf("empty timestamp")
	}

	key := fmt.Sprintf("%s/%s/%s/%s/%s.gz", s.keyPrefix, installid, machine, timestamp, filename)
	return key, nil
}

// --

// SetupTestRoutes prepares test handlers for the capture api in the provided router.
func (s *Server) SetupTestRoutes(mux *mux.Router) {
	mux.Handle("/capture", s.auth(s.handlePostCaptureTest)).Methods("POST")
}

// handlePostCaptureTest returns an s3 link for the  provided capture files
func (s *Server) handlePostCaptureTest(w http.ResponseWriter, r *http.Request) {
	s.postCaptureImpl(w, r, false)
}
