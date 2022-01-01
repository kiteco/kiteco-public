package indexing

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/filesystem"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/client/internal/performance"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

const sampleCPURate = 500 * time.Millisecond

type artifact struct {
	requestPath       string
	rootPath          string
	object            localcode.Cleanuper
	fileHashes        map[uint64]bool
	libraryFileHashes map[uint64]bool
	// watchedDirs is the set of directories which were registered with the watcher
	watchedDirs []string
}

func (a *artifact) Contains(path string) bool {
	// may be directory or file path
	return a.fileHashes[filePathHash(path)] || a.fileHashes[filePathHash(withTrailingSlash(path))] || a.libraryFileHashes[filePathHash(path)] || a.libraryFileHashes[filePathHash(withTrailingSlash(path))]
}

// --
type filesystemAPI interface {
	FileSystem() *filesystem.LocalFS
	Files() []string
	Changes() <-chan filesystem.Change
	RootDir() string
	KiteDir() string

	// WatchFile registers a new watch for the given file or directory.
	// If WatchFile was called n times, then UnwatchFile must also be called n times to
	// remove the registered watch.
	Watch(fileOrDir string) error
	// UnwatchFile removes a registered watch of filePath
	// If WatchFile was called n times, then UnwatchFile must also be called n times to
	// remove the registered watch.
	Unwatch(fileOrDir string) error
}

// --

type manager struct {
	fs          filesystemAPI
	permissions component.PermissionsManager
	ids         userids.IDs

	rw       sync.RWMutex
	builders map[lang.Language]localcode.BuilderFunc

	requests        sync.Map
	requestNotifier chan struct{}

	changes sync.Map

	artifacts     sync.Map
	unsavedFiles  sync.Map
	filteredPaths sync.Map
	timedOutFiles sync.Map

	ctxCancel func()
	debug     bool
	pool      *workerpool.Pool
}

func newManager(ctx context.Context, fs filesystemAPI, ids userids.IDs) *manager {
	childCtx, cancel := context.WithCancel(ctx)
	m := &manager{
		fs:              fs,
		ids:             ids,
		builders:        make(map[lang.Language]localcode.BuilderFunc),
		requestNotifier: make(chan struct{}, 1),
		pool:            workerpool.NewWithCtx(childCtx, 1),
		ctxCancel:       cancel,
	}
	go m.indexingLoop(childCtx)
	return m
}

// Terminate releases resources used by this manager
func (m *manager) Terminate() {
	if m.ctxCancel != nil {
		m.ctxCancel()
		m.ctxCancel = nil
	}
}

func (m *manager) reset() {
	m.artifacts.Range(func(key, value interface{}) bool {
		artifact := value.(*artifact)
		artifact.object.Cleanup()
		for _, d := range artifact.watchedDirs {
			_ = m.fs.Unwatch(d)
		}

		m.artifacts.Delete(key)
		return true
	})
}

func (m *manager) registerBuilder(lang lang.Language, b localcode.BuilderFunc) {
	m.rw.Lock()
	defer m.rw.Unlock()
	m.builders[lang] = b
}

func (m *manager) getBuilder(lang lang.Language) (localcode.BuilderFunc, bool) {
	m.rw.RLock()
	defer m.rw.RUnlock()
	b := m.builders[lang]
	return b, b != nil
}

func (m *manager) allArtifacts() []*artifact {
	var artifacts []*artifact
	m.artifacts.Range(func(key, value interface{}) bool {
		artifacts = append(artifacts, value.(*artifact))
		return true
	})

	return artifacts
}

func (m *manager) anyArtifact() (*artifact, error) {
	var a *artifact
	m.artifacts.Range(func(key, value interface{}) bool {
		a = value.(*artifact)
		return false
	})

	if a == nil {
		return nil, localcode.ErrNoArtifact("no containing artifact found")
	}

	return a, nil

}

type changeBundle struct {
	path string
	ts   time.Time
}

type requestBundle struct {
	path       string
	trackExtra interface{}
	ts         time.Time
}

