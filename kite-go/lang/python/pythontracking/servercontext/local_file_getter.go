package servercontext

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"

	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/pkg/errors"
)

// LocalFileGetter implements localcode.FileGetter and is used to retrieve local files from S3 based on their filenames,
// caching them in a local directory.
type LocalFileGetter struct {
	bucket string
	hashes map[string]string
}

// NewLocalFileGetter ...
func NewLocalFileGetter(bucket string, files []*localfiles.File) *LocalFileGetter {
	hashes := make(map[string]string, len(files))
	for _, f := range files {
		hashes[f.Name] = f.HashedContent
	}

	return &LocalFileGetter{
		bucket: bucket,
		hashes: hashes,
	}
}

// GetHash gets the specified hash
func (l *LocalFileGetter) GetHash(key string) ([]byte, error) {
	uri := fmt.Sprintf("s3://%s/%s", l.bucket, key)
	reader, err := awsutil.NewCachedS3ReaderWithOptions(awsutil.CachedReaderOptions{
		CacheRoot: envutil.GetenvDefault("LOCALFILES_CACHE", "/var/kite/s3cache"),
	}, uri)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create cached S3 reader")
	}
	defer reader.Close()

	unzip, err := gzip.NewReader(reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decompress file")
	}

	buf, err := ioutil.ReadAll(unzip)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}
	return buf, nil
}

// Get implements localcode.FileGetter
func (l *LocalFileGetter) Get(filename string) ([]byte, error) {
	hash, found := l.hashes[filename]
	if !found {
		return nil, fmt.Errorf("no hash available for filename: %s", filename)
	}

	return l.GetHash(hash)
}
