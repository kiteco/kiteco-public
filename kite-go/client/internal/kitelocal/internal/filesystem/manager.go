package filesystem

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/performance"
	"github.com/kiteco/kiteco/kite-go/client/internal/watch"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/client/readdir"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/filters"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	sendChangesRate    = 1 * time.Second
	clearCacheRate     = 1 * time.Hour
	directoryCacheSize = 10000
	fileCacheSize      = 1000

	permissionsWarningMessage = "Kite will request permission to look for Python libraries"
	permissionsWarningInfo    = `Kite accesses files in your home directory to automatically discover Python virtualenvs and to index your code. macOS may prompt you for permission.`
	permissionsWarningKey     = "suppressPermissionsWarning"
)

var supportedFileTypes = map[string]struct{}{".py": {}}

// Change contains a set of paths which were added, modified or removed
type Change struct {
	Paths []string
}

// Options defines the settings of a new filesystem manager
type Options struct {
	// RootDir is the directory to walk and to watch.
	RootDir string
	// KiteDir is the path to Kite's user data directory
	KiteDir string
	// An optional filter function to filter files in root dir
	IsFileAccepted func(string) bool
	// An optional filter function to filter directories in root dir
	IsDirAccepted func(string) bool
	// An optional filter function to filter library directories in root dir
	IsLibraryDir func(string) bool
	// DutyCycle is a float 0 < x <= 1.0 which defines the length of a duty cycle
	DutyCycle float64
	// WatcherMetrics is used to record the number of active watches
	WatcherMetrics *metrics.WatcherMetric
}

// Manager locates all supported files in a directory and listens for changes to those files
// Manager keeps a sorted list of files which can be accessed by callers.
type Manager struct {
	rootDir string
	kiteDir string

	watcher        *watcher
	watcherCancel  func()
	watcherMetrics *metrics.WatcherMetric

	libraryWalker *libraryWalker

	LocalFS *LocalFS
}

// NewManager returns a new manager
func NewManager(opts Options) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	purgeChan := make(chan []string, 1)

	watcher := newWatcher(ctx, opts, purgeChan)
	libWalker := newLibraryWalker(opts)
	fs := NewLocalFS(ctx, opts, purgeChan)

	return &Manager{
		rootDir:        opts.RootDir,
		kiteDir:        opts.KiteDir,
		watcher:        watcher,
		watcherCancel:  cancel,
		watcherMetrics: opts.WatcherMetrics,
		libraryWalker:  libWalker,
		LocalFS:        fs,
	}
}

// Name implement component interface Core
func (f *Manager) Name() string {
	return "fs watcher"
}

// Initialize implements component interface Initializer
func (f *Manager) Initialize(opts component.InitializerOptions) {
	if f.rootDir != "" {
		if fi, err := os.Stat(f.rootDir); err != nil || !fi.IsDir() {
			rollbar.Error(errors.New("unable to stat root directory"), f.rootDir, err.Error())
			log.Printf("error finding root directory: %s", err)
			return
		}
	}

	f.libraryWalker.initialize(opts)
	f.watcher.initialize(opts)
	f.LocalFS.initialize(opts)
}

// Terminate implements component interface Terminater
func (f *Manager) Terminate() {
	// terminate watcher by cancelling the context
	f.watcherCancel()
}

// RootDir returns the root directory tracked by the filesystem Manager
func (f *Manager) RootDir() string {
	return f.rootDir
}

// KiteDir returns Kite's user data directory
func (f *Manager) KiteDir() string {
	return f.kiteDir
}

// Files returns a copy of the list of supported files which were found inside the root dir
// The list is sorted
func (f *Manager) Files() []string {
	filesCount.Add(1)
	return f.libraryWalker.listDirs()
}

// FileSystem returns the local fs object
func (f *Manager) FileSystem() *LocalFS {
	return f.LocalFS
}

// Changes returns the channel were change events will be delivered
func (f *Manager) Changes() <-chan Change {
	return f.watcher.changeChan
}

// Watch registers a new watch for the given file or directory.
// Invocations of Watch and Unwatch must match.
// If Watch was called n times, then Unwatch must also be called n times to
// remove the registered watch.
func (f *Manager) Watch(fileOrDir string) error {
	if f.watcherMetrics != nil {
		defer func() {
			f.watcherMetrics.Set(f.watcher.watcherFS.WatchCount())
		}()
	}
	return f.watcher.watcherFS.Watch(fileOrDir)
}

