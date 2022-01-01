package fileutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinEmpty(t *testing.T) {
	assert.Equal(t, Join(), "")
}

func TestJoinPath(t *testing.T) {
	assert.Equal(t, Join("foo", "bar"), "foo/bar", "relative file paths should join properly")
	assert.Equal(t, Join("/foo", "bar"), "/foo/bar", "absolute file paths should join properly")
}

func TestJoinUrl(t *testing.T) {
	assert.Equal(t, Join("http://kite.com", "foo", "bar"), "http://kite.com/foo/bar", "HTTP URL paths should join properly")
	assert.Equal(t, Join("s3://bucketname", "foo", "bar"), "s3://bucketname/foo/bar", "S3 URL paths should join properly")
	assert.Equal(t, Join("s3://", "bucket", "dir"), "s3://bucket/dir", "S3 without bucket name in URL should join properly")
	assert.NotEqual(t, Join("s3:/bucketname", "foo", "bar"), "s3://bucketname/foo/bar", "Bad schema (s3:/ rather than s3://) should not join properly")
}
