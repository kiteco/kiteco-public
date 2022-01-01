package indexing

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"

	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/filesystem"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --
func Test_RequestingArtifact(t *testing.T) {
	fs := newTestFilesystem()
	m := newManager(context.Background(), fs, userids.NewUserIDs("", ""))
	m.debug = true

	content := []byte("temporary file's content")
	tmpdir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	var files []string
	for i := 0; i < 10; i++ {
		tmpfile := filepath.Join(tmpdir, fmt.Sprintf("%d.py", i))
		err = ioutil.WriteFile(tmpfile, content, 0666)
		assert.NoError(t, err)
		// canonicalize filename
		tmpfile, err = localpath.ToUnix(tmpfile)
		assert.NoError(t, err)
		tmpfile = strings.ToLower(tmpfile)
		files = append(files, tmpfile)
	}

	req := files[0]

	var buildCount int32
	builder := func(t *testing.T) localcode.BuilderFunc {
		return func(_ kitectx.Context, params localcode.BuilderParams) (*localcode.BuilderResult, error) {
			atomic.AddInt32(&buildCount, 1)

			require.True(t, params.Local)
			require.NotNil(t, params.FileGetter)

			var localfiles []string
			children, err := fs.GetChildren(kitectx.Background(), path.Dir(req))
			require.NoError(t, err)
			for _, file := range children {
				localfiles = append(localfiles, file.Name)
			}

			sort.Strings(localfiles)
			require.Equal(t, files, localfiles)
			require.Equal(t, req, params.Filename)

			return &localcode.BuilderResult{
				Root:  params.Filename,
				Files: children,
				LocalArtifact: &testArtifact{
					ts: time.Now(),
				},
			}, nil
		}
	}(t)

	m.registerBuilder(lang.Python, builder)

	// make sure there are no artifacts that are matching
	obj, err := m.anyArtifact()
	require.Error(t, err)
	require.Nil(t, obj)
	require.Zero(t, atomic.LoadInt32(&buildCount))

	// make sure artifactThatContains returns no artifact as well
	obj, err = m.artifactThatContains(req, false, nil)
	require.Error(t, err)
	require.Nil(t, obj)
	require.Zero(t, atomic.LoadInt32(&buildCount))

	// make a request
	obj, err = m.artifactThatContains(req, true, nil)
	require.Error(t, err)
	require.Nil(t, obj)

	// wait to allow builder to run
	waitForBuild(buildCount)

	// make sure builder ran
	require.EqualValues(t, 1, atomic.LoadInt32(&buildCount))

	// should return artifact now
	obj, err = m.artifactThatContains(req, true, nil)
	require.NoError(t, err)
	require.NotNil(t, obj)

	ta := obj.object.(*testArtifact)
	require.False(t, ta.ts.IsZero())

	dirs := make(map[string]bool)
	for _, file := range files {
		d := filepath.Dir(file)
		dirs[d] = true
	}

	// Note: filehashes also contains unique parent directories of the files
	require.Equal(t, len(files)+len(dirs), len(obj.fileHashes))
	require.Equal(t, req, obj.requestPath)

	// anyArtifact should return something now as well
	obj, err = m.anyArtifact()
	require.NoError(t, err)
	require.NotNil(t, obj)
}

func Test_Changes(t *testing.T) {
	fs := newTestFilesystem()
	m := newManager(context.Background(), fs, userids.NewUserIDs("", ""))
	m.debug = true

	content := []byte("temporary file's content")
	tmpdir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	var files []string
	for i := 0; i < 10; i++ {
		tmpfile := filepath.Join(tmpdir, fmt.Sprintf("%d.py", i))
		err = ioutil.WriteFile(tmpfile, content, 0666)
		assert.NoError(t, err)
		// canonicalize filename
		tmpfile, err = localpath.ToUnix(tmpfile)
		assert.NoError(t, err)
		tmpfile = strings.ToLower(tmpfile)
		files = append(files, tmpfile)
	}
	req := files[0]

	var buildCount int32
	builder := func(t *testing.T) localcode.BuilderFunc {
		return func(_ kitectx.Context, params localcode.BuilderParams) (*localcode.BuilderResult, error) {
			atomic.StoreInt32(&buildCount, 1)
			children, err := fs.GetChildren(kitectx.Background(), path.Dir(req))
			require.NoError(t, err)
			return &localcode.BuilderResult{
				Root:  params.Filename,
				Files: children,
				LocalArtifact: &testArtifact{
					ts: time.Now(),
				},
			}, nil
		}
	}(t)

	m.registerBuilder(lang.Python, builder)
	m.artifactThatContains(req, true, nil)

	// wait for initial artifact
	waitForBuild(buildCount)

	obj1, err := m.artifactThatContains(req, false, nil)
	require.NoError(t, err)
	require.NotNil(t, obj1)
	atomic.StoreInt32(&buildCount, 0)

	select {
	case fs.changes <- filesystem.Change{Paths: []string{req}}:
	default:
		require.Fail(t, "changes channel blocked")
	}

	// wait for change to trigger rebuild
	waitForBuild(buildCount)

	obj2, err := m.artifactThatContains(req, false, nil)
	require.NoError(t, err)
	require.NotNil(t, obj2)

	// make sure the new artifact is actually a rebuilt artifact
	require.True(t, obj2.object.(*testArtifact).ts.After(obj1.object.(*testArtifact).ts))

	// make a change that does not affect current artifacts
	select {
	case fs.changes <- filesystem.Change{Paths: []string{"/a/different/path.py"}}:
	default:
		require.Fail(t, "changes channel blocked")
	}

	obj3, err := m.artifactThatContains(req, false, nil)
	require.NoError(t, err)
	require.NotNil(t, obj3)

	// make sure that obj2 and obj3 are the same (not rebuilt)
	require.True(t, obj2.object.(*testArtifact).ts.Equal(obj3.object.(*testArtifact).ts))
}

