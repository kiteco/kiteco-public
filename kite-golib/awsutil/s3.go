package awsutil

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

var (
	localRegion = envutil.GetenvDefault("AWS_REGION", "")
	// Path to the S3 cache.
	cacheroot = envutil.GetenvDefault("KITE_S3CACHE", "/var/kite/s3cache")
	// if KITE_USE_AZURE_MIRROR is set to '1', NewS3Reader and NewCachedS3Reader will attempt to find the
	// corresponding file in the Azure mirror. If not found, the file will be copied from S3
	// to the Azure mirror, at which point it will be read from Azure again.
	useAzureMirror = envutil.GetenvDefault("KITE_USE_AZURE_MIRROR", "0") == "1"
)

// IsS3URI returns true if the path is an s3 uri.
func IsS3URI(path string) bool {
	return strings.HasPrefix(path, "s3://")
}

// SetCacheRoot allows for direct configuration of the cacheroot
func SetCacheRoot(path string) {
	cacheroot = path
}

// NewS3 creates an s3 client.
func NewS3(region string) (*s3.S3, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return s3.New(sess, aws.NewConfig().WithRegion(region)), nil
}

// NewS3Reader returns a io.ReadCloser that will read the contents
// of the file pointed to by the uri. URI will be of the form
// s3://bucket-name/path/to/file
func NewS3Reader(uri string) (io.ReadCloser, error) {
	s3uri, err := ValidateURI(uri)
	if err != nil {
		return nil, err
	}

	head, err := headS3URL(s3uri)
	if err != nil {
		return nil, err
	}

	return newS3ReaderWithHead(uri, head)
}

// --

func newS3ReaderWithHead(uri string, head *s3.HeadObjectOutput) (io.ReadCloser, error) {
	s3url, err := ValidateURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid s3 uri: %s: %v", uri, err)
	}

	if useAzureMirror {
		log.Printf("requesting %s from Azure mirror", uri)
		if head == nil {
			return nil, fmt.Errorf("cannot get from Azure mirror: cannot find S3 header for %s", uri)
		}
		checksum := checksumFromHead(head)

		// If the file is gzipped in S3, we want to set the appropriate header in Azure as well
		var isGZipped bool
		if head.ContentEncoding != nil && *head.ContentEncoding == "gzip" {
			isGZipped = true
		}

		r, err := mirroredAzureReader(s3url, checksum)
		if err != nil {
			log.Printf("getting %s from Azure failed: %v", uri, err)
			log.Printf("attempting to copy %s from S3 to Azure", uri)

			if err := copyToAzure(s3url, isGZipped); err != nil {
				return nil, fmt.Errorf("error copying %s to Azure: %v", uri, err)
			}
			return mirroredAzureReader(s3url, checksum)
		}
		return r, nil
	}

	s3url = regionURI(s3url)
	return objectReader(s3url, false)
}

func objectReader(uri *url.URL, acceptGZip bool) (io.ReadCloser, error) {
	region, err := objectRegion(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to determine region: %s", err)
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	// Re-create client for bucket's region, and get the object
	s3client := s3.New(sess, aws.NewConfig().WithRegion(region))

	key := strings.TrimPrefix(uri.Path, "/")
	getObjInput := &s3.GetObjectInput{
		Bucket: &uri.Host,
		Key:    &key,
	}

	req, getObjOutput := s3client.GetObjectRequest(getObjInput)
	if acceptGZip {
		// If this header is set, the S3 library returns the raw gzipped object (if it's gzipped) as opposed
		// to automatically decompressing it in the reader
		req.HTTPRequest.Header.Add("Accept-Encoding", "gzip")
	}

	err = req.Send()
	return getObjOutput.Body, err
}

func objectRegion(uri *url.URL) (string, error) {
	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}

	s3client := s3.New(sess, aws.NewConfig().WithRegion("us-west-1"))

	// Discover the region that this bucket is located in
	bucketLocInput := &s3.GetBucketLocationInput{
		Bucket: &uri.Host,
	}
	bucketLocOutput, err := s3client.GetBucketLocation(bucketLocInput)
	if err != nil {
		return "", err
	}

	var region string
	if bucketLocOutput.LocationConstraint == nil {
		region = "us-east-1"
	} else {
		region = *bucketLocOutput.LocationConstraint
	}

	return region, nil
}

