package localcode

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// ErrNoArtifact is the error returned when no artifact could be fetched
type ErrNoArtifact string

// Error implements error
func (e ErrNoArtifact) Error() string {
	if e == "" {
		return "no artifact found"
	}
	return fmt.Sprintf("no artifact found: %s", string(e))
}

var maxLoadedArtifacts = 2

type loadedArtifact struct {
	artifact

	obj Cleanuper

	dirtyAt      time.Time
	loadedAt     time.Time
	accessedAt   time.Time
	markedIdleAt time.Time
}

func (l *loadedArtifact) dirty() bool {
	return l.dirtyAt.After(l.LatestFileUpdate)
}

type erroredArtifact struct {
	artifact
	erroredAt time.Time
}

// UserContext is an implementation of Context that uses a remote worker to retrieve
// localcode artifacts for a user/machine.
type UserContext struct {
	ctx    context.Context
	cancel context.CancelFunc

	userID  int64
	machine string

	requestClient  *requestClient
	artifactClient *artifactClient

	fw               sync.Mutex
	firstRequest     time.Time
	gotFirstArtifact bool

	rw           sync.RWMutex
	artifacts    map[string]*loadedArtifact
	idle         map[string]*loadedArtifact
	errors       map[string]*erroredArtifact
	lastRequests map[string]time.Time

	pw          sync.Mutex
	pollingOnce sync.Once
	polling     map[userMachineFile]time.Time
}

func newUserContext(uid int64, machine string, requestClient *requestClient, artifactClient *artifactClient) *UserContext {
	ctx, cancel := context.WithCancel(context.Background())
	uc := &UserContext{
		ctx:            ctx,
		cancel:         cancel,
		userID:         uid,
		machine:        machine,
		requestClient:  requestClient,
		artifactClient: artifactClient,
		artifacts:      make(map[string]*loadedArtifact),
		idle:           make(map[string]*loadedArtifact),
		errors:         make(map[string]*erroredArtifact),
		lastRequests:   make(map[string]time.Time),
		polling:        make(map[userMachineFile]time.Time),
	}
	uc.printf("created")
	return uc
}

// LatestFileDriver returns the latest available file driver of the file
func (u *UserContext) LatestFileDriver(unixFilepath string) (core.FileDriver, error) {
	return nil, errors.Errorf("only available for local code")
}

// ArtifactForFile returns a loaded artifact that can serve the provided file or directory path.
// This object should not be stored or persisted anywhere.
// Always request a new Artifact as updates may have been made by the localcode framework.
func (u *UserContext) ArtifactForFile(path string) (interface{}, error) {
	return u.artifactForFile(path, false)
}

// RequestForFile returns a loaded artifact that can serve the provided filename. If one is not found
// it will request one from the worker. This object should not be stored or persisted anywhere. Always
// request a new Artifact as updates may have been made by the localcode framework.
// trackExtra may be logged as tracking data in case of any error.
func (u *UserContext) RequestForFile(filename string, trackExtra interface{}) (interface{}, error) {
	return u.artifactForFile(filename, true)
}

// AnyArtifact returns the most general artifact that can be found. This object
// should not be stored or persisted anywhere. Always request a new Artifact as updates may have
// been made by the localcode framework.
func (u *UserContext) AnyArtifact() (interface{}, error) {
	u.rw.Lock()
	defer u.rw.Unlock()

	if len(u.artifacts) == 0 {
		return nil, ErrNoArtifact("")
	}

	var best *loadedArtifact
	for _, loaded := range u.artifacts {
		// Pick the newest artifact, to ensure we always reflect reindexing changes.
		if best == nil || loaded.loadedAt.After(best.loadedAt) {
			best = loaded
		}
	}

	best.accessedAt = time.Now()
	return best.obj, nil
}

// Status contains metadata about an artifact
type Status struct {
	Path       string
	Files      int
	FileHashes map[uint64]bool
}

// Status returns a list of Status objects for the provided user/machine
func (u *UserContext) Status(n int) []*Status {
	u.rw.Lock()
	defer u.rw.Unlock()

	var indices []*Status
	for _, loaded := range u.artifacts {
		if len(indices) >= n {
			break
		}
		indices = append(indices, &Status{
			Path:  loaded.Root,
			Files: len(loaded.Files),
		})
	}

	return indices
}

// StatusResponse is the response sent from the local code status endpoint
type StatusResponse struct {
	Indices []IndexResponse
}

// IndexResponse contains data on a local code artifact
type IndexResponse struct {
	Path       string          `json:"path"`        // path for current index
	Files      int             `json:"files"`       // number of files in index
	FileHashes map[uint64]bool `json:"file_hashes"` // file hashes in this index
}

