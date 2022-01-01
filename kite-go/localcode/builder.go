package localcode

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/diskcache"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	builderLock sync.Mutex
	builders    = map[lang.Language]BuilderFunc{}
)

// RegisterBuilder registers a builder function for a language
func RegisterBuilder(l lang.Language, b BuilderFunc) {
	builderLock.Lock()
	defer builderLock.Unlock()
	builders[l] = b
}

func getBuilder(l lang.Language) (BuilderFunc, bool) {
	builderLock.Lock()
	defer builderLock.Unlock()
	b, ok := builders[l]
	return b, ok
}

// --

// FileGetter is an interface for getting file content from content hash key
// TODO(naman) merge this into the FileSystem interface
type FileGetter interface {
	Get(key string) ([]byte, error)
}

// Putter is an interface for making artifacts available
type Putter interface {
	Put(name string, content []byte) error
	PutWriter(name string) (io.WriteCloser, error)
}

// ErrSkipDir is used as a return value from WalkFuncs to indicate that
// the directory named in the call is to be skipped. It is not returned
// as an error by any function.
var ErrSkipDir = filepath.SkipDir

// ErrLibDir is passed to the WalkFunc when a library directory is skipped
var ErrLibDir = errors.New("skip library directory")

// ErrUnacceptedFile is passed to the WalkFunc when a file is not accepted
var ErrUnacceptedFile = errors.New("skip unaccepted file")

// ErrNonAbsolutePath is returned by Stat and Glob if they are called with a non-absolute path
var ErrNonAbsolutePath = errors.New("received a non-absolute path")

// WalkFunc is the type of the function called for each file or directory
// visited by Walk. It is modeled on filepath.WalkFunc.
type WalkFunc func(path string, fi FileInfo, err error) error

// FileSystem is an interface for interacting with a file system.
type FileSystem interface {
	Stat(path string) (FileInfo, error)
	Glob(dir, pattern string) ([]string, error)
	// Walk recursively walks path and applies walkFn to each entry
	Walk(ctx kitectx.Context, path string, walkFn WalkFunc) error
}

// LibraryManager is an interface for interacting with library directories
type LibraryManager interface {
	// AddProject adds the project paths for virtualenv discovery
	AddProject(paths []string)
	// Dirs lists all library directories found by the manager
	Dirs() []string
	// Increments the count for the library type used
	MarkUsed(path string)
	// Stats contains stats about library files used
	Stats() map[string]int
}

// FileInfo encapsulates the file info for a file.
type FileInfo struct {
	IsDir bool
	Size  int64
}

// ParseInfo encapsulates parse info for the files in a build.
type ParseInfo struct {
	ParseDurations []time.Duration `json:"parse_durations"`
	ParseTimeouts  int64           `json:"parse_timeouts"`
	ParseFailures  int64           `json:"parse_failures"`
	ParseErrors    []string        `json:"parse_errors"`
}

// BuilderParams contains all the input for the BuilderFunc
type BuilderParams struct {
	UserID         int64
	MachineID      string
	Filename       string
	Files          []*localfiles.File
	LibraryFiles   []string
	LibraryManager LibraryManager
	FileGetter     FileGetter
	Putter         Putter
	FileSystem     FileSystem
	Local          bool
	BuildDurations map[string]time.Duration
	BuildInfo      map[string]int
	ParseInfo      *ParseInfo
}

// BuilderResult is returned by a BuilderFunc to keep track of what has been built
type BuilderResult struct {
	Root          string
	Files         []*localfiles.File
	LibraryFiles  []*localfiles.File
	LibraryDirs   []string
	MissingHashes map[string]bool
	LocalArtifact Cleanuper
}

// BuilderFunc is a function signature used to build artifacts
// the caller is responsible for catch-all panic handling
type BuilderFunc func(ctx kitectx.Context, params BuilderParams) (*BuilderResult, error)

// --

// LocalFileSystem implements FileSystem & FileGetter for local files; it does not do any cross-platform normalization
type LocalFileSystem struct {
	Include func(path string, info FileInfo) bool
}

func (fs LocalFileSystem) include(path string, info FileInfo) bool {
	if fs.Include == nil {
		return true
	}
	return fs.Include(path, info)
}

// Stat implements FileSystem
func (fs LocalFileSystem) Stat(path string) (FileInfo, error) {
	if !filepath.IsAbs(path) {
		return FileInfo{}, ErrNonAbsolutePath
	}
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}

	if !fs.include(path, FileInfo{IsDir: info.IsDir()}) {
		return FileInfo{}, errors.New("path excluded from filesystem")
	}

	return FileInfo{
		IsDir: info.IsDir(),
		Size:  info.Size(),
	}, nil
}

// Glob implements FileSystem
func (fs LocalFileSystem) Glob(dir, pattern string) ([]string, error) {
	if !filepath.IsAbs(dir) {
		return nil, ErrNonAbsolutePath
	}
	return filepath.Glob(filepath.Join(dir, pattern))
}