func (m *manager) artifactThatContains(path string, makeRequest bool, trackExtra interface{}) (*artifact, error) {
	// Check for an artifact requested with this path
	obj, ok := m.artifacts.Load(path)
	if ok {
		return obj.(*artifact), nil
	}

	// Check for any artifact that contains this path
	var a *artifact
	m.artifacts.Range(func(key, value interface{}) bool {
		if value.(*artifact).Contains(path) {
			if a != nil {
				// if file belongs to multiple indices, return index with the most files
				if len(value.(*artifact).fileHashes) < len(a.fileHashes) {
					return true
				}
			}
			a = value.(*artifact)
		}
		return true
	})

	if a != nil {
		return a, nil
	}

	// Make a request for this artifact if we didn't find one and makeRequest is set
	if makeRequest {
		m.requests.Store(path, requestBundle{
			path:       path,
			trackExtra: trackExtra,
			ts:         time.Now(),
		})

		select {
		case m.requestNotifier <- struct{}{}:
		default:
			m.logf("dropped request for %s", path)
		}
	}

	return nil, localcode.ErrNoArtifact("no containing artifact found")
}

func (m *manager) indexingLoop(ctx context.Context) {
	for {
		if err := m.changeOrRequest(ctx); err != nil {
			return
		}
	}
}

func (m *manager) changeOrRequest(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			rollbar.PanicRecovery(r)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-m.requestNotifier:
		m.pool.Add([]workerpool.Job{
			func() error {
				return m.handleRequest(time.Now())
			},
		})

	case change := <-m.fs.Changes():
		m.logf("indexer received changes:")
		for _, path := range change.Paths {
			m.logf(" - changed: %s", path)
		}

		rebuild := m.outdatedArtifacts(change)
		for _, re := range rebuild {
			m.changes.Store(re, changeBundle{
				path: re,
				ts:   time.Now(),
			})
		}
		m.pool.Add([]workerpool.Job{
			func() error {
				m.handleFileChange(time.Now())
				return nil
			},
		})
	}

	return nil
}

// handleRequest updates after a notification about a new request was received
func (m *manager) handleRequest(addedTs time.Time) error {
	// Collect all the requests
	var requests []requestBundle
	m.requests.Range(func(_, value interface{}) bool {
		requests = append(requests, value.(requestBundle))
		return true
	})

	// Sort so most recent request is at the top
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].ts.After(requests[j].ts)
	})

	m.logf("handling %d pending requests", len(requests))
	for _, req := range requests {
		m.logf(" - requested: %s", req.path)
	}

	// For each request, make sure we don't have an artifact that satisfies the
	// request, then build an artifact.
	for _, request := range requests {
		// filter requests
		nativePath, err := localpath.FromUnix(request.path)
		if err != nil {
			m.logf("could not convert %s to native path: %v", request.path, err)
			m.requests.Delete(request.path)
			_, exists := m.filteredPaths.Load(request.path)
			if !exists {
				clienttelemetry.KiteTelemetry("Index Build Filtered", map[string]interface{}{
					"reason": "from unix error",
				})
				m.filteredPaths.Store(request.path, true)
			}
			continue
		}
		_, timedOut := m.timedOutFiles.Load(request.path)
		if timedOut {
			m.logf("build previously timed out for %s", request.path)
			m.requests.Delete(request.path)
			_, exists := m.filteredPaths.Load(request.path)
			if !exists {
				clienttelemetry.KiteTelemetry("Index Build Filtered", map[string]interface{}{
					"reason": "timed out",
				})
				m.filteredPaths.Store(request.path, true)
			}
			continue
		}
		if _, err := os.Stat(nativePath); os.IsNotExist(err) {
			m.logf("file does not exist, dropping request for %s", request.path)
			m.requests.Delete(request.path)
			//fixme add watch for unsaved file?
			m.unsavedFiles.Store(request.path, true)
			_, exists := m.filteredPaths.Load(request.path)
			if !exists {
				clienttelemetry.KiteTelemetry("Index Build Filtered", map[string]interface{}{
					"reason": "file does not exist",
				})
				m.filteredPaths.Store(request.path, true)
			}
			continue
		}
		if _, err := m.artifactThatContains(request.path, false, nil); err == nil {
			m.logf("removing stale request for %s", request.path)
			m.requests.Delete(request.path)
			_, exists := m.filteredPaths.Load(request.path)
			if !exists {
				clienttelemetry.KiteTelemetry("Index Build Filtered", map[string]interface{}{
					"reason": "stale request",
				})
				m.filteredPaths.Store(request.path, true)
			}
			continue
		}

		err = m.build(request.path, "request", addedTs, request.trackExtra)
		if err != nil {
			log.Println("!! error building artifact from request:", err)
			if kitectx.IsDeadlineExceeded(err) {
				m.timedOutFiles.Store(request.path, true)
			}
		}

		m.requests.Delete(request.path)
	}

	return nil
}