// StatusResponse constructs the response for requests for local code index status
func (u *UserContext) StatusResponse(n int) *StatusResponse {
	statuses := u.Status(n)

	var resp StatusResponse
	for _, index := range statuses {
		resp.Indices = append(resp.Indices, IndexResponse{
			Path:  index.Path,
			Files: index.Files,
		})
	}

	return &resp
}

// Cleanup cleans up any state UserContext may be holding on to via artifacts
func (u *UserContext) Cleanup() error {
	u.printf("cleanup called")
	u.rw.Lock()
	defer u.rw.Unlock()

	u.cancel()

	var artifacts []*loadedArtifact
	for id, artifact := range u.artifacts {
		artifacts = append(artifacts, artifact)
		delete(u.artifacts, id)
	}

	for id, artifact := range u.idle {
		artifacts = append(artifacts, artifact)
		delete(u.idle, id)
	}

	if len(artifacts) > 0 {
		go u.cleanup(artifacts)
	}

	u.recordHadRequestArtifact()

	return nil
}

// --

func (u *UserContext) artifactForFile(filename string, request bool) (interface{}, error) {
	u.rw.Lock()
	defer u.rw.Unlock()

	// Temporary fix for windows case sensitivity issue
	if strings.HasPrefix(filename, "/windows/") {
		filename = strings.ToLower(filename)
	}

	var best *loadedArtifact
	for _, loaded := range u.artifacts {
		// Pick the newest matching artifact, to ensure we always reflect reindexing changes.
		if loaded.contains(filename) && (best == nil || loaded.loadedAt.After(best.loadedAt)) {
			best = loaded
		}
	}

	if best != nil {
		best.accessedAt = time.Now()
		if best.dirty() {
			usingDirtyArtifact.Hit()
			if request && time.Since(u.lastRequests[filename]) > requestRate {
				u.lastRequests[filename] = time.Now()
				go u.makeRequest(filename)
			}
		} else {
			usingDirtyArtifact.Miss()
		}
		return best.obj, nil
	}

	if request && time.Since(u.lastRequests[filename]) > requestRate {
		u.lastRequests[filename] = time.Now()
		go u.makeRequest(filename)
	}

	var latestError *erroredArtifact
	for _, errored := range u.errors {
		if errored.contains(filename) && (latestError == nil || errored.erroredAt.After(latestError.erroredAt)) {
			latestError = errored
		}
	}
	if latestError != nil {
		return nil, ErrNoArtifact(latestError.Error)
	}

	return nil, ErrNoArtifact("")
}

// --

func (u *UserContext) observeFileSync(filenames []string) {
	u.rw.Lock()
	defer u.rw.Unlock()
	for _, fn := range filenames {
		for _, artifact := range u.artifacts {
			if artifact.contains(fn) {
				artifact.dirtyAt = time.Now()
			}
		}
	}

	// Reset "no files selected" errors. This can occur if we attempt
	// to index files before file syncing is complete.
	for id, artifact := range u.errors {
		if strings.Contains(artifact.Error, "no files selected") {
			delete(u.errors, id)
		}
	}
}

var (
	requestRate = time.Second * 15
	pollingRate = time.Second * 5
)

func (u *UserContext) makeRequest(filename string) {
	if _, ok := u.erroredArtifact(filename); ok {
		return
	}

	umf := userMachineFile{u.userID, u.machine, filename}

	// Look for a matching artifact. If we find an artifact:
	// - If don't have the artifact, load it, and stop polling
	// - If we do have it, and our version is NOT dirty, stop polling
	//
	// Otherwise, we want to keep polling and allow a request to be made
	// so we rebuild an updated index.
	artifact, err := u.artifactClient.findArtifact(umf)
	if err == nil {
		if !u.containsArtifact(artifact.UUID) {
			u.printf("found artifact for %s, loading...", filename)
			u.loadArtifact(artifact)
			u.stopPollingFor(umf)
			return
		} else if !u.isDirty(artifact.UUID) {
			u.stopPollingFor(umf)
			return
		}
	}

	u.printf("requesting artifact for %s", filename)
	err = u.requestClient.requestArtifact(umf)
	if err != nil {
		u.printf("error requesting artifact: %v", err)
	} else {
		u.maybeFirstRequest()
		u.startPollingFor(umf)
		u.pollingOnce.Do(func() {
			go u.pollForArtifacts(u.ctx)
		})
	}
}

// --
func (u *UserContext) maybeFirstRequest() {
	u.fw.Lock()
	defer u.fw.Unlock()
	if u.firstRequest.IsZero() {
		u.printf("making first request")
		u.firstRequest = time.Now()
	}
}

func (u *UserContext) maybeFirstArtifact() {
	u.fw.Lock()
	defer u.fw.Unlock()
	if !u.firstRequest.IsZero() && !u.gotFirstArtifact {
		u.gotFirstArtifact = true
		d := time.Since(u.firstRequest)
		firstArtifactDuration.RecordDuration(d)

		// To help track down latency issues
		if d > time.Minute {
			u.printf("first artifact took %s!", d)
		}
	}
}

