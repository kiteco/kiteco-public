package azureutil

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	// maximum number of times to retry reading from an Azure blob
	maxRetries = 5
	// time to wait between retries
	retryInterval = 500 * time.Millisecond
)

// NewBlobReader that reads from the specified blob
func NewBlobReader(container, path string) (io.ReadCloser, error) {
	bs, err := getBlobService()
	if err != nil {
		return nil, err
	}

	ctn := bs.GetContainerReference(container)
	blob := ctn.GetBlobReference(path)

	r, err := blob.Get(nil)
	if err != nil {
		return nil, err
	}

	return &retriableReader{
		container: container,
		path:      path,
		blob:      blob,
		r:         r,
	}, nil
}

type retriableReader struct {
	container string
	path      string
	blob      *storage.Blob
	r         io.ReadCloser

	offset     uint64
	numRetries int
}

// Read implements io.ReadCloser
func (r *retriableReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if err == nil || err == io.EOF {
		r.offset += uint64(n)
		return n, err
	}

	// we repeatedly try to re-open the reader at the current offset until we succeed
	r.Close()
	for r.numRetries < maxRetries {
		time.Sleep(retryInterval)
		r.numRetries++
		log.Printf("retrying read from Azure blob (%s, %s) at offset %d, err: %v",
			r.container, r.path, r.offset, err)
		r.r, err = r.blob.GetRange(&storage.GetBlobRangeOptions{
			Range: &storage.BlobRange{
				Start: r.offset,
			},
		})
		if err == nil {
			// if we succeed, we go back to the normal flow of the reader
			return r.Read(p)
		}
		log.Printf("error re-opening reader for Azure blob (%s, %s) at offset %d, retrying: %v",
			r.container, r.path, r.offset, err)
	}

	return 0, fmt.Errorf(
		"error reading from Azure blob (%s, %s): max retries exceeded, last err: %v",
		r.container, r.path, err)
}

// Close implements io.ReadCloser
func (r *retriableReader) Close() error {
	if r.r != nil {
		return r.r.Close()
	}
	return nil
}
