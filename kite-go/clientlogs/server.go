package clientlogs

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/rollbar/rollbar-go"
)

const defaultMaxTracebackSize = 100 << 10 // 100KB, note: maximum rollbar message size is 128KB

var (
	maxLogRead       = 100 << 10 // 100KB
	maxTracebackSize = defaultMaxTracebackSize
)

const (
	clientLogBucket = "XXXXXXX"
	crashLogBucket  = "XXXXXXX"
	rollbarToken    = "XXXXXXX"
	s3URL           = "https://s3.console.aws.amazon.com/s3/object"
)

var region = aws.USWest

// Server provides an endpoint for posting client logs and
// uploading them to S3
type Server struct {
	auth         func(http.HandlerFunc) http.HandlerFunc
	keyPrefix    string // used to separate production and dev logs
	crashRollbar *rollbar.Client
	m            sync.Mutex
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
		auth:         auth,
		keyPrefix:    keyPrefix,
		crashRollbar: newRollbarClient(),
	}
}

// SetupRoutes prepares handlers for the client logs api in the provided router.
func (s *Server) SetupRoutes(mux *mux.Router) {
	mux.HandleFunc("/clientlogs", s.auth(s.handleProcessLogs)).Methods("POST")
	mux.HandleFunc("/windowscrash", s.auth(s.handleWindowsCrash)).Methods("POST")
	mux.HandleFunc("/servicestatus", s.auth(s.handleServiceStatus)).Methods("POST")
	mux.HandleFunc("/logupload", s.auth(s.handleLogUpload)).Methods("POST")
}

// Close closes the windows rollbar client, which waits for unsent messages to be sent.
func (s *Server) Close() {
	if err := s.crashRollbar.Close(); err != nil {
		log.Println(err)
	}
}

// handleProcessLogs handles processing and uploading the provided log files to s3
func (s *Server) handleProcessLogs(w http.ResponseWriter, r *http.Request) {
	key, err := s.createKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// hold on to compressed contents
	defer r.Body.Close()
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)

	// check if the log contains a crash
	_, statusCode := s.findAndReportCrash(tee, &buf, r)
	if statusCode != http.StatusOK {
		_, err := ioutil.ReadAll(tee)
		if err != nil {
			log.Printf("error reading body of logs for %s: %v\n", key, err)
			http.Error(w, "error reading body", http.StatusInternalServerError)
			return
		}
	}

	// store client log in s3
	errStr, statusCode := s.storeLogs(clientLogBucket, key, buf.Bytes())
	if statusCode != http.StatusOK {
		http.Error(w, errStr, statusCode)
	}

	w.WriteHeader(http.StatusOK)
}

// handleWindowsCrash handles uploading the provided log files to Windows crash bucket on s3
func (s *Server) handleWindowsCrash(w http.ResponseWriter, r *http.Request) {
	// hold on to compressed contents
	defer r.Body.Close()
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)

	// check if the log contains a crash
	errStr, statusCode := s.findAndReportCrash(tee, &buf, r)
	if statusCode != http.StatusOK {
		http.Error(w, errStr, statusCode)
	}

	w.WriteHeader(http.StatusOK)
}

// handleServiceStatus handles status pings from the Kite service, forwarding them to Segment
func (s *Server) handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("error decoding request: %v", err)
		http.Error(w, "error decoding request", http.StatusBadRequest)
		return
	}

	var installID string
	var ok bool
	if installID, ok = req["install_id"].(string); !ok {
		log.Printf("invalid install id type: %T", req["install_id"])
		http.Error(w, "bad install ID", http.StatusBadRequest)
		return
	}

	if installID == "" {
		log.Printf("empty install ID")
		http.Error(w, "empty install ID", http.StatusBadRequest)
		return
	}

	//err := s.segment.Enqueue(analytics.Track{
	//	UserId: installID,
	//	Event:  "service_status",
	//	Properties: map[string]interface{}{
	//		"props":      req,
	//		"install_id": installID,
	//		"sent_at":    time.Now().Unix(),
	//		"os":         "windows",
	//	}})
	//if err != nil {
	//	log.Printf("error enqueuing service status message: %v", err)
	//}

	w.WriteHeader(http.StatusOK)
}