// Unwatch removes a registered watch of filePath
// It only removes the watch of 'filePath' when the last call of Watch(filePath) is matched with
// a corresponding call of Unwatch(filePath).
// If Watch was called n times, then Unwatch must also be called n times to
// remove the registered watch.
func (f *Manager) Unwatch(fileOrDir string) error {
	if f.watcherMetrics != nil {
		defer func() {
			f.watcherMetrics.Set(f.watcher.watcherFS.WatchCount())
		}()
	}
	return f.watcher.watcherFS.Unwatch(fileOrDir)
}

// ReadyChan returns a channel to be notified when the initial walk of the root dir finished. The channel is closed once it's done.
// Reading from that channel will succeed.
func (f *Manager) ReadyChan() <-chan bool {
	return f.libraryWalker.readyChan
}

// StartWalk starts the library walk
func (f *Manager) StartWalk() {
	// walk filesystem in the background to avoid delays of Client startup
	go f.libraryWalker.backgroundWalk()
}

// --

// watcher watches for filesystem changes in the rootDir
type watcher struct {
	ctx       context.Context
	rootDir   string
	watcherFS watch.Filesystem
	// Channel where events of the filesystem watcher are send.
	changeChan chan Change
	// channel were a single event is sent as soon as the watcher is operational.
	// The channel is closed after the watcher is ready.
	readyChan chan bool
	// channel where events are sent to determine if cache purge is necessary
	purgeChan      chan []string
	m              sync.Mutex
	changes        map[string]bool
	isFileAccepted func(string) bool
}

// newWatcher creates a new filesystem Watcher
func newWatcher(ctx context.Context, opts Options, ch chan []string) *watcher {
	return &watcher{
		ctx:            ctx,
		rootDir:        opts.RootDir,
		changeChan:     make(chan Change, 100),
		readyChan:      make(chan bool, 1),
		changes:        make(map[string]bool),
		isFileAccepted: opts.IsFileAccepted,
		purgeChan:      ch,
	}
}

func (w *watcher) initialize(opts component.InitializerOptions) {
	if w.isFileAccepted == nil {
		w.isFileAccepted = func(path string) bool {
			e := filepath.Ext(path)
			_, ok := supportedFileTypes[e]
			return ok
		}
	}

	watcherOpts := watch.Options{
		Latency: 500 * time.Millisecond,
		OnDrop: func() {
			log.Println("dropping fs watcher event")
		},
	}

	eventCh := make(chan []watch.Event, 100) // buffered channel to receive watcher events
	var paths []string
	if w.rootDir != "" {
		paths = append(paths, w.rootDir)
	}
	fs, err := watch.NewFilesystem(w.ctx, paths, eventCh, w.readyChan, watcherOpts)
	if err != nil {
		rollbar.Error(errors.New("unable to watch root directory"), w.rootDir, err.Error())
		log.Printf("unable to watch root directory: %s", w.rootDir)
		return
	}

	w.watcherFS = fs

	// install watcher of fs events, changes are collected and send out to the output channel
	go w.watchRootDir(eventCh)
}

// watchRootDir listens to update and remove events to files inside of the configured root directory
func (w *watcher) watchRootDir(ch <-chan []watch.Event) {
	log.Printf("watching dir %s", w.rootDir)
	defer log.Printf("finished watching dir %s", w.rootDir)

	ticker := time.NewTicker(sendChangesRate)

	for {
		select {
		case <-w.ctx.Done():
			ticker.Stop()
			return

		case changes := <-ch:
			eventsPerGroup.Set(int64(len(changes)))
			for _, e := range changes {
				if !w.isFileAccepted(e.Path) {
					// skip changes to unsupported files
					continue
				}

				path, err := canonicalizePath(e.Path)
				if err != nil {
					log.Println(err)
					rollbar.Error(errors.New("unable to canonicalize path"), e.Path, err.Error())
					continue
				}

				switch {
				case e.Type == localfiles.RemovedEvent:
					deleteCount.Add(1)
					continue
				case e.Type == localfiles.ModifiedEvent:
					// notifications about new files are send as ModifiedEvent
					storeCount.Add(1)
				}

				w.m.Lock()
				w.changes[path] = true
				w.m.Unlock()
			}
		case <-ticker.C:
			// send changes
			w.m.Lock()
			if len(w.changes) > 0 {
				var paths []string
				for c := range w.changes {
					paths = append(paths, c)
				}
				select {
				case w.changeChan <- Change{Paths: paths}:
					w.purgeChan <- paths
				default:
					log.Printf("fs change notification dropped")
				}
				w.changes = make(map[string]bool)
			}
			w.m.Unlock()
		}
	}
}