// CachePath returns the location of where the S3 file will be saved on disk.
func CachePath(s3url *url.URL) string {
	return CachePathAt(cacheroot, s3url)
}

// CachePathAt returns the location where the s3 file will be saved on disk,
// rooted at the specified cacheroot.
func CachePathAt(cacheroot string, s3url *url.URL) string {
	return filepath.Join(cacheroot, s3url.Host, s3url.Path)
}

// CachedReaderOptions contains options for a cached s3 reader
type CachedReaderOptions struct {
	CacheRoot string
	Logger    io.Writer
}

// NewCachedS3Reader returns an io.ReadCloser that will read the
// contents of the file pointed to by the uri. If the file exists in
// the local cache then its MD5 hash will be computed and compared to
// that of the remote S3 object. If they match then the file will be
// read from the cache; if not then the file will be read from S3. Note
// that in the latter case, the object will _not_ be downloaded in its
// entirety before it can be read locally; rather, it will be copied
// to the local cache as it is received.
func NewCachedS3Reader(uri string) (io.ReadCloser, error) {
	return NewCachedS3ReaderWithOptions(CachedReaderOptions{
		CacheRoot: cacheroot,
		Logger:    os.Stderr,
	}, uri)
}

func logf(w io.Writer, fmtstr string, args ...interface{}) {
	if w == nil {
		return
	}
	w.Write([]byte(fmt.Sprintf(fmtstr, args...)))
}

// NewCachedS3ReaderWithOptions returns a cached s3 reader using the specified options.
func NewCachedS3ReaderWithOptions(opts CachedReaderOptions, uri string) (io.ReadCloser, error) {
	s3url, err := ValidateURI(uri)
	if err != nil {
		return nil, err
	}

	if opts.CacheRoot == "" {
		opts.CacheRoot = cacheroot
	}

	cachepath := CachePathAt(opts.CacheRoot, s3url)

	// Get the header/etag for the remote object
	var etag []byte
	head, err := headS3URL(s3url)
	if err != nil {
		head = nil
		logf(opts.Logger, "failed to compute remote checksum: %v, will try local cache\n", err)
	} else {
		etag = checksumFromHead(head)
	}

	// Attempt to load it from the cache
	r, err := tryCache(etag, cachepath)
	if err == nil {
		logf(opts.Logger, "cache hit on %s\n", uri)
		logf(opts.Logger, "loading from %s\n", cachepath)
		return r, nil
	}

	// Fall back to loading from S3 while copying into the local cache
	logf(opts.Logger, "cache miss on %s: %v\n", uri, err)
	r, err = newS3ReaderWithHead(uri, head)
	if err != nil {
		return nil, err
	}

	cacherootTmp := filepath.Join(opts.CacheRoot, "tmp")
	err = os.MkdirAll(cacherootTmp, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return newLateCopyReader(r, cachepath, cacherootTmp, etag)
}

type bufferedS3Writer struct {
	f     *os.File
	s3uri *url.URL
}

// Write writes to disk
func (w bufferedS3Writer) Write(p []byte) (int, error) {
	return w.f.Write(p)
}

// Close flushes to disk, copies the written data to s3, and closes the file
func (w bufferedS3Writer) Close() error {
	defer os.Remove(w.f.Name()) // delete the buffer file from disk
	defer w.f.Close()           // after closing the buffer file handle

	w.f.Sync()               // flush to disk
	_, err := w.f.Seek(0, 0) // seek to beginning to allow s3 library to read
	if err != nil {
		return err
	}

	// upload to S3
	region, err := objectRegion(w.s3uri)
	if err != nil {
		return fmt.Errorf("unable to determine region: %s", err)
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	// Re-create client for bucket's region, and get the object
	s3client := s3.New(sess, aws.NewConfig().WithRegion(region))

	key := strings.TrimPrefix(w.s3uri.Path, "/")
	input := &s3.PutObjectInput{
		Bucket: aws.String(w.s3uri.Host),
		Key:    aws.String(key),
		Body:   w.f,
	}
	_, err = s3client.PutObject(input)
	if err != nil {
		return err
	}

	return nil
}

func (w bufferedS3Writer) Name() string {
	return w.s3uri.String()
}

// NamedWriteCloser is a file-like object extending io.WriteCloser with a string Name() similar to os.File.Name()
type NamedWriteCloser interface {
	io.WriteCloser
	Name() string
}

// NewBufferedS3Writer returns an io.WriteCloser that will write
// to disk and upload to S3 on Close
func NewBufferedS3Writer(uri string) (NamedWriteCloser, error) {
	s3url, err := ValidateURI(uri)
	if err != nil {
		return nil, err
	}

	f, err := ioutil.TempFile("", "s3buffer")
	if err != nil {
		return nil, err
	}
	return bufferedS3Writer{f: f, s3uri: s3url}, nil
}

// NewBufferedS3WriterWithTmp returns an io.WriteCloser that will write
// to disk and upload to S3 on Close using the specified tmp dir to store
// the intermediate files.
func NewBufferedS3WriterWithTmp(tmpDir, uri string) (NamedWriteCloser, error) {
	s3url, err := ValidateURI(uri)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return nil, err
	}

	f, err := ioutil.TempFile(tmpDir, "s3buffer")
	if err != nil {
		return nil, err
	}
	return bufferedS3Writer{f: f, s3uri: s3url}, nil
}

// S3PutObject writes the contents of the specified reader
// to the specified s3 URI.
func S3PutObject(r io.ReadSeeker, uri string) error {
	s3URL, err := ValidateURI(uri)
	if err != nil {
		return err
	}

	region, err := objectRegion(s3URL)
	if err != nil {
		return fmt.Errorf("unable to determine region: %s", err)
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	s3client := s3.New(sess, aws.NewConfig().WithRegion(region))

	key := strings.TrimPrefix(s3URL.Path, "/")
	_, err = s3client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s3URL.Host),
		Key:    aws.String(key),
		Body:   r,
	})

	return err
}