// handleLogUpload uploads the provided log file to s3 and returns a link
func (s *Server) handleLogUpload(w http.ResponseWriter, r *http.Request) {
	s.logUploadImpl(w, r, true)
}

func (s *Server) logUploadImpl(w http.ResponseWriter, r *http.Request, upload bool) {
	defer r.Body.Close()

	key, err := s.createKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if upload {
		contents, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading body of logs for %s: %v\n", key, err)
			http.Error(w, "error reading body", http.StatusInternalServerError)
			return
		}

		// store client log in s3
		errStr, statusCode := s.storeLogs(clientLogBucket, key, contents)
		if statusCode != http.StatusOK {
			http.Error(w, errStr, statusCode)
		}
	}

	link, err := url.Parse(s3URL)
	if err != nil {
		http.Error(w, "error parsing s3 url", http.StatusInternalServerError)
	}
	link.Path = path.Join(link.Path, clientLogBucket, key)

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

// --

// SetupTestRoutes prepares test handlers for the capture api in the provided router.
func (s *Server) SetupTestRoutes(mux *mux.Router) {
	mux.Handle("/logupload", s.auth(s.handleLogUploadTest)).Methods("POST")
}

// handleLogUploadTest returns an s3 link for the provided log file
func (s *Server) handleLogUploadTest(w http.ResponseWriter, r *http.Request) {
	s.logUploadImpl(w, r, false)
}

var readerPool = &sync.Pool{
	New: func() interface{} { return new(gzip.Reader) },
}

func (s *Server) findAndReportCrash(tee io.Reader, buf *bytes.Buffer, r *http.Request) (string, int) {
	key, platform, err := s.createCrashKey(r)
	if err != nil {
		return err.Error(), http.StatusBadRequest
	}

	cur, version, installID, err := s.readLogTail(tee, buf)
	if err != nil {
		log.Printf("error reading logs for %s: %v\n", key, err)
		return err.Error(), http.StatusInternalServerError
	}

	// find crash
	errStr, traceback := findCrash(cur)
	if traceback == nil {
		// do not report/store if no crash is found
		return "", http.StatusOK
	}

	if err := s.reportCrash(key, platform, version, installID, errStr, traceback); err != nil {
		log.Printf("error reporting crash for %s: %v\n", key, err)
		return "error reporting crash", http.StatusInternalServerError
	}

	// send compressed contents to s3
	errStr, statusCode := s.storeLogs(crashLogBucket, key, buf.Bytes())
	if statusCode != http.StatusOK {
		log.Printf("error storing logs for %s: %s\n", key, errStr)
		return errStr, statusCode
	}

	return "", http.StatusOK
}

func (s *Server) readLogTail(tee io.Reader, buf *bytes.Buffer) ([]byte, string, string, error) {
	// uncompress request body for processing
	gzipReader := readerPool.Get().(*gzip.Reader)
	defer readerPool.Put(gzipReader)
	if err := gzipReader.Reset(tee); err != nil {
		return nil, "", "", err
	}
	var version, installID string
	firstIter := true
	cur := make([]byte, maxLogRead*2)
	// read gzipped file, discarding all but (at most) last 100kB
	var n int
	var err error
	for {
		// copy previous read into first maxLogRead bytes (size is 2*maxLogRead)
		copy(cur[0:maxLogRead], cur[maxLogRead:len(cur)])

		// read into last maxLogRead bytes (size is 2*maxLogRead)
		n, err = gzipReader.Read(cur[maxLogRead:len(cur)])
		if err != nil {
			if err != io.EOF {
				log.Println("error reading log from gzip reader:", err)
				break
			}
		}

		if n <= 0 {
			break
		}

		if firstIter {
			// extract version, installID from beginning of log
			version, _ = findVersion(cur)
			installID, _ = findInstallID(cur)
			firstIter = false
		}

		if err == io.EOF {
			break
		}
	}
	// clear rest of bytes after last read
	for i := maxLogRead + n; i < len(cur); i++ {
		cur[i] = 0
	}
	return cur, version, installID, nil
}

