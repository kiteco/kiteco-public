package localfiles

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

var (
	// contentHashSet is a dataset of content hashes that are known to exist for at least
	// 10 users. This is used to avoid uploading content multiple times. See the /content endpoint below.
	contentHashSet = "s3://kite-data/localfiles/2017-06-05/contenthashes.gob.gz"
)

// ContentStoreType defines different file systems available for storing file content.
type ContentStoreType string

const (
	// LocalContentStore uses the local file system to store file content.
	LocalContentStore ContentStoreType = "local"

	// S3ContentStore uses S3 to store file content.
	S3ContentStore ContentStoreType = "s3"

	// InMemoryContentStore uses an in-memory map to store file content.
	InMemoryContentStore ContentStoreType = "mem"
)

// --

type contentFileSystem interface {
	write(string, []byte) error
	exists(string) (bool, error)
	read(string) ([]byte, error)
}

// --

type s3ContentFileSystem struct {
	bucketName    string
	region        string
	contenthashes map[string]bool
}

func newS3ContentFileSystem(bucketName, region string) (*s3ContentFileSystem, error) {
	fs := &s3ContentFileSystem{
		bucketName:    bucketName,
		region:        region,
		contenthashes: make(map[string]bool),
	}

	err := fs.loadContentHashSet(contentHashSet)
	if err != nil {
		log.Println("error loading content hash set:", err)
	}

	return fs, nil
}

func (s *s3ContentFileSystem) write(key string, buf []byte) error {
	fileCounter.Add(1)
	fileSizeSample.Record(int64(len(buf)))
	start := time.Now()
	defer func() {
		writeDuration.RecordDuration(time.Since(start))
	}()

	b := &bytes.Buffer{}
	comp := gzip.NewWriter(b)
	_, err := comp.Write(buf)
	if err != nil {
		return err
	}
	if err = comp.Close(); err != nil {
		return err
	}
	s3Client, err := s.getS3Client()
	if err != nil {
		return err
	}
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(b.Bytes()),
		ContentType: aws.String("binary/octet-stream"),
	}
	_, err = s3Client.PutObject(input)
	if err != nil {
		return err
	}
	return nil
}

func (s *s3ContentFileSystem) exists(key string) (bool, error) {
	return s.contenthashes[key], nil
}

func (s *s3ContentFileSystem) read(key string) ([]byte, error) {
	s3Client, err := s.getS3Client()
	if err != nil {
		return nil, err
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}
	output, err := s3Client.GetObject(input)
	if err != nil {
		return nil, err
	}
	decomp, err := gzip.NewReader(output.Body)
	if err != nil {
		return nil, err
	}
	defer decomp.Close()
	return ioutil.ReadAll(decomp)
}

func (s *s3ContentFileSystem) getS3Client() (*s3.S3, error) {
	sess, err := session.NewSession()
	if err != nil {
		log.Printf("error getting aws session: %s", err)
		return nil, err
	}

	return s3.New(sess, aws.NewConfig().WithRegion(s.region)), nil
}

func (s *s3ContentFileSystem) loadContentHashSet(path string) error {
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gunzip, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	err = gob.NewDecoder(gunzip).Decode(&s.contenthashes)
	if err != nil {
		return err
	}

	log.Printf("loaded %d content hashes", len(s.contenthashes))
	return nil
}

// --

type localContentFileSystem struct {
	root string
}

func newLocalContentFileSystem(root string) (*localContentFileSystem, error) {
	err := os.MkdirAll(root, os.ModePerm)
	if err != nil {
		log.Printf("could not create directory %s: %s", root, err)
		return nil, err
	}
	return &localContentFileSystem{
		root: root,
	}, nil
}

func (l *localContentFileSystem) write(key string, buf []byte) error {
	filename := path.Join(l.root, key)
	dir := path.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf, os.ModePerm)
}

func (l *localContentFileSystem) exists(key string) (bool, error) {
	filename := path.Join(l.root, key)
	if _, err := os.Stat(filename); err != nil {
		return false, err
	}
	return true, nil
}

func (l *localContentFileSystem) read(key string) ([]byte, error) {
	filename := path.Join(l.root, key)
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("file not found")
	}
	return content, nil
}

// --

type inMemoryContentFileSystem struct {
	files map[string][]byte
	mutex sync.Mutex
}

func newInMemoryContentFileSystem() (*inMemoryContentFileSystem, error) {
	return &inMemoryContentFileSystem{
		files: make(map[string][]byte),
		mutex: sync.Mutex{},
	}, nil
}

func (m *inMemoryContentFileSystem) write(key string, buf []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.files[key] = buf
	return nil
}

func (m *inMemoryContentFileSystem) exists(key string) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.files[key]; !exists {
		return false, fmt.Errorf("file not found")
	}
	return true, nil
}

func (m *inMemoryContentFileSystem) read(key string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	content, exists := m.files[key]
	if !exists {
		return nil, fmt.Errorf("file not found")
	}
	return content, nil
}