func Test_MultipleOverlappingRequests(t *testing.T) {
	fs := newTestFilesystem()
	m := newManager(context.Background(), fs, userids.NewUserIDs("", ""))
	m.debug = true

	content := []byte("temporary file's content")
	tmpdir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	var files []string
	for i := 0; i < 10; i++ {
		tmpfile := filepath.Join(tmpdir, fmt.Sprintf("%d.py", i))
		err = ioutil.WriteFile(tmpfile, content, 0666)
		assert.NoError(t, err)
		// canonicalize filename
		tmpfile, err = localpath.ToUnix(tmpfile)
		assert.NoError(t, err)
		tmpfile = strings.ToLower(tmpfile)
		files = append(files, tmpfile)
	}
	req := files[0]

	var buildCount int32
	builder := func(t *testing.T) localcode.BuilderFunc {
		return func(_ kitectx.Context, params localcode.BuilderParams) (*localcode.BuilderResult, error) {
			atomic.AddInt32(&buildCount, 1)

			// Introduce a sleep to slow things down, and allowing duplicates to pile up
			time.Sleep(time.Second)

			children, err := fs.GetChildren(kitectx.Background(), path.Dir(req))
			require.NoError(t, err)
			return &localcode.BuilderResult{
				Root:  params.Filename,
				Files: children,
				LocalArtifact: &testArtifact{
					ts: time.Now(),
				},
			}, nil
		}
	}(t)

	m.registerBuilder(lang.Python, builder)

	// Generate multiple requests that should all map to one artifact
	for i := 0; i < 10; i++ {
		for _, file := range files {
			m.artifactThatContains(file, true, nil)
		}
	}

	waitForBuild(buildCount)

	// should only have built an artifact once
	require.EqualValues(t, 1, atomic.LoadInt32(&buildCount))
}

func Test_FileDoesNotExist(t *testing.T) {
	fs := newTestFilesystem()
	m := newManager(context.Background(), fs, userids.NewUserIDs("", ""))
	m.debug = true

	content := []byte("temporary file's content")
	tmpdir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	var files []string
	for i := 0; i < 10; i++ {
		tmpfile := filepath.Join(tmpdir, fmt.Sprintf("%d.py", i))
		err = ioutil.WriteFile(tmpfile, content, 0666)
		assert.NoError(t, err)
		// canonicalize filename
		tmpfile, err = localpath.ToUnix(tmpfile)
		assert.NoError(t, err)
		tmpfile = strings.ToLower(tmpfile)
		files = append(files, tmpfile)
	}
	req := files[0]

	var buildCount int32
	builder := func(t *testing.T) localcode.BuilderFunc {
		return func(_ kitectx.Context, params localcode.BuilderParams) (*localcode.BuilderResult, error) {
			atomic.AddInt32(&buildCount, 1)
			children, err := fs.GetChildren(kitectx.Background(), path.Dir(req))
			require.NoError(t, err)
			return &localcode.BuilderResult{
				Root:  params.Filename,
				Files: children,
				LocalArtifact: &testArtifact{
					ts: time.Now(),
				},
			}, nil
		}
	}(t)

	m.registerBuilder(lang.Python, builder)

	// create request for a file that does not exist
	file := filepath.Join(tmpdir, "notexist.py")
	m.requests.Store(file, requestBundle{
		path:       file,
		trackExtra: nil,
		ts:         time.Now(),
	})

	err = m.handleRequest(time.Now())
	require.NoError(t, err)

	// should not have built any artifacts
	require.Zero(t, atomic.LoadInt32(&buildCount))
	_, exists := m.requests.Load(file)
	assert.False(t, exists)

	// create a change for a file that does not exist
	m.changes.Store(file, changeBundle{
		path: file,
		ts:   time.Now(),
	})
	m.handleFileChange(time.Now())
	require.NoError(t, err)

	// should not have built any artifacts
	require.Zero(t, atomic.LoadInt32(&buildCount))
	_, exists = m.requests.Load(file)
	assert.False(t, exists)

	// check that saving a new file triggers an index build
	// first create request for edits of new file
	m.requests.Store(file, requestBundle{
		path:       file,
		trackExtra: nil,
		ts:         time.Now(),
	})

	err = m.handleRequest(time.Now())
	require.NoError(t, err)

	// save file
	err = ioutil.WriteFile(file, content, 0666)
	assert.NoError(t, err)
	m.changes.Store(file, changeBundle{
		path: file,
		ts:   time.Now(),
	})
	m.handleFileChange(time.Now())

	// wait for build
	waitForBuild(buildCount)

	// should have built an artifact
	require.EqualValues(t, 1, atomic.LoadInt32(&buildCount))
}

