package indexing

import (
	"context"

	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/filesystem"
	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Context manages local code indexing. Implements localcode.Context
type Context struct {
	indexer        *manager
	DriverProvider driver.Provider
}

// NewContext returns a new context
func NewContext(ctx context.Context, python *python.Services, fs *filesystem.Manager, userIDs userids.IDs) *Context {
	l := &Context{
		indexer: newManager(ctx, fs, userIDs),
	}

	builderLoader := &pythonbatch.BuilderLoader{
		Graph:   python.ResourceManager,
		Options: pythonbatch.DefaultLocalOptions,
	}

	l.indexer.registerBuilder(lang.Python, builderLoader.Build)

	return l
}

// LatestFileDriver returns the latest available file driver of the file
func (l *Context) LatestFileDriver(unixFilepath string) (core.FileDriver, error) {
	fd := l.DriverProvider.LatestDriver(kitectx.Background(), unixFilepath)
	if fd == nil {
		return nil, errors.Errorf("file driver unavailable")
	}
	return fd, nil
}

// Terminate releases resources used by this cotnext
func (l *Context) Terminate() {
	l.indexer.Terminate()
	l.Cleanup()
}

// Reset clears all build artifacts
func (l *Context) Reset() {
	l.indexer.reset()
}

// ArtifactForFile returns a loaded artifact that can serve the provided file or directory path.
// This object should not be stored or persisted anywhere.
// Always request a new Artifact as updates may have been made by the localcode framework.
func (l *Context) ArtifactForFile(path string) (interface{}, error) {
	artifact, err := l.indexer.artifactThatContains(path, false, nil)
	if err != nil {
		return nil, err
	}
	return artifact.object, nil
}

// RequestForFile returns a loaded artifact that can serve the provided filename. If one is not found
// it will request one from the worker. This object should not be stored or persisted anywhere. Always
// request a new Artifact as updates may have been made by the localcode framework.
// trackExtra may be logged as tracking data in case of any error.
func (l *Context) RequestForFile(filename string, trackExtra interface{}) (interface{}, error) {
	artifact, err := l.indexer.artifactThatContains(filename, true, trackExtra)
	if err != nil {
		return nil, err
	}
	return artifact.object, nil
}

// AnyArtifact returns the most general artifact that can be found. This object
// should not be stored or persisted anywhere. Always request a new Artifact as updates may have
// been made by the localcode framework.
func (l *Context) AnyArtifact() (interface{}, error) {
	artifact, err := l.indexer.anyArtifact()
	if err != nil {
		return nil, err
	}
	return artifact.object, nil
}

// Status returns a list of Status objects for the provided user/machine
func (l *Context) Status(n int) []*localcode.Status {
	var indices []*localcode.Status
	for _, loaded := range l.indexer.allArtifacts() {
		if len(indices) >= n {
			break
		}
		indices = append(indices, &localcode.Status{
			Path:       loaded.rootPath,
			Files:      len(loaded.fileHashes),
			FileHashes: loaded.fileHashes,
		})
	}

	return indices
}

// StatusResponse constructs the response for requests for local code index status
func (l *Context) StatusResponse(n int) *localcode.StatusResponse {
	statuses := l.Status(n)

	var resp localcode.StatusResponse
	for _, index := range statuses {
		resp.Indices = append(resp.Indices, localcode.IndexResponse{
			Path:       index.Path,
			Files:      index.Files,
			FileHashes: index.FileHashes,
		})
	}

	return &resp
}

// TestFlush waits until all jobs in the pool have been processed. This must only be called in test cases.
func (l *Context) TestFlush(ctx context.Context) {
	_ = l.indexer.pool.Wait()
}

// Cleanup cleans up any state UserContext may be holding on to via artifacts
func (l *Context) Cleanup() error {
	return nil
}
