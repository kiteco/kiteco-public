package localcode

import (
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	spooky "github.com/dgryski/go-spooky"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/diskcache"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/xtgo/uuid"
)

// BuildTimeout is longer than the default 30 second timeout for the non-fetching part of the build
const BuildTimeout = 60 * time.Second

// WorkerOptions contains options for the Worker
type WorkerOptions struct {
	NumWorkers int

	FileCacheRoot   string
	FileCacheSizeMB int

	ArtifactCacheRoot   string
	ArtifactCacheSizeMB int
}

// Worker is the top level object responsible for local code processing
type Worker struct {
	opts WorkerOptions

	requests *requestServer

	artifactcache *diskcache.Cache
	artifacts     *artifactServer

	filecache *diskcache.Cache
	content   *localfiles.ContentStore
}

// NewWorker returns a new Worker
func NewWorker(opts WorkerOptions, store *localfiles.ContentStore) (*Worker, error) {
	filecache, err := diskcache.Open(opts.FileCacheRoot, diskcache.Options{
		MaxSize:         int64(opts.FileCacheSizeMB * 1 << 20),
		BytesUntilFlush: 1 << 20, // flush stale entries after writing each 1MB
	})
	if err != nil {
		return nil, err
	}

	artifactcache, err := diskcache.Open(opts.ArtifactCacheRoot, diskcache.Options{
		MaxSize:         int64(opts.ArtifactCacheSizeMB * 1 << 20),
		BytesUntilFlush: 1 << 20, // flush stale entries after writing each 1MB
	})
	if err != nil {
		return nil, err
	}

	worker := &Worker{
		opts:          opts,
		requests:      newRequestServer(),
		artifactcache: artifactcache,
		artifacts:     newArtifactServer(artifactcache),
		filecache:     filecache,
		content:       store,
	}

	for i := 0; i < opts.NumWorkers; i++ {
		go worker.run(i)
	}

	return worker, nil
}

// SetupRoutes configures HTTP endpoints
func (w *Worker) SetupRoutes(mux *mux.Router) {
	w.requests.setupRoutes(mux)
	w.artifacts.setupRoutes(mux)
}

// --

func (w *Worker) run(i int) {
	log.Println("starting worker loop", i)
	defer log.Println("stopping worker loop", i)

	for {
		w.do()
	}
}

func (w *Worker) do() {
	defer func() {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
			log.Printf("do paniced: %s", ex)
		}
	}()

	// Get the next request to work on
	request, ok := w.requests.next()
	if !ok {
		time.Sleep(time.Second)
		return
	}

	var (
		uid                         int64
		id, mid, filename           string
		err                         error
		files                       []*localfiles.File
		l                           lang.Language
		result                      *BuilderResult
		latestFileUpdate            time.Time
		listDuration, buildDuration time.Duration
	)

	id = uuid.NewTime().String()
	uid = request.UserID
	mid = request.Machine

	logf := func(msg string, vals ...interface{}) {
		log.Printf("localcode.Worker (%d, %s, %s): %s", uid, mid, id, fmt.Sprintf(msg, vals...))
	}

	defer w.requests.completed(request)
	defer loopDuration.DeferRecord(time.Now())

	filename = request.latestFile()
	logf("processing request for %s", filename)

	// List all files for the (user_id, machine) pair
	ts := time.Now()
	files, err = w.content.Files.List(uid, mid)
	listDuration = time.Since(ts)
	if err != nil {
		listFilesErrorRatio.Hit()
		err = fmt.Errorf("error listing files: %v", err)
		logf(err.Error())
		w.artifacts.publishArtifact(errorArtifact(request, filename, id, err))
		return
	}

	listFilesErrorRatio.Miss()
	listFilesDuration.RecordDuration(listDuration)

	logf("got %d files from the database in %s", len(files), listDuration)

	for _, f := range files {
		if f.UpdatedAt.After(latestFileUpdate) {
			latestFileUpdate = f.UpdatedAt
		}
	}

	umf := userMachineFile{uid, mid, filename}
	if artifact, ok := w.artifacts.artifactFor(umf); ok {
		if !latestFileUpdate.After(artifact.LatestFileUpdate) {
			logf("no changes to rebuild for %s, satisfied by (%s, %s)", filename, artifact.UUID, artifact.Root)
			noNewFilesRatio.Hit()
			return
		}
	}

	noNewFilesRatio.Miss()

	// Find a builder
	l = lang.FromFilename(filename)
	builder, ok := getBuilder(l)
	if !ok {
		err = fmt.Errorf("could not find builder for %s, skipping", filename)
		logf(err.Error())
		w.artifacts.publishArtifact(errorArtifact(request, filename, id, err))
		return
	}

	// Build the Putter
	putter := newLocalPutter(w.artifactcache, id)

	// Build the FileGetter
	fileGetter := NewCachedFileGetter(w.filecache, w.content)

	ts = time.Now()

	// Actually build
	params := BuilderParams{
		UserID:     uid,
		MachineID:  mid,
		Filename:   filename,
		Files:      files,
		FileGetter: fileGetter,
		Putter:     putter,
	}

	err = kitectx.Background().WithTimeout(BuildTimeout, func(ctx kitectx.Context) (err error) {
		result, err = builder(ctx, params)
		return
	})
	buildDuration = time.Since(ts)
	if err != nil {
		if _, ok := err.(kitectx.ContextExpiredError); ok {
			builderTimeoutRatio.Hit()
		} else {
			builderTimeoutRatio.Miss()
		}

		builderErrorRatio.Hit()
		err = fmt.Errorf("builder error: %v", err)
		logf(err.Error())
		w.artifacts.publishArtifact(errorArtifact(request, filename, id, err))
		return
	}

	builderErrorRatio.Miss()
	builderDuration.RecordDuration(buildDuration)

	artifact := artifact{
		UUID:              id,
		UserID:            uid,
		Machine:           mid,
		Root:              result.Root,
		Language:          l,
		Files:             putter.files(),
		IndexedPathHashes: hashFilePaths(result.Files),
		LatestFileUpdate:  latestFileUpdate,
	}

	// Publish to artifacts server
	logf("publishing")
	w.artifacts.publishArtifact(artifact)
}

func errorArtifact(req *userRequest, fn, id string, err error) artifact {
	return artifact{
		UUID:     id,
		UserID:   req.UserID,
		Machine:  req.Machine,
		Root:     fn,
		Language: lang.FromFilename(fn),
		Error:    err.Error(),
	}
}

func hashFilePaths(files []*localfiles.File) map[string]bool {
	ret := make(map[string]bool)
	for _, file := range files {
		ret[filePathHash(file.Name)] = true
		if dir := path.Dir(file.Name); dir != "" {
			ret[filePathHash(withTrailingSlash(dir))] = true
		}
	}
	return ret
}

func filePathHash(path string) string {
	fp := spooky.Hash32([]byte(path))
	return fmt.Sprintf("%x", fp)
}

func withTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}
	return path + "/"
}