// --

// libraryWalker walks the filesystem and stores library files
type libraryWalker struct {
	rootDir string
	// Signals when the initial walk of the filesystem completed.
	// This is a buffered channel to avoid blocking when sending to it
	readyChan chan bool
	// Signals when a walk should occur (at most once)
	walkChan chan bool
	// keeps all supported files located in rootDir
	dirs          sync.Map
	walked        uint64 // num dirs walked
	throttle      *dutycyclelimiter
	isDirAccepted func(string) bool
	isLibraryDir  func(string) bool
}

// newLibraryWalker creates a new LibraryWalker
func newLibraryWalker(opts Options) *libraryWalker {
	walkChan := make(chan bool, 1)
	walkChan <- true
	return &libraryWalker{
		rootDir:       opts.RootDir,
		isDirAccepted: opts.IsDirAccepted,
		isLibraryDir:  opts.IsLibraryDir,
		readyChan:     make(chan bool, 1),
		walkChan:      walkChan,
		throttle:      newDutycyclelimiter(opts.DutyCycle, time.Second),
	}
}

func (l *libraryWalker) initialize(opts component.InitializerOptions) {
	if l.isDirAccepted == nil {
		l.isDirAccepted = func(path string) bool {
			return !filters.IsFilteredDir(runtime.GOOS, path)
		}
	}

	if l.isLibraryDir == nil {
		l.isLibraryDir = func(path string) bool {
			return filters.IsLibraryDir(path)
		}
	}

	if !opts.Platform.IsNewInstall {
		go l.backgroundWalk()
	}
}

func (l *libraryWalker) backgroundWalk() {
	toWalk := <-l.walkChan
	if !toWalk {
		return
	}
	// signal that no further walks should occur
	close(l.walkChan)

	if runtime.GOOS == "darwin" && performance.OsVersion() >= "10.15" {
		platform.DispatchWarning(permissionsWarningKey, permissionsWarningMessage, permissionsWarningInfo)
	}

	log.Printf("walking root dir %s", l.rootDir)
	defer log.Printf("finished walking root dir %s", l.rootDir)

	// reset directory count
	syncDirCount.Set(0)

	start := time.Now()
	l.walkDir(l.rootDir)
	numDirs := len(l.listDirs())
	duration := time.Since(start)
	clienttelemetry.KiteTelemetry("Background Library Walk Completed", map[string]interface{}{
		"walked_dirs":    atomic.LoadUint64(&l.walked),
		"library_dirs":   numDirs,
		"scanned_dirs":   numDirs, // deprecated
		"since_start_ns": duration,
	})
	log.Printf("Library dirs: %d", numDirs)
	log.Printf("Walk took: %s", time.Since(start))
	// signal that the initial walk finished
	l.readyChan <- true
	close(l.readyChan)
}

func (l *libraryWalker) listDirs() []string {
	var dirs []string
	l.dirs.Range(func(path, _ interface{}) bool {
		dirs = append(dirs, path.(string))
		return true
	})

	sort.Strings(dirs)
	return dirs
}

// walkDir traverses all subdirectories and files inside of walkDir and stores library directories it finds as canonicalized paths. Input paths can be unix or windows paths.
func (l *libraryWalker) walkDir(path string) {
	if !l.isDirAccepted(path) {
		return
	}

	syncDirCount.Add(1)
	atomic.AddUint64(&l.walked, 1)

	// throttle the walk
	l.throttle.Take()

	for _, e := range readdir.List(path) {
		filePath := filepath.Join(path, e.Path)

		var isDir = e.IsDir
		if !e.DTypeEnabled {
			walkStatCount.Add(1)
			if fi, err := os.Lstat(filePath); err != nil {
				// skip entries we can't process
				log.Printf("lib walkDir: error %s", err.Error())
				continue
			} else {
				isDir = fi.IsDir()
			}
		}

		canonPath, err := canonicalizePath(filePath)
		if err != nil {
			log.Println(err)
			rollbar.Error(errors.New("unable to canonicalize path"), filePath, err.Error())
			continue
		}
		if isDir {
			if l.isLibraryDir(canonPath) {
				// stop walking once library dir found
				storeCount.Add(1)
				l.dirs.Store(canonPath, true)
			} else {
				l.walkDir(filePath)
			}
		}
	}
}

// --

