package event

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

// StoreType defines different block store backends that are available
type StoreType string

const (
	// LocalStore uses the local filesystem as a block store
	LocalStore StoreType = "local"

	// S3Store uses S3 as a block store
	S3Store StoreType = "s3"

	// InMemoryStore uses an in-memory map as a block store
	InMemoryStore StoreType = "mem"
)

// --

type blockFileSystem interface {
	writeBlock(buf []byte, metadata *Metadata) error
	readBlock(metadata *Metadata) ([]byte, error)
}

// --

type s3BlockFileSystem struct {
	bucketName string
}

func newS3BlockFileSystem(bucketName string) *s3BlockFileSystem {
	fs := &s3BlockFileSystem{
		bucketName: bucketName,
	}
	bucket, err := fs.getBucket()
	if err != nil {
		log.Fatalln(err)
	}
	bucket.PutBucket(s3.Private)
	return fs
}

func (s *s3BlockFileSystem) writeBlock(buf []byte, metadata *Metadata) error {
	bucket, err := s.getBucket()
	if err != nil {
		return err
	}
	err = bucket.Put(metadata.Filename, buf, "binary/octet-stream", s3.Private, s3.Options{})
	if err != nil {
		return err
	}
	return nil
}

func (s *s3BlockFileSystem) readBlock(metadata *Metadata) ([]byte, error) {
	bucket, err := s.getBucket()
	if err != nil {
		return nil, err
	}
	return bucket.Get(metadata.Filename)
}

func (s *s3BlockFileSystem) getBucket() (*s3.Bucket, error) {
	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, fmt.Errorf("error authenticating with AWS: %s", err)
	}

	client := s3.New(auth, aws.USWest)
	return client.Bucket(s.bucketName), nil
}

// --

type localBlockFileSystem struct {
	root string
}

func newLocalBlockFileSystem(root string) *localBlockFileSystem {
	err := os.MkdirAll(root, os.ModePerm)
	if err != nil {
		log.Fatalf("could not create directory %s: %s", root, err)
	}
	return &localBlockFileSystem{
		root: root,
	}
}

func (l *localBlockFileSystem) writeBlock(buf []byte, metadata *Metadata) error {
	filename := path.Join(l.root, metadata.Filename)
	dir := path.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf, os.ModePerm)
}

func (l *localBlockFileSystem) readBlock(metadata *Metadata) ([]byte, error) {
	filename := path.Join(l.root, metadata.Filename)
	return ioutil.ReadFile(filename)
}

// --

type inMemoryBlockFileSystem struct {
	files map[string][]byte
	mutex sync.Mutex
}

func newInMemoryBlockFileSystem() *inMemoryBlockFileSystem {
	return &inMemoryBlockFileSystem{
		files: make(map[string][]byte),
		mutex: sync.Mutex{},
	}
}

func (m *inMemoryBlockFileSystem) writeBlock(buf []byte, metadata *Metadata) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.files[metadata.Filename] = buf
	return nil
}

func (m *inMemoryBlockFileSystem) readBlock(metadata *Metadata) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.files[metadata.Filename], nil
}
