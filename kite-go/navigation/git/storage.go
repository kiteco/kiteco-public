package git

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
)

// Storage ...
type Storage interface {
	read(int64) ([]byte, error)
	write([]byte, int64) error
	lock()
	unlock()
}

// StorageOptions ...
type StorageOptions struct {
	UseDisk bool
	Path    string
}

// NewStorage ...
func NewStorage(opts StorageOptions) (Storage, error) {
	if !opts.UseDisk {
		return newSliceStorage(), nil
	}
	return newDiskStorage(opts.Path)
}

var errDataExceedsMaxSize = errors.New("data exceeds maxSize")

type diskStorage struct {
	path localpath.Absolute
	m    *sync.Mutex
}

func newDiskStorage(path string) (diskStorage, error) {
	abs, err := localpath.NewAbsolute(path)
	if err != nil {
		return diskStorage{}, err
	}
	return diskStorage{
		path: abs,
		m:    new(sync.Mutex),
	}, nil
}

func (s diskStorage) lock() {
	s.m.Lock()
}

func (s diskStorage) unlock() {
	s.m.Unlock()
}

func (s diskStorage) read(maxSize int64) ([]byte, error) {
	file, err := s.path.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSize {
		return nil, errDataExceedsMaxSize
	}
	return data, nil
}

func (s diskStorage) write(data []byte, maxSize int64) error {
	if int64(len(data)) > maxSize {
		return errDataExceedsMaxSize
	}
	return ioutil.WriteFile(string(s.path), data, storagePermissions)
}

func readFromStorage(s Storage, maxSize int64) (repoBundle, error) {
	data, err := s.read(maxSize)
	if os.IsNotExist(err) {
		return newRepoBundle(), nil
	}
	if err != nil {
		return repoBundle{}, err
	}
	return unmarshalRepoBundle(data)
}

type sliceStorage struct {
	data []byte
	m    *sync.Mutex
}

func newSliceStorage() *sliceStorage {
	return &sliceStorage{
		m: new(sync.Mutex),
	}
}

func (s sliceStorage) lock() {
	s.m.Lock()
}

func (s sliceStorage) unlock() {
	s.m.Unlock()
}

func (s sliceStorage) read(maxSize int64) ([]byte, error) {
	if s.data == nil {
		return nil, os.ErrNotExist
	}
	size := len(s.data)
	if size > int(maxSize) {
		return nil, errDataExceedsMaxSize
	}
	data := make([]byte, size)
	copy(data, s.data)
	return data, nil
}

func (s *sliceStorage) write(data []byte, maxSize int64) error {
	if int64(len(s.data)) > maxSize {
		return errDataExceedsMaxSize
	}
	s.data = data
	return nil
}