// LocalFS implements the FileSystem API
type LocalFS struct {
	isDirAccepted  func(string) bool
	isLibraryDir   func(string) bool
	isFileAccepted func(string) bool
	dirCache       *lru.Cache
	fileCache      *lru.Cache
	purgeChan      chan []string
	ctx            context.Context
}

// NewLocalFS creates a new LocalFS
func NewLocalFS(ctx context.Context, opts Options, ch chan []string) *LocalFS {
	dirCache, _ := lru.New(directoryCacheSize)
	fileCache, _ := lru.New(fileCacheSize)
	return &LocalFS{
		isDirAccepted:  opts.IsDirAccepted,
		isLibraryDir:   opts.IsLibraryDir,
		isFileAccepted: opts.IsFileAccepted,
		dirCache:       dirCache,
		fileCache:      fileCache,
		purgeChan:      ch,
		ctx:            ctx,
	}
}

func (fs *LocalFS) initialize(opts component.InitializerOptions) {
	if fs.isDirAccepted == nil {
		fs.isDirAccepted = func(path string) bool {
			// Make sure path is a native path
			localPath, err := localpath.FromUnix(path)
			if err != nil {
				log.Printf("error converting path from unix: %s", err.Error())
				return false
			}
			return !filters.IsFilteredDir(runtime.GOOS, localPath)
		}
	}
	if fs.isLibraryDir == nil {
		fs.isLibraryDir = func(path string) bool {
			return filters.IsLibraryDir(path)
		}
	}
	if fs.isFileAccepted == nil {
		fs.isFileAccepted = func(path string) bool {
			e := filepath.Ext(path)
			_, ok := supportedFileTypes[e]
			return ok
		}
	}
	// watch for changes that trigger a cache purge
	go fs.watchCaches()
}

// Stat implements FileSystem Stat. If the file does not exist, it returns an error.
// Since FileInfo is not used in file selection, it returns empty FileInfo.
func (fs *LocalFS) Stat(path string) (localcode.FileInfo, error) {
	fi := localcode.FileInfo{}
	localPath, err := localpath.FromUnix(path)
	if err != nil {
		return fi, err
	}
	if !filepath.IsAbs(localPath) {
		return fi, localcode.ErrNonAbsolutePath
	}

	localDir := filepath.Dir(localPath)

	// check if directory exists in directory cache
	if val, ok := fs.getDir(localDir); ok {
		if !val {
			return fi, os.ErrNotExist
		}
	}

	// check if file exists in file cache
	if fs.fileCache != nil {
		val, ok := fs.fileCache.Get(localPath)
		if ok {
			if val.(bool) {
				return fi, nil
			}
			return fi, os.ErrNotExist
		}
	}

	_, fileErr := os.Stat(localPath)

	// if the file does not exist, add its directory to the directory cache if needed
	if os.IsNotExist(fileErr) {
		err = fs.statDir(localDir)
		if err != nil {
			return fi, err
		}
	}

	// add file to file cache
	fs.fileCache.Add(localPath, fileErr == nil)

	return fi, fileErr
}

// Glob returns files matching the input pattern rooted at the input directory.
func (fs *LocalFS) Glob(root, pattern string) ([]string, error) {
	localPath, err := localpath.FromUnix(root)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(localPath) {
		return nil, localcode.ErrNonAbsolutePath
	}

	patternPath := filepath.Join(localPath, pattern)
	found, err := filepath.Glob(patternPath)
	if err != nil {
		return nil, err
	}
	var canonFound []string
	for _, p := range found {
		canonPath, err := canonicalizePath(p)
		if err != nil {
			log.Println(err)
			continue
		}
		canonFound = append(canonFound, canonPath)
	}
	return canonFound, nil
}

// Walk recursively walks root, applying the walkFn to all entries. It is modeled on filepath.Walk.
// If root is a directory, the WalkFunc is called on its contents in the following order:
// errored entries, files, dirs.
// Paths passed to the WalkFunc are canonicalized.
func (fs *LocalFS) Walk(ctx kitectx.Context, root string, walkFn localcode.WalkFunc) error {
	ctx.CheckAbort()

	prefix := slashed(root)
	localPath, err := localpath.FromUnix(prefix)
	if err != nil {
		err = walkFn(localPath, localcode.FileInfo{IsDir: false}, err)
	} else {
		fi, err := os.Lstat(localPath)
		if err != nil {
			err = walkFn(localPath, localcode.FileInfo{IsDir: false}, err)
		} else {
			err = fs.walk(ctx, localPath, localcode.FileInfo{IsDir: fi.IsDir()}, walkFn)
		}
	}
	if err == localcode.ErrSkipDir {
		return nil
	}
	return err
}

