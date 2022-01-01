package fileutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

type fileMapOptions struct {
	fileMap   FileMap
	localOnly bool
}

var (
	optsGuarded fileMapOptions
	optsRW      sync.RWMutex
)

// SetLocalFileMap initializes the local filemap to use for datasets
func SetLocalFileMap(fm FileMap) {
	optsRW.Lock()
	defer optsRW.Unlock()
	optsGuarded.fileMap = fm
}

// SetLocalOnly disallows fileutil operations from accessing either S3 or the local S3 cache.
// Once set, fileutil operations may still access S3 data from the configured FileMap if available,
// and explicitly requested local files will open the files on disk.
func SetLocalOnly() {
	optsRW.Lock()
	defer optsRW.Unlock()
	optsGuarded.localOnly = true
}

func newReader(path string, s3ReaderMaker func(uri string) (io.ReadCloser, error)) (io.ReadCloser, error) {
	opts := func() fileMapOptions {
		optsRW.RLock()
		defer optsRW.RUnlock()
		return optsGuarded
	}()

	if strings.HasPrefix(path, "s3://") && opts.fileMap != nil {
		if r, err := NewFileMapReader(strings.TrimPrefix(path, "s3://"), opts.fileMap); err == nil {
			return r, nil
		}
	}

	if strings.HasPrefix(path, "s3://") {
		if opts.localOnly {
			return nil, errors.New("fileutil cannot load from S3 in local-only mode")
		}
		return s3ReaderMaker(path)
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, fmt.Errorf("error getting %s: %s", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			defer resp.Body.Close()
			io.Copy(ioutil.Discard, resp.Body)
			return nil, errors.Wrapf(err, "status code %d", resp.StatusCode)
		}
		return resp.Body, nil
	}

	return os.Open(path)
}

// NewReader opens a local or remote path for reading. If the path looks like
// "s3://bucket/path/to/object" then this will read an object from S3. Otherwise, this
// will read a path from the local filesystem.
func NewReader(path string) (io.ReadCloser, error) {
	return newReader(path, awsutil.NewS3Reader)
}

// DownloadedFile returns the path of a file downloaded to disk. If the input path is local, this will return that file path
// (after checking the path exists). Otherwise, if the path looks like an S3 path it will attempt to download that
// object and return the local path on disk. Repeated calls will return the same local path.
func DownloadedFile(path string) (string, error) {
	reader, readErr := NewCachedReader(path)
	if readErr != nil {
		return "", readErr
	}

	_, copyErr := io.Copy(ioutil.Discard, reader)
	if copyErr != nil {
		return "", copyErr
	}

	closeErr := reader.Close()
	if closeErr != nil {
		return "", closeErr
	}

	s3url, parseErr := awsutil.ValidateURI(path)
	if parseErr != nil {
		return path, nil
	}
	return awsutil.CachePath(s3url), nil
}

// NewCachedReader opens a local or remote path for reading. If the path looks like
// "s3://bucket/path/to/object" then this will read an object from S3. Otherwise, this
// will read a path from the local filesystem. Caching only applies to S3 paths.
func NewCachedReader(path string) (io.ReadCloser, error) {
	return newReader(path, awsutil.NewCachedS3Reader)
}

// NamedWriteCloser is a file-like object extending io.WriteCloser with a string Name() similar to os.File.Name()
type NamedWriteCloser interface {
	io.WriteCloser
	Name() string
}

// NewBufferedWriter opens a local or remote path for writing. If the path starts with
// "s3://", then this will write to a local buffer, copying to s3 on close. Otherwise,
// this will write to the local FS.
func NewBufferedWriter(path string) (NamedWriteCloser, error) {
	if awsutil.IsS3URI(path) {
		return awsutil.NewBufferedS3Writer(path)
	}
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(path)
}

// NewBufferedWriterWithCache is the same as NewBufferedWriter but uses the
// specified cache dir if the output file needs to be copied to s3
func NewBufferedWriterWithCache(path, cache string) (NamedWriteCloser, error) {
	if awsutil.IsS3URI(path) {
		if cache == "" {
			return awsutil.NewBufferedS3Writer(path)
		}
		return awsutil.NewBufferedS3WriterWithTmp(cache, path)
	}
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(path)
}

// ReadFile reads the contents of a local or remote path.
func ReadFile(path string) ([]byte, error) {
	r, err := NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ListDir returns the fully qualified names for the members
// of the provided directory. If the directory is local these
// will simply be the paths, if the directory is on s3 then
// these will be the keys to the entries. The results of
// this function are intended to be used in conjunction
// with NewCachedReader.
func ListDir(path string) ([]string, error) {
	if awsutil.IsS3URI(path) {
		trimmed := strings.TrimPrefix(path, "s3://")

		parts := strings.Split(trimmed, "/")
		bucket := parts[0]
		prefix := strings.Join(parts[1:], "/")

		keys, err := awsutil.S3ListObjects("us-west-1", bucket, prefix)
		if err != nil {
			return nil, fmt.Errorf("error reading from s3 path %s: %v", path, err)
		}

		var paths []string
		for _, key := range keys {
			path := Join("s3://", bucket, key)
			paths = append(paths, path)
		}
		return paths, nil
	}

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("error reading dir %s: %v", path, err)
	}

	var paths []string
	for _, entry := range entries {
		paths = append(paths, Join(path, entry.Name()))
	}

	return paths, nil
}
