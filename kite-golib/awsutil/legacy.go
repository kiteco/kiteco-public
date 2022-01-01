package awsutil

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

// ShardedFile represents a collection of part-* files in a directory
type ShardedFile struct {
	bucket *s3.Bucket
	keys   []string
}

// NewShardedFile returns a ShardedFile object.
func NewShardedFile(dir string) (*ShardedFile, error) {
	s3url, err := ValidateURI(dir)
	if err != nil {
		return nil, err
	}

	bucket, err := GetBucket(s3url.Host)
	if err != nil {
		return nil, err
	}

	key := strings.TrimPrefix(s3url.Path, "/")
	list, err := bucket.List(path.Join(key, "part-"), "/", "a", 100)
	if err != nil {
		return nil, err
	}

	sf := &ShardedFile{bucket: bucket}
	for _, key := range list.Contents {
		sf.keys = append(sf.keys, key.Key)
	}
	sort.Strings(sf.keys)

	return sf, nil
}

// Shards returns the number of shards in the file
func (s *ShardedFile) Shards() int {
	return len(s.keys)
}

// URL returns the full url of the n-th shard.
func (s *ShardedFile) URL(n int) string {
	return fmt.Sprintf("s3://%s/%s", s.bucket.Name, s.keys[n])
}

// Reader returns a io.ReadCloser for the n-th shard.
func (s *ShardedFile) Reader(index int) (io.ReadCloser, error) {
	return NewS3Reader(s.URL(index))
}

// CachedReader returns a io.ReadCloser for the n-th shard.
func (s *ShardedFile) CachedReader(index int) (io.ReadCloser, error) {
	return NewCachedS3Reader(s.URL(index))
}

// --

// GetBucket retrieves a bucket from S3.
func GetBucket(bucket string) (*s3.Bucket, error) {
	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}

	client := s3.New(auth, aws.USWest)

	return client.Bucket(bucket), nil
}
