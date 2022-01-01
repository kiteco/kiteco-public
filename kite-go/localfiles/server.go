package localfiles

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/gziphttp"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var (
	numWriters = envutil.GetenvDefaultInt("LOCALFILES_S3_WRITERS", 16)
)

// Server is an http wrapper around the file-server application.
type Server struct {
	store *ContentStore
	auth  *community.UserValidation

	uploadRequestChan chan UploadRequest
}

// NewServer creates a new server with the provided content store.
func NewServer(store *ContentStore, auth *community.UserValidation) *Server {
	s := &Server{
		store:             store,
		auth:              auth,
		uploadRequestChan: make(chan UploadRequest, 100),
	}

	go s.uploadLoop()

	return s
}

// SetupRoutes prepares handlers for the file-server API in the given Router.
func (s *Server) SetupRoutes(mux *mux.Router) {
	mux.HandleFunc("/files", s.auth.Wrap(gziphttp.Wrap(s.HandleFiles))).Methods("GET")
	mux.HandleFunc("/filestream", s.auth.Wrap(gziphttp.Wrap(s.HandleFileStream))).Methods("GET")

	// TODO: deprecate
	mux.HandleFunc("/files/{machineid}/purge", s.auth.Wrap(gziphttp.Wrap(s.HandlePurgeFiles))).Methods("GET")
	mux.HandleFunc("/content", s.auth.Wrap(gziphttp.Wrap(s.HandleMissingContent))).Methods("POST")

	// Updated API
	mux.HandleFunc("/files/purge", s.auth.Wrap(gziphttp.Wrap(s.HandlePurgeFiles))).Methods("GET")
	mux.HandleFunc("/missing-content", s.auth.Wrap(gziphttp.Wrap(s.HandleMissingContent))).Methods("POST")
}

// HandleFiles shows all existing file objects.
func (s *Server) HandleFiles(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		handleFilesDuration.RecordDuration(time.Since(start))
	}()

	uid := community.GetUser(r).ID
	machine := community.GetMachine(r)

	files, err := s.store.Files.List(uid, machine)

	if time.Since(start) > time.Minute {
		log.Printf("HandleFiles took %s; (%d, %s) returned %d files, err: %s",
			time.Since(start), uid, machine, len(files), err)
	}

	listFilesCount.Record(int64(len(files)))

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(files); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleFileStream lists files for the authenticated user. It emits multiple
// json objects, one for each file, instead of a list. This allows us to stream
// the response to the client instead of having to buffer it on the server.
func (s *Server) HandleFileStream(w http.ResponseWriter, r *http.Request) {
	defer handleFileStreamDuration.DeferRecord(time.Now())

	uid := community.GetUser(r).ID
	machine := community.GetMachine(r)

	files := s.store.Files.ListChan(r.Context(), uid, machine)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	enc := json.NewEncoder(w)
	for file := range files {
		err := enc.Encode(file)
		if err != nil {
			log.Printf("error encoding file for (%d, %s) %+v: %s", uid, machine, file, err)
			return
		}
	}
}