func (m *manager) outdatedArtifacts(change filesystem.Change) []string {
	// Loop over artifacts to determine if any artifacts need to be rebuilt
	// because of the paths changed here.
	var rebuild []string
	m.artifacts.Range(func(key, value interface{}) bool {
		for _, path := range change.Paths {
			if value.(*artifact).Contains(path) {
				rebuild = append(rebuild, key.(string))
				return true
			}
		}
		return true
	})
	m.unsavedFiles.Range(func(key, _ interface{}) bool {
		for _, path := range change.Paths {
			if key.(string) == path {
				rebuild = append(rebuild, key.(string))
				m.unsavedFiles.Delete(path)
				return true
			}
		}
		return true
	})
	return rebuild
}

// handleChange is called when a filesystem events was received
func (m *manager) handleFileChange(addedTs time.Time) {
	// Collect all the changes
	var rebuild []changeBundle
	m.changes.Range(func(_, value interface{}) bool {
		rebuild = append(rebuild, value.(changeBundle))
		return true
	})

	// Sort so most recent change is at the top
	sort.Slice(rebuild, func(i, j int) bool {
		return rebuild[i].ts.After(rebuild[j].ts)
	})

	m.logf("indexer rebuilding %d artifacts", len(rebuild))
	for _, re := range rebuild {
		m.logf(" - rebuilding: %s", re.path)
	}

	for _, re := range rebuild {
		m.changes.Delete(re.path)
		nativePath, err := localpath.FromUnix(re.path)
		if err != nil {
			m.logf("could not convert %s to native path: %v", re.path, err)
			_, exists := m.filteredPaths.Load(re.path)
			if !exists {
				clienttelemetry.KiteTelemetry("Index Build Filtered", map[string]interface{}{
					"reason": "from unix error",
				})
				m.filteredPaths.Store(re.path, true)
			}
			continue
		}
		_, timedOut := m.timedOutFiles.Load(re.path)
		if timedOut {
			m.logf("build previously timed out for %s", re.path)
			_, exists := m.filteredPaths.Load(re.path)
			if !exists {
				clienttelemetry.KiteTelemetry("Index Build Filtered", map[string]interface{}{
					"reason": "timed out",
				})
				m.filteredPaths.Store(re.path, true)
			}
			continue
		}
		if _, err := os.Stat(nativePath); os.IsNotExist(err) {
			m.logf("file does not exist, skipping build for %s", re.path)
			_, exists := m.filteredPaths.Load(re.path)
			if !exists {
				clienttelemetry.KiteTelemetry("Index Build Filtered", map[string]interface{}{
					"reason": "file does not exist",
				})
				m.filteredPaths.Store(re.path, true)
			}
			continue
		}
		err = m.build(re.path, "change", addedTs, nil)
		if err != nil {
			log.Println("!! error rebuilding artifact from changes:", err)
			if kitectx.IsDeadlineExceeded(err) {
				m.timedOutFiles.Store(re.path, true)
			}
		}
	}
}

