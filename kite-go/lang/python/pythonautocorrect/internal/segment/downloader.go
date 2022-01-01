package segment

import (
	"bufio"
	"fmt"
	"io"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

type downloader struct {
	scanner *bufio.Scanner
	rc      io.ReadCloser
}

func newDownloader(bucket, key string) (*downloader, error) {
	uri := fmt.Sprintf("s3://%s/%s", bucket, key)

	r, err := awsutil.NewCachedS3Reader(uri)
	if err != nil {
		return nil, fmt.Errorf("error getting object `%s`: %v", uri, err)
	}

	return &downloader{
		scanner: bufio.NewScanner(r),
		rc:      r,
	}, nil
}

func (d *downloader) Next() bool {
	if d.scanner.Scan() {
		return true
	}
	return false
}

// Err returns any non-EOF errors.
func (d *downloader) Err() error {
	return d.scanner.Err()
}

// Value returns raw segment json bytes that the downloader is currently on.
// NOTE: this is not a copy, so be sure to use the contents
// before calling `d.Next` again.
func (d *downloader) Value() []byte {
	return d.scanner.Bytes()
}

// Close the underlying file
func (d *downloader) Close() {
	d.rc.Close()
}