func Test_FilteredRequest(t *testing.T) {
	fs := newTestFilesystem()
	m := newManager(context.Background(), fs, userids.NewUserIDs("", ""))
	m.debug = true

	content := []byte("temporary file's content")
	tmpdir, err := ioutil.TempDir("", "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir) // clean up

	var files []string
	for i := 0; i < 10; i++ {
		tmpfile := filepath.Join(tmpdir, fmt.Sprintf("%d.py", i))
		err = ioutil.WriteFile(tmpfile, content, 0666)
		assert.NoError(t, err)
		// canonicalize filename
		tmpfile, err = localpath.ToUnix(tmpfile)
		assert.NoError(t, err)
		tmpfile = strings.ToLower(tmpfile)
		files = append(files, tmpfile)
	}
	req := files[0]

	var buildCount int32
	builder := func(t *testing.T) localcode.BuilderFunc {
		return func(_ kitectx.Context, params localcode.BuilderParams) (*localcode.BuilderResult, error) {
			atomic.AddInt32(&buildCount, 1)
			children, err := fs.GetChildren(kitectx.Background(), path.Dir(req))
			require.NoError(t, err)
			return &localcode.BuilderResult{
				Root:  params.Filename,
				Files: children,
				LocalArtifact: &testArtifact{
					ts: time.Now(),
				},
			}, nil
		}
	}(t)

	m.registerBuilder(lang.Python, builder)

	// create request for a file that does not exist
	var badDir string
	if runtime.GOOS == "windows" {
		badDir = filepath.Join(tmpdir, "appdata")
	} else {
		badDir = filepath.Join(tmpdir, "Library")
	}
	err = os.Mkdir(badDir, os.ModePerm)
	require.NoError(t, err)
	file := filepath.Join(badDir, "notexist.py")
	m.requests.Store(file, requestBundle{
		path:       file,
		trackExtra: nil,
		ts:         time.Now(),
	})

	err = m.handleRequest(time.Now())
	require.NoError(t, err)

	// should not have built any artifacts
	require.Zero(t, atomic.LoadInt32(&buildCount))
	_, exists := m.requests.Load(file)
	assert.False(t, exists)
}

// --

type testFilesystem struct {
	files   []string
	changes chan filesystem.Change
}

func newTestFilesystem() *testFilesystem {
	return &testFilesystem{
		changes: make(chan filesystem.Change, 10),
	}
}

func (t *testFilesystem) RootDir() string {
	return "/"
}

func (t *testFilesystem) KiteDir() string {
	return "/"
}

func (t *testFilesystem) Files() []string {
	return t.files
}

func (t *testFilesystem) Changes() <-chan filesystem.Change {
	return t.changes
}

func (t *testFilesystem) Watch(path string) error {
	return nil
}

func (t *testFilesystem) Unwatch(path string) error {
	return nil
}

func (t *testFilesystem) FileSystem() *filesystem.LocalFS {
	noopFilter := func(path string) bool {
		return true
	}
	falseFilter := func(path string) bool {
		return false
	}
	opts := filesystem.Options{
		IsFileAccepted: noopFilter,
		IsDirAccepted:  noopFilter,
		IsLibraryDir:   falseFilter,
	}
	return filesystem.NewLocalFS(context.Background(), opts, nil)
}

func (t *testFilesystem) GetChildren(ctx kitectx.Context, root string) ([]*localfiles.File, error) {
	var files []*localfiles.File
	err := t.FileSystem().Walk(ctx, root, func(path string, fi localcode.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir {
			files = append(files, &localfiles.File{
				Name:          path,
				HashedContent: path,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// --

type testArtifact struct {
	ts time.Time
}

func (t *testArtifact) Cleanup() error {
	return nil
}

// --

func waitForBuild(buildCount int32) {
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		if atomic.LoadInt32(&buildCount) == 1 {
			break
		}
	}
}
