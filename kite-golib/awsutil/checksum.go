package awsutil

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const checksumExtension = ".s3cache-checksum"

func hexEncode(buf []byte) []byte {
	dst := make([]byte, hex.EncodedLen(len(buf)))
	hex.Encode(dst, buf)
	return dst
}

// Compute a checksum of the contents of a local file. Note that the type of hash is not configurable
// because the corresponding ChecksumS3 only supports the MD5 hash.
func checksumLocal(path string) ([]byte, error) {
	h := md5.New()
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = io.Copy(h, f)
	if err != nil {
		return nil, fmt.Errorf("error copying contents of %s into hash: %v", path, err)
	}
	checksum := hexEncode(h.Sum(nil))

	// Test for a checksum file
	checksumPath := path + checksumExtension
	buf, err := ioutil.ReadFile(checksumPath)
	if os.IsNotExist(err) {
		return checksum, nil
	} else if err != nil {
		return nil, err
	}
	lines := bytes.Split(buf, []byte("\n"))
	if len(lines) != 2 {
		return nil, fmt.Errorf("expected %s to contain two lines but found %d lines", checksumPath, len(lines))
	}
	local, etag := lines[0], lines[1]
	if !bytes.Equal(local, checksum) {
		return nil, fmt.Errorf("first line of %s was %s but file checksum was %s", checksumPath, string(checksum), string(local))
	}
	return etag, nil
}

// Store a checksum with a corresponding etag
func storeChecksumFor(path string, local []byte, etag []byte) error {
	checksumPath := path + checksumExtension
	contents := bytes.Join([][]byte{local, etag}, []byte("\n"))
	err := ioutil.WriteFile(checksumPath, contents, 0777)
	if err != nil {
		return fmt.Errorf("error writing checksum to %s: %v", checksumPath, err)
	}
	return nil
}

// ChecksumS3 fetches the S3 checksum of the named file
func ChecksumS3(path string) ([]byte, error) {
	s3url, err := ValidateURI(path)
	if err != nil {
		return nil, err
	}
	return checksumS3URL(s3url)
}

// SavedChecksum returns, for a file, the
func SavedChecksum(localPath string) ([]byte, error) {
	checksumPath := localPath + checksumExtension
	buf, err := ioutil.ReadFile(checksumPath)
	if err != nil {
		return nil, err
	}
	lines := bytes.Split(buf, []byte("\n"))
	if len(lines) != 2 {
		return nil, fmt.Errorf("expected %s to contain two lines but found %d lines", checksumPath, len(lines))
	}
	return lines[0], nil
}

func checksumS3URL(s3url *url.URL) ([]byte, error) {
	head, err := headS3URL(s3url)
	if err != nil {
		return nil, err
	}

	return checksumFromHead(head), nil
}

func checksumFromHead(head *s3.HeadObjectOutput) []byte {
	// For multipart uploads from s3cmd, we can find the original md5
	// in a s3cmd specific metadata field 'S3cmd-Attrs':
	if s3vals, ok := head.Metadata["S3cmd-Attrs"]; ok {
		pairs := strings.Split(*s3vals, "/")
		for _, p := range pairs {
			if strings.HasPrefix(p, "md5:") {
				return []byte(strings.TrimPrefix(p, "md5:"))
			}
		}
	}

	return bytes.Trim([]byte(*head.ETag), "\"")
}

func headS3URL(s3url *url.URL) (*s3.HeadObjectOutput, error) {
	region, err := objectRegion(s3url)
	if err != nil {
		return nil, fmt.Errorf("unable to determine region: %s", err)
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	// Re-create client for bucket's region
	s3client := s3.New(sess, aws.NewConfig().WithRegion(region))

	key := strings.TrimPrefix(s3url.Path, "/")
	headObjInput := &s3.HeadObjectInput{
		Bucket: &s3url.Host,
		Key:    &key,
	}
	return s3client.HeadObject(headObjInput)
}
