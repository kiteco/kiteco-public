package awsutil

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// Wraps an io.ReadCloser and copies all data received to a temporary file. If
// the entire stream is consumed without error then the output file is
// moved to a specified destination path. If any errors are encountered or the file
// is closed before reading EOF then the temporary file is instead destroyed.
//
// This differs from io.TeeReader because (1) lateCopyReader implements io.ReadCloser
// not io.Reader; (2) lateCopyReader tracks errors encountered during Read() and
// Close() and changes whether it eventually stores the file on that basis.
type lateCopyReader struct {
	path     string
	temp     *os.File
	tee      io.Reader
	hash     hash.Hash
	orig     io.Closer
	checksum []byte
}

func newLateCopyReader(r io.ReadCloser, copyto, tmpDir string, checksum []byte) (*lateCopyReader, error) {
	f, err := ioutil.TempFile(tmpDir, "")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary file: %v", err)
	}
	h := md5.New()
	cpr := &lateCopyReader{
		temp:     f,
		path:     copyto,
		tee:      io.TeeReader(io.TeeReader(r, h), f),
		hash:     h,
		orig:     r,
		checksum: checksum,
	}
	return cpr, nil
}

func (r *lateCopyReader) Read(p []byte) (int, error) {
	// Note that despite the documentation, it _is_ possible to get n > 0 and err == EOF
	n, err := r.tee.Read(p)
	if err != nil {
		if err == io.EOF {
			cacheErr := r.commitToCache()
			if cacheErr != nil {
				log.Println(cacheErr)
			}
		} else {
			r.cancel()
		}
	}
	return n, err
}

func (r *lateCopyReader) Close() error {
	r.cancel()
	return r.orig.Close()
}

func (r *lateCopyReader) commitToCache() error {
	// Only move the file to its final destination if (1) there have been no errors and (2) EOF has been received.
	if r.temp != nil {
		path := r.temp.Name()

		// Close the temporary file
		err := r.temp.Close()
		if err != nil {
			return fmt.Errorf("error closing temporary file: %v", err)
		}

		// Move the temporary file to its final destination
		dir := filepath.Dir(r.path)
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return fmt.Errorf("error creating dir within cache: %v", err)
		}
		err = os.Rename(path, r.path)
		if err != nil {
			return fmt.Errorf("error moving temp file into cache: %v", err)
		}

		// Store the checksum file
		if r.checksum != nil {
			err = storeChecksumFor(r.path, hexEncode(r.hash.Sum(nil)), r.checksum)
			if err != nil {
				return err
			}
		}

		// Make sure not to do this twice
		r.temp = nil
	}

	return nil
}

func (r *lateCopyReader) cancel() {
	if r.temp != nil {
		path := r.temp.Name()

		// Close the temporary file
		err := r.temp.Close()
		if err != nil {
			log.Println("error closing temporary file:", err)
		}

		// Delete the temporary file
		err = os.Remove(path)
		if err != nil {
			log.Printf("error deleting %s: %v\n", path, err)
		}

		// Make sure not to do this twice
		r.temp = nil
	}
}

// Given a url to an S3 object and the local cache path for that object, determine whether the
// local cache exists and is up to date, and, if so, open the file. Otherwise, return nil
func tryCache(checksum []byte, cachepath string) (io.ReadCloser, error) {
	// if checksum is nil then do not check the hash
	if checksum != nil {
		local, err := checksumLocal(cachepath)
		if err != nil {
			return nil, fmt.Errorf("failed to compute local checksum: %v", err)
		}

		if !bytes.Equal(local, checksum) {
			return nil, errors.New("file exists in cache but is out of date")
		}
	}

	f, err := os.Open(cachepath)
	if err != nil {
		return nil, err
	}
	return f, err
}