func (u *UserContext) recordHadRequestArtifact() {
	u.fw.Lock()
	defer u.fw.Unlock()

	if u.firstRequest.IsZero() {
		u.printf("never made artifact request")
		neverMadeRequest.Hit()
	} else {
		neverMadeRequest.Miss()
		if !u.gotFirstArtifact {
			u.printf("never got local code artifact")
			neverGotArtifact.Hit()
		} else {
			neverGotArtifact.Miss()
		}
	}
}

// --

func (u *UserContext) startPollingFor(umf userMachineFile) {
	u.pw.Lock()
	defer u.pw.Unlock()

	// Only add it if its not already there, so we track the *first* time
	// we started polling for a umf.
	if _, ok := u.polling[umf]; !ok {
		u.polling[umf] = time.Now()
	}
}

func (u *UserContext) stopPollingFor(umf userMachineFile) {
	u.pw.Lock()
	defer u.pw.Unlock()
	if ts, ok := u.polling[umf]; ok {
		d := time.Since(ts)
		allArtifactDuration.RecordDuration(time.Since(ts))

		// To help track down latency issues
		if d > time.Minute {
			u.printf("artifact took %s!", d)
		}
	}
	delete(u.polling, umf)
}

func (u *UserContext) toPoll() []userMachineFile {
	u.pw.Lock()
	defer u.pw.Unlock()
	var pending []userMachineFile
	for umf := range u.polling {
		pending = append(pending, umf)
	}
	return pending
}

func (u *UserContext) pollForArtifacts(ctx context.Context) {
	u.printf("starting pollForArtifacts")
	ticker := time.NewTicker(pollingRate)
	defer ticker.Stop()

	var lastpending int

	for {
		select {
		case <-ctx.Done():
			u.printf("stopping pollForArtifacts")
			return
		case <-ticker.C:
			pending := u.toPoll()

			// Only log when number of items we're polling for changes
			if len(pending) != lastpending {
				u.printf("polling for %d artifacts...", len(pending))
				lastpending = len(pending)
			}

			for _, umf := range pending {
				// Look for a matching artifact. If we find an artifact:
				// - If don't have the artifact, load it, and stop polling
				// - If we do have it, and our version is NOT dirty, stop polling
				//
				// Otherwise, we want to keep polling so we load an updated index.
				artifact, err := u.artifactClient.findArtifact(umf)
				if err == nil {
					if !u.containsArtifact(artifact.UUID) {
						u.printf("found artifact for %s while polling, loading...", umf.Filename)
						u.loadArtifact(artifact)
						u.stopPollingFor(umf)
					} else if !u.isDirty(artifact.UUID) {
						u.stopPollingFor(umf)
					}
				}
			}
		}
	}
}

// --

func (u *UserContext) loadArtifact(artifact artifact) {
	defer perLoaderDuration.DeferRecord(time.Now())
	defer func() {
		if r := recover(); r != nil {
			rollbar.PanicRecovery(r, artifact.UUID, artifact.UserID, artifact.Machine, artifact.Root)

			// Set the artifact as errored
			u.printf("loaded artifact panicked (%d, %s, %s) %s",
				artifact.UserID, artifact.Machine, artifact.UUID, artifact.Root)

			u.rw.Lock()
			defer u.rw.Unlock()
			artifact.Error = fmt.Sprintf("%v", r)
			u.errors[artifact.UUID] = &erroredArtifact{
				artifact:  artifact,
				erroredAt: time.Now(),
			}
		}
	}()

	if artifact.err() {
		u.rw.Lock()
		defer u.rw.Unlock()
		u.printf("loaded errored artifact %s", artifact.UUID)
		u.errors[artifact.UUID] = &erroredArtifact{
			artifact:  artifact,
			erroredAt: time.Now(),
		}
		return
	}

	loader, ok := getLoader(artifact.Language)
	if !ok {
		u.printf("no loader for %s found", artifact.Language.Name())
		return
	}

	getter := newArtifactGetter(artifact, u.artifactClient)
	obj, err := loader(getter)
	if err != nil {
		loaderErrorRatio.Hit()
		u.printf("loader for language %s returned error: %s", artifact.Language.Name(), err)

		u.rw.Lock()
		defer u.rw.Unlock()
		artifact.Error = fmt.Sprintf("error loading artifact: %v", err)
		u.errors[artifact.UUID] = &erroredArtifact{
			artifact:  artifact,
			erroredAt: time.Now(),
		}
		return
	}

	loaderErrorRatio.Miss()

	u.printf("loaded artifact (%s, %s)", artifact.UUID, artifact.Root)

	u.maybeFirstArtifact()

	u.rw.Lock()
	defer u.rw.Unlock()
	u.artifacts[artifact.UUID] = &loadedArtifact{
		artifact:   artifact,
		obj:        obj,
		loadedAt:   time.Now(),
		accessedAt: time.Now(),
	}

	// Ensure we only keep maxLoadedArtifacts around
	if len(u.artifacts) > maxLoadedArtifacts {
		var oldest *loadedArtifact
		for id, loaded := range u.artifacts {
			// Don't remove the artifact we just added
			if id == artifact.UUID {
				continue
			}

			// Find the least recently accessed artifact
			if oldest == nil || loaded.accessedAt.Before(oldest.accessedAt) {
				oldest = loaded
			}
		}

		if oldest != nil {
			u.markInactiveLocked([]*loadedArtifact{oldest})
		}
	}

	for _, errArtifact := range u.errors {
		if artifact.contains(errArtifact.Root) {
			indexWithErroredFile.Hit()
			return
		}
	}

	indexWithErroredFile.Miss()
}