type entry struct {
	path string
	info localcode.FileInfo
}

func (fs *LocalFS) walk(ctx kitectx.Context, root string, fi localcode.FileInfo, walkFn localcode.WalkFunc) error {
	ctx.CheckAbort()

	if fi.IsDir {
		dirAccepted := fs.isDirAccepted(root)

		canonPath, err := canonicalizePath(root)
		if err != nil {
			log.Println(err)
			rollbar.Error(errors.New("unable to canonicalize path"), root, err.Error())
			return err
		}
		if fs.isLibraryDir(canonPath) {
			// skip library directories
			if err := walkFn(canonPath, fi, localcode.ErrLibDir); err != nil {
				// allow caller to determine what to do with library directories
				return err
			}
		}

		if err := walkFn(root, fi, nil); err != nil {
			return err
		}

		var files, dirs []entry
		for _, e := range readdir.List(root) {
			filePath := filepath.Join(root, e.Path)
			entryInfo := localcode.FileInfo{IsDir: e.IsDir}

			if !e.DTypeEnabled {
				fi, err := os.Lstat(filePath)
				if err != nil {
					// allow caller to decide how to handle errored entries
					err = errors.Errorf("fs walkDir: error %s", err.Error())
					if err := walkFn(filePath, entryInfo, err); err != nil && err != localcode.ErrSkipDir {
						return err
					}
					// skip entries we can't stat if caller hasn't stopped walk
					continue
				}
				entryInfo.IsDir = fi.IsDir()
				entryInfo.Size = fi.Size()
			}

			if entryInfo.IsDir {
				// only recursively walk if directory is accepted
				if dirAccepted {
					dirs = append(dirs, entry{path: filePath, info: entryInfo})
				}
			} else {
				files = append(files, entry{path: filePath, info: entryInfo})
			}
		}

		// walk files before directories
		for _, entry := range append(files, dirs...) {
			err = fs.walk(ctx, entry.path, entry.info, walkFn)
			if err != nil {
				if !entry.info.IsDir || err != localcode.ErrSkipDir {
					return err
				}
			}
		}
	} else {
		if !fs.isFileAccepted(root) {
			return walkFn(root, fi, localcode.ErrUnacceptedFile)
		}
		canonPath, err := canonicalizePath(root)
		if err != nil {
			log.Println(err)
			rollbar.Error(errors.New("unable to canonicalize path"), root, err.Error())
			return err
		}
		return walkFn(canonPath, fi, nil)
	}
	return nil
}

func slashed(path string) string {
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

func (fs *LocalFS) getDir(dir string) (bool, bool) {
	if fs.dirCache == nil {
		return false, false
	}
	if val, ok := fs.dirCache.Get(dir); ok {
		return val.(bool), ok
	}
	return false, false
}

func (fs *LocalFS) statDir(dir string) error {
	val, ok := fs.getDir(dir)
	if ok {
		if val {
			return nil
		}
		return os.ErrNotExist
	}

	_, err := os.Stat(dir)
	fs.dirCache.Add(dir, err == nil)
	return err
}

func (fs *LocalFS) watchCaches() {
	ticker := time.NewTicker(clearCacheRate)
	defer ticker.Stop()

	for {
		select {
		case <-fs.ctx.Done():
			return
		case changes := <-fs.purgeChan:
			for _, path := range changes {
				localPath, err := localpath.FromUnix(path)
				if err != nil {
					continue
				}
				if val, ok := fs.dirCache.Peek(filepath.Dir(localPath)); ok && !val.(bool) {
					fs.dirCache.Purge()
				}
				if val, ok := fs.fileCache.Peek(localPath); ok && !val.(bool) {
					fs.fileCache.Purge()
				}
			}
		case <-ticker.C:
			fs.dirCache.Purge()
			fs.fileCache.Purge()
		}
	}
}

// --

// canonicalize paths for platform compatibility
func canonicalizePath(path string) (string, error) {
	remotePath, err := localpath.ToUnix(path)
	if err != nil {
		return "", errors.Errorf("error canonicalizing path: %s", err)
	}

	// Lowercase paths on windows to ensure consistent casing.
	if runtime.GOOS == "windows" {
		remotePath = strings.ToLower(remotePath)
	}

	return remotePath, nil
}