func (s *Server) reportCrash(key, platform, version, installID, errStr string, traceback []byte) error {
	link, err := url.Parse(s3URL)
	if err != nil {
		return err
	}
	link.Path = path.Join(link.Path, crashLogBucket, key)

	// send error to rollbar
	level := rollbar.ERR
	err = loggedClientError{errStr, traceback}
	data := map[string]interface{}{
		"platform":   platform,
		"install_id": installID,
		"link":       link.String(),
		"traceback":  string(traceback),
	}

	if s.crashRollbar.Token() == "" {
		// log error if we are in a debug environment
		log.Printf("rollbar [%s]: %v %v", level, err, data)
		return nil
	}

	s.m.Lock()
	defer s.m.Unlock()
	s.crashRollbar.SetCodeVersion(version)
	s.crashRollbar.ErrorWithExtras(level, err, data)
	return nil
}

func (s *Server) storeLogs(bucket, key string, contents []byte) (string, int) {
	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		log.Printf("error authenticating with AWS: %v\n", err)
		return "error authenticating with AWS", http.StatusInternalServerError
	}

	// this is transmitted as a raw string (gzipped) so that we can
	// open the resulting log files directly in the web browser for S3
	if err := s3.New(auth, region).Bucket(bucket).PutHeader(key, contents, map[string][]string{
		"Content-Type":     []string{"text/plain"},
		"Content-Encoding": []string{"gzip"},
	}, s3.Private); err != nil {
		log.Printf("error uploading logs to s3 for %s:%s: %v\n", bucket, key, err)
		return "error uploading to s3", http.StatusInternalServerError
	}
	return "", http.StatusOK
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
		log.Printf("empty machineid when posting logs for user %s\n", installid)
		return "", errors.Errorf("empty machineid")
	}

	key := fmt.Sprintf("%s/%s/%s/%s.gz", s.keyPrefix, installid, machine, filename)
	return key, nil
}

func (s *Server) createCrashKey(r *http.Request) (string, string, error) {
	machine := r.URL.Query().Get("machineid")
	if machine == "" {
		log.Println("empty machineid when posting logs")
		return "", "", errors.Errorf("empty machineid")
	}

	platform := r.URL.Query().Get("platform")
	if platform == "" {
		log.Println("empty platform when posting logs for machine")
		return "", "", errors.Errorf("empty platform")
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		log.Println("empty file name when posting logs for user")
		return "", "", errors.Errorf("empty file name")
	}

	key := fmt.Sprintf("%s/%s/%s/%s.gz", platform, s.keyPrefix, machine, filename)
	return key, platform, nil
}

// --

func newRollbarClient() *rollbar.Client {
	token := ""
	env := os.Getenv("ROLLBAR_ENV")
	if env == "" {
		env = "development"
	}
	release := os.Getenv("RELEASE")
	// Check if release is not set or is "test-instance" (set on test-N machines)
	if release != "" && release != "test-instance" {
		token = rollbarToken
	}

	return rollbar.New(token, env, release, "", "")
}

var (
	versionPattern   = regexp.MustCompile(`(?:version:\s)(\d\.\d+\.*\d*\.\d)`)
	installIDPattern = regexp.MustCompile(`(?:install ID:\s)((\w+\-)+\w+)`)
	errorPattern     = regexp.MustCompile(`(?m)((panic:)|(runtime:)|(Exception 0x)|(fatal error:)|(goroutine.[0-9]+)).*$`)
)

func findVersion(contents []byte) (string, error) {
	// version takes the form of X.YYYY.M.D.X or X.YYYYMMDD.X
	loc := versionPattern.FindSubmatch(contents)
	if len(loc) < 2 {
		return "", errors.Errorf("version not found")
	}
	return string(loc[1]), nil
}

func findInstallID(contents []byte) (string, error) {
	// install id is alpha-numeric characters separated by dashes
	loc := installIDPattern.FindSubmatch(contents)
	if len(loc) < 2 {
		return "", errors.Errorf("install id not found")
	}
	return string(loc[1]), nil
}

func findCrash(contents []byte) (string, []byte) {
	// find the beginning of the traceback
	loc := errorPattern.FindIndex(contents)
	if loc == nil {
		// traceback not found
		return "", nil
	}

	// limit traceback size
	tracebackSize := len(contents[loc[0]:])
	if tracebackSize > maxTracebackSize {
		tracebackSize = maxTracebackSize
	}

	return string(contents[loc[0]:loc[1]]), bytes.Trim(contents[loc[0]:][:tracebackSize], "\x00")
}