// HandleCreateFile creates file objects for the batch of FileEvents sent in
// the request. If a file already exists for the user, it updates the file
// content hash. Any FileEvent that includes the Content field is added to
// the hashed files database.
func (s *Server) HandleCreateFile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		handleCreateFileDuration.RecordDuration(time.Since(start))
	}()

	var err error
	var reader io.Reader
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		reader = r.Body
	}

	var ur UploadRequest
	dec := json.NewDecoder(reader)
	if err := dec.Decode(&ur); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ur.start = time.Now()
	ur.userID = community.GetUser(r).ID
	ur.machine = community.GetMachine(r)
	for _, f := range ur.Files {
		f.UserID = ur.userID
	}

	select {
	case s.uploadRequestChan <- ur:
		handleUploadRequestDropRate.Miss()
	default:
		handleUploadRequestDropRate.Hit()
		http.Error(w, "upload queue full", http.StatusRequestTimeout)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// HandlePurgeFiles deleted all files for the provided user id and machine id.
func (s *Server) HandlePurgeFiles(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		handlePurgeFilesDuration.RecordDuration(time.Since(start))
	}()

	uid := community.GetUser(r).ID
	machine := community.GetMachine(r)

	err := s.store.Files.DeleteUserMachine(uid, machine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleMissingContent takes as input a list of hashed files and returns a
// list of file hashes that don't exist.
func (s *Server) HandleMissingContent(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		handleMissingContentDuration.RecordDuration(time.Since(start))
	}()

	var hashes []string
	if err := json.NewDecoder(r.Body).Decode(&hashes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mark all files as not missing - this prevents the client from uploading new content
	var missing []string
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(missing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// --

func (s *Server) handleUploadRequest(ur UploadRequest) (err error) {
	defer func(err *error) {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
			*err = fmt.Errorf("panic recovered: %v", ex)
		}
	}(&err)

	start := time.Now()
	defer handleUploadRequestDuration.DeferRecord(start)

	handleUploadRequestContentBlobs.Record(int64(len(ur.Contents)))
	handleUploadRequestFileChanges.Record(int64(len(ur.Files)))

	blobStart := time.Now()

	validFile := make(map[string]bool)    // file passes language filter
	validContent := make(map[string]bool) // content passes language filter
	for _, event := range ur.Files {
		if lang.FromFilename(event.Name) == lang.Python {
			validFile[event.Name] = true
			validContent[event.HashedContent] = true
		}
	}

	var putFailed []string
	for hash, content := range ur.Contents {
		if !validContent[hash] {
			continue
		}
		if content == nil {
			continue
		}

		err = s.store.putContent(hash, content.Content)
		if err != nil {
			// Record write failures so we don't update the DB below
			validContent[hash] = false
			log.Println(err)
			putFailed = append(putFailed, hash)
			continue
		}
	}

	handleUploadRequestContentDuration.RecordDuration(time.Since(blobStart))

	// Update database
	var names []string
	var modified, removed []*FileEvent
	var missingContent []string
	for _, event := range ur.Files {
		// Don't update the DB if the write failed (modified events only)
		if event.Type == ModifiedEvent && (!validContent[event.HashedContent] || !validFile[event.Name]) {
			continue
		}

		// Check if content exists in upload request
		_, exists := ur.Contents[event.HashedContent]
		if !exists {
			// Check if content exists in the content hash set
			exists, _ = s.store.Exists(event.HashedContent)
		}
		if !exists {
			// If the content is not accounted for, do not write to db
			handleUploadContentMissing.Hit()
			log.Printf("missing content for hash %s in request", event.HashedContent)
			missingContent = append(missingContent, event.HashedContent)
			continue
		}
		handleUploadContentMissing.Miss()

		switch event.Type {
		case ModifiedEvent:
			modified = append(modified, event)
		case RemovedEvent:
			removed = append(removed, event)
		}

		names = append(names, event.Name)
	}

	dbStart := time.Now()
	err = s.store.Files.BatchCreateOrUpdate(modified)
	if err != nil {
		rollbar.Error(fmt.Errorf("error during FileManager.BatchCreateOrUpdate: %s", err.Error()), ur.userID, ur.machine)
		log.Println("error during FileManager.BatchCreateOrUpdate:", err)
	}
	handleUploadRequestModifiedDuration.RecordDuration(time.Since(dbStart))

	rmStart := time.Now()
	var deleteFailed []string
	for _, rem := range removed {
		err = s.store.Delete(rem.UserID, rem.Machine, rem.Name)
		if err != nil {
			rollbar.Error(fmt.Errorf("error during FileManager.Delete: %s", err.Error()), rem.UserID, rem.Machine, rem.Name)
			log.Println("error during ContentStore.Delete:", err)
			deleteFailed = append(deleteFailed, rem.HashedContent)
		}
	}
	handleUploadRequestRemovedDuration.RecordDuration(time.Since(rmStart))
	handleUploadRequestDatabaseDuration.RecordDuration(time.Since(dbStart))

	triggerObservers(ur.userID, ur.machine, names)

	return
}

// uploadLoop runs as a goroutine and triggers file updates that come in on the update channel.
func (s *Server) uploadLoop() {
	var err error
	for i := 0; i < numWriters; i++ {
		go func() {
			for {
				select {
				case ur := <-s.uploadRequestChan:
					handleUploadRequestWaitDuration.RecordDuration(time.Since(ur.start))
					err = s.handleUploadRequest(ur)
					if err != nil {
						log.Println("error in handleUploadRequest:", err)
					}
				}
			}
		}()
	}
}