func (u *UserContext) madeRequest() bool {
	u.fw.Lock()
	defer u.fw.Unlock()
	return !u.firstRequest.IsZero()
}

func (u *UserContext) gotArtifact() bool {
	u.fw.Lock()
	defer u.fw.Unlock()
	return u.gotFirstArtifact
}

func (u *UserContext) hasArtifact() bool {
	u.rw.RLock()
	defer u.rw.RUnlock()
	return len(u.artifacts) > 0
}

func (u *UserContext) hasErroredArtifact() bool {
	u.rw.RLock()
	defer u.rw.RUnlock()
	return len(u.errors) > 0
}

func (u *UserContext) hasDirtyArtifact() bool {
	u.rw.RLock()
	defer u.rw.RUnlock()
	for _, artifact := range u.artifacts {
		if artifact.dirty() {
			return true
		}
	}
	return false
}

func (u *UserContext) containsArtifact(id string) bool {
	u.rw.RLock()
	defer u.rw.RUnlock()
	_, ok1 := u.artifacts[id]
	_, ok2 := u.idle[id]
	_, ok3 := u.errors[id]
	return ok1 || ok2 || ok3
}

func (u *UserContext) isDirty(id string) bool {
	u.rw.RLock()
	defer u.rw.RUnlock()
	if artifact, ok := u.artifacts[id]; ok {
		return artifact.dirty()
	}
	return false
}

func (u *UserContext) erroredArtifact(filename string) (*erroredArtifact, bool) {
	u.rw.RLock()
	defer u.rw.RUnlock()
	for _, artifact := range u.errors {
		if artifact.Root == filename {
			return artifact, true
		}
	}
	return nil, false
}

// markInactive queues a symbol index up for cleanup
func (u *UserContext) markInactiveLocked(inactives []*loadedArtifact) {
	for _, inactive := range inactives {
		inactive.markedIdleAt = time.Now()
		delete(u.artifacts, inactive.UUID)
		u.idle[inactive.UUID] = inactive
	}
}

// cleanupInactive will mark artifacts as inactive and queue them up for cleanup. Note
// we don't immediately clean them up incase any client of UserContext was just given
// one of these artifacts. We allow 1 minute to make sure nothing is cleaned up while
// being used.
func (u *UserContext) cleanupInactive() {
	u.rw.Lock()
	defer u.rw.Unlock()

	var removed []*loadedArtifact

	// Scan artifacts for anything thats idle. Put into idle map
	for id, artifact := range u.artifacts {
		if time.Since(artifact.accessedAt) > maxInactiveArtifact {
			artifact.markedIdleAt = time.Now()
			delete(u.artifacts, id)
			u.idle[id] = artifact
		}
	}

	// Scan idle entries for anything older than a minute, actually cleanup now
	for id, artifact := range u.idle {
		if time.Since(artifact.markedIdleAt) > time.Minute {
			removed = append(removed, artifact)
			delete(u.idle, id)
		}
	}

	// Cleanup anything we may have removed
	if len(removed) > 0 {
		go u.cleanup(removed)
	}
}

func (u *UserContext) cleanup(removed []*loadedArtifact) {
	for _, rem := range removed {
		if !rem.dirtyAt.IsZero() && rem.accessedAt.After(rem.dirtyAt) {
			servedDirtyDuration.RecordDuration(rem.accessedAt.Sub(rem.dirtyAt))
		}

		u.printf("cleaning up artifact %s (%s)", rem.UUID, rem.Root)
		err := rem.obj.Cleanup()
		if err != nil {
			u.printf("error cleaning up artifact %s: %v", rem.UUID, err)
		}
	}
}

func (u *UserContext) printf(msg string, vars ...interface{}) {
	log.Printf("localcode.UserContext (%d, %s): %s", u.userID, u.machine, fmt.Sprintf(msg, vars...))
}