func (m *manager) build(req, source string, addedTs time.Time, trackExtra interface{}) (err error) {
	start := time.Now()
	waitDuration := time.Since(addedTs)
	m.logf("building artifact from path %s", req)

	var libFilesUsed, libDirsUsed, libDirsFound []string
	var files []*localfiles.File
	var libMgr *filesystem.LibraryManager
	var watchedDirs []string
	buildInfo := make(map[string]int)
	buildDurations := make(map[string]time.Duration)
	parseInfo := &localcode.ParseInfo{}
	cpuInfoChan := make(chan CPUInfo, 1)

	defer func() {
		sinceStart := time.Since(start)
		m.logf("completed build (%s) for path %s", sinceStart, req)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		cpuInfo := <-cpuInfoChan
		var libInfo map[string]int
		if libMgr != nil {
			libInfo = libMgr.Stats()
		}
		clienttelemetry.KiteTelemetry("Index Build", map[string]interface{}{
			"extra":              trackExtra,
			"goos":               runtime.GOOS,
			"error":              errStr,
			"since_start_ns":     sinceStart,
			"library_dirs_found": len(libDirsFound),
			"library_dirs_used":  len(libDirsUsed),
			"files":              len(files),
			"watched_dirs_linux": len(watchedDirs),
			"library_files_used": len(libFilesUsed),
			"artifacts":          len(m.allArtifacts()),
			"build_durations":    buildDurations,
			"wait_duration_ns":   waitDuration,
			"source":             source,
			"parse_info":         parseInfo,
			"build_info":         buildInfo,
			"cpu_info":           cpuInfo,
			"library_info":       libInfo,
		})
		m.logf("Build durations: %v", buildDurations)
		m.logf("Wait time: %s", waitDuration)
		m.logf("Build info: %v", buildInfo)
		m.logf("Parse timeouts: %d", parseInfo.ParseTimeouts)
		m.logf("Parse failures: %d", parseInfo.ParseFailures)
		m.logf("Parse errors: %v", parseInfo.ParseErrors)
		m.logf("CPU info: %v", cpuInfo)
		m.logf("Files used: %d", len(files))
		m.logf("Library dirs used: %d", len(libDirsUsed))
		m.logf("Library files used: %d", len(libFilesUsed))
		m.logf("Library info: %v", libInfo)
		m.logf("Directories watched: %v", len(watchedDirs))
	}()

	doneChan := make(chan struct{})
	defer close(doneChan) // defer after the above so that the <-cpuInfoChan doesn't block
	go sampleCPU(doneChan, cpuInfoChan)

	defer func() { // defer this after the above so the panic error message is sent to Kite
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
			if err == nil {
				err = fmt.Errorf("panic during indexing build")
			} else {
				err = fmt.Errorf("panic during indexing build: %s", err)
			}
		}
	}()

	builder, ok := m.getBuilder(lang.FromFilename(req))
	if !ok {
		return fmt.Errorf("no builder/loader found")
	}

	libDirsFound = m.fs.Files()
	libMgr = filesystem.NewLibraryManager(m.fs.RootDir(), m.fs.KiteDir(), libDirsFound)

	params := localcode.BuilderParams{
		UserID:         m.ids.UserID(),
		MachineID:      m.ids.MachineID(),
		Filename:       req,
		LibraryFiles:   libDirsFound,
		LibraryManager: libMgr,
		FileGetter:     LocalGetter{},
		FileSystem:     m.fs.FileSystem(),
		Local:          true,
		BuildDurations: buildDurations,
		BuildInfo:      buildInfo,
		ParseInfo:      parseInfo,
	}

	m.logf("building...")

	var result *localcode.BuilderResult
	err = kitectx.Background().WithTimeout(localcode.BuildTimeout, func(ctx kitectx.Context) (err error) {
		result, err = builder(ctx, params)
		return
	})
	if err != nil {
		return err
	}
	files = result.Files
	libDirsUsed = result.LibraryDirs
	for _, f := range result.LibraryFiles {
		libFilesUsed = append(libFilesUsed, f.Name)
	}

	a := &artifact{
		requestPath:       req,
		object:            result.LocalArtifact,
		fileHashes:        hashFilePaths(files),
		libraryFileHashes: hashFilePaths(result.LibraryFiles),
	}

	m.artifacts.Range(func(key, value interface{}) bool {
		if key.(string) == req || value.(*artifact).Contains(req) {
			m.logf("purging %s", key.(string))
			m.artifacts.Delete(key)
			value.(*artifact).object.Cleanup()

			for _, d := range value.(*artifact).watchedDirs {
				_ = m.fs.Unwatch(d)
			}
		}
		return true
	})
	m.unsavedFiles.Range(func(key, _ interface{}) bool {
		path := key.(string)
		if path == req || a.Contains(path) {
			m.unsavedFiles.Delete(key)
		}
		return true
	})

	// collect dirs to watch and attach to artifact before it's saved
	watchedDirsMap := make(map[string]bool)
	watchedDirsMap[filepath.Dir(req)] = true
	for _, f := range files {
		watchedDirsMap[filepath.Dir(f.Name)] = true
	}
	for _, f := range result.LibraryFiles {
		watchedDirsMap[filepath.Dir(f.Name)] = true
	}
	// attach watched dirs to artifact
	for d := range watchedDirsMap {
		watchedDirs = append(watchedDirs, d)
	}
	a.watchedDirs = watchedDirs

	// watch file and related files of the new artifact, but only if the value was stored
	_, loaded := m.artifacts.LoadOrStore(req, a)
	if !loaded {
		m.logf("adding %d watched dirs for artifact %s", len(a.watchedDirs), req)
		for _, d := range a.watchedDirs {
			if err = m.fs.Watch(d); err != nil {
				// adding the watch failed, but we still keep the dir in watchedDirs
				// modifying the artifact after it's saved has side-effects
				// and calling Unwatch on a path without a watch will just ignore it
				//
				// If we fail to watch a directory for artifact A,
				// but succeed for B, and then the A is cleaned up.
				// Then the watch will be cleaned up even though artifact B still exists.
				// This is unlikely, but might become an issue.
				// In that case we have to safely update the watchedDirs.
				// imo (jansorg), ignoring this case has higher benefits than adding locking for the data update.
				log.Printf("error registering watch for %s: %v", d, err)
			}
		}
	}

	return nil
}