// S3ListObjects lists the objects in an s3 bucket with a given prefix.
// NOTE: we ignore objects with size 0 since they typically correspond
// to directories and are thus not fetchable.
func S3ListObjects(region, bucket, prefix string) ([]string, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	client := s3.New(sess, aws.NewConfig().WithRegion(region))

	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	var keys []string
	err = client.ListObjectsPages(params, func(p *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range p.Contents {
			if *obj.Size == 0 {
				// skip size zero objects, these correspond to directories
				continue
			}
			keys = append(keys, *obj.Key)
		}
		return true
	})

	if err != nil {
		return nil, fmt.Errorf("error list objects in `%s` (%s): %v", bucket, region, err)
	}
	return keys, nil
}

// --

// ValidateURI checks whether the given uri points to S3.
func ValidateURI(uri string) (*url.URL, error) {
	s3url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if s3url.Scheme != "s3" {
		return nil, fmt.Errorf(s3url.String(), "url is not a s3 path")
	}
	return s3url, nil
}

// Exists returns whether an object exists at the provided URI
func Exists(uri string) (bool, error) {
	s3url, err := ValidateURI(uri)
	if err != nil {
		return false, err
	}
	_, err = headS3URL(s3url)
	return err == nil, nil
}

// --

func regionURI(uri *url.URL) *url.URL {
	// empty region and us-west-1 should use the original URI because the default
	// buckets are in us-west-1
	if localRegion == "" || localRegion == "us-west-1" {
		return uri
	}

	regionBucket := fmt.Sprintf("kite-prod-data-%s", localRegion)
	regionURI, err := url.Parse(fmt.Sprintf("s3://%s/%s%s", regionBucket, uri.Host, uri.Path))
	if err != nil {
		log.Println("error parsing region-specific location for", uri.String(), err)
		return uri
	}

	origChecksum, err := checksumS3URL(uri)
	if err != nil {
		log.Println("error checksumming original uri", uri.String(), err)
		return uri
	}

	regionChecksum, err := checksumS3URL(regionURI)
	if err != nil {
		log.Println("error checksumming region uri", regionURI.String(), err)
		return uri
	}

	// If the checksums match, return the region-local URI
	if bytes.Equal(origChecksum, regionChecksum) {
		log.Println("selecting region-local uri:", regionURI.String())
		return regionURI
	}

	log.Printf("checksum mismatch in %s, selecting original uri: %s", localRegion, uri.String())
	return uri
}