// Walk implements FileSystem
func (fs LocalFileSystem) Walk(ctx kitectx.Context, root string, walkFn WalkFunc) error {
	fi, err := os.Stat(root)
	if err != nil {
		err = walkFn(root, FileInfo{IsDir: false}, err)
	} else {
		err = fs.walk(ctx, root, FileInfo{IsDir: fi.IsDir()}, walkFn)
	}
	if err == ErrSkipDir {
		return nil
	}
	return err
}

type entry struct {
	path string
	info FileInfo
}

func (fs LocalFileSystem) walk(ctx kitectx.Context, root string, info FileInfo, walkFn WalkFunc) error {
	if !info.IsDir {
		return walkFn(root, info, nil)
	}
	fis, err := ioutil.ReadDir(root)
	err1 := walkFn(root, info, err)
	if err != nil || err1 != nil {
		return err1
	}
	var files, dirs []entry
	for _, fi := range fis {
		filePath := filepath.Join(root, fi.Name())
		if fi.IsDir() {
			dirs = append(dirs, entry{filePath, FileInfo{IsDir: fi.IsDir()}})
			continue
		}
		files = append(files, entry{filePath, FileInfo{IsDir: fi.IsDir()}})
	}
	for _, e := range append(files, dirs...) {
		if !fs.include(e.path, e.info) {
			if e.info.IsDir {
				return ErrSkipDir
			}
		}
		err = fs.walk(ctx, e.path, e.info, walkFn)
		if err != nil {
			if !e.info.IsDir || err != ErrSkipDir {
				return err
			}
		}
	}
	return nil
}

// Get implements FileGetter
func (fs LocalFileSystem) Get(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fs.include(path, FileInfo{IsDir: info.IsDir()}) {
		return nil, errors.New("file excluded from file system")
	}

	return ioutil.ReadAll(f)
}

// cachedFileGetter implements FileGetter. It uses localfiles.ContentStore to retrieve requested data,
// and is backed by an on-disk cache.
type cachedFileGetter struct {
	filecache *diskcache.Cache
	getter    FileGetter
}

// NewCachedFileGetter returns a FileGetter that uses a DiskCache for the underlying getter
func NewCachedFileGetter(cache *diskcache.Cache, getter FileGetter) FileGetter {
	return &cachedFileGetter{
		filecache: cache,
		getter:    getter,
	}
}

// Get implements FileGetter
func (l *cachedFileGetter) Get(key string) ([]byte, error) {
	defer fileGetDuration.DeferRecord(time.Now())

	if content, err := l.filecache.Get([]byte(key)); err == nil {
		fileGetCacheRatio.Hit()
		fileGetFoundRatio.Hit()
		return content, nil
	}

	fileGetCacheRatio.Miss()

	if content, err := l.getter.Get(key); err == nil {
		err = l.filecache.Put([]byte(key), content)
		if err != nil {
			log.Println(err)
		}
		fileGetFoundRatio.Hit()
		return content, nil
	}

	fileGetFoundRatio.Miss()

	return nil, fmt.Errorf("could not find content for %s", string(key))
}

// --

// localPutter implements Putter. It uses a disk-backed storage cache to store
// artifacts based on a key of artifact UUID and name. This key is consistent with
// the key used by artifactServer to retrieve data when requested.
type localPutter struct {
	id    string
	cache *diskcache.Cache
	names []string
}

func newLocalPutter(cache *diskcache.Cache, id string) *localPutter {
	return &localPutter{id: id, cache: cache}
}

// Put implements Putter
func (t *localPutter) Put(name string, data []byte) error {
	defer putDuration.DeferRecord(time.Now())
	defer artifactDuration(name).RecordDuration(time.Since(time.Now()))

	artifactByte(name).Record(int64(len(data)))

	key := artifactKey(t.id, name)
	err := t.cache.Put(key, data)
	if err == nil {
		t.names = append(t.names, name)
	}
	return err
}

// PutWriter implements Putter
func (t *localPutter) PutWriter(name string) (io.WriteCloser, error) {
	key := artifactKey(t.id, name)
	w, err := t.cache.PutWriter(key)
	if err == nil {
		t.names = append(t.names, name)
	}
	return &timedWriter{
		name:  name,
		w:     w,
		start: time.Now(),
	}, nil
}

type timedWriter struct {
	name  string
	w     io.WriteCloser
	start time.Time
	bytes int
}

func (t *timedWriter) Write(buf []byte) (int, error) {
	t.bytes += len(buf)
	return t.w.Write(buf)
}

func (t *timedWriter) Close() error {
	putWriterDuration.RecordDuration(time.Since(t.start))
	artifactDuration(t.name).RecordDuration(time.Since(t.start))
	artifactByte(t.name).Record(int64(t.bytes))

	return t.w.Close()
}

func (t *localPutter) files() []string {
	return t.names
}