// CPUInfo contains cpu sample stats
type CPUInfo struct {
	Count int     `json:"count"`
	Sum   float64 `json:"sum"`
	Max   float64 `json:"max"`
}

func sampleCPU(doneChan chan struct{}, cpuInfoChan chan<- CPUInfo) {
	ticker := time.NewTicker(sampleCPURate)
	defer ticker.Stop()

	var info CPUInfo
	for {
		select {
		case <-doneChan:
			cpuInfoChan <- info
			return
		case <-ticker.C:
			sample := performance.CPUUsage()
			info.Count++
			info.Sum += sample
			if sample > info.Max {
				info.Max = sample
			}
		}
	}
}

func (m *manager) logf(msg string, objs ...interface{}) {
	if m.debug {
		log.Printf("!! "+msg, objs...)
	}
}

// --

func hashFilePaths(files []*localfiles.File) map[uint64]bool {
	ret := make(map[uint64]bool)
	for _, file := range files {
		ret[filePathHash(file.Name)] = true
		if dir := path.Dir(file.Name); dir != "" {
			ret[filePathHash(withTrailingSlash(dir))] = true
			if strings.HasSuffix(file.Name, "__init__.py") {
				// include parent of package so it can be located
				if parent := path.Dir(dir); parent != "" {
					ret[filePathHash(withTrailingSlash(parent))] = true
				}
			}
		}
	}
	return ret
}

func filePathHash(path string) uint64 {
	return spooky.Hash64([]byte(path))
}

func withTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}
	return path + "/"
}

// --

// LocalGetter implements localcode.Getter
type LocalGetter struct{}

// Get returns the content of the input file
func (l LocalGetter) Get(key string) ([]byte, error) {
	// TODO(tarak): EPIC HACK!! We treat the key as the path. This will work
	// because we are setting the file hashes in []*localfile.File to the actual
	// path. This is to avoid hacking the API (keeps hacks isolated to here)
	path := key
	localPath, err := localpath.FromUnix(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(localPath)
}
