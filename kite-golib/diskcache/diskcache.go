// DiskCache is an LRU cache that reads and writes to the filesystem.

package diskcache

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/bufutil"
)

var (
	// ErrNoSuchKey is returned by Cache.Get when a key does not exist in the cache
	ErrNoSuchKey = errors.New("key does not exist in cache")
)

// Options represents options for a cache
type Options struct {
	MaxSize         int64 // MaxSize is the maximum total size of the cache in bytes
	BytesUntilFlush int64
}

// Cache represents a disk-based LRU cache
type Cache struct {
	Path            string
	opts            Options
	bytesSinceFlush int64 // bytes written since last flushCapacity
}

// Open creates a cache with contents stored as files in the given directory.
// It creates the directory if it does not already exist.
func Open(path string, opts Options) (*Cache, error) {
	err := os.MkdirAll(path, 0777)
	if err != nil {
		return nil, err
	}
	return &Cache{
		Path: path,
		opts: opts,
	}, nil
}

// OpenTemp creates a temporary directory and returns a cache backed by this
// directory. The user must remove the directory and any files within it when
// done.
func OpenTemp(opts Options) (*Cache, error) {
	path, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return Open(path, opts)
}

// Get looks up the value for the given key and returns it. If the key does not
// exist then ErrNotFound is returned.
func (c *Cache) Get(key []byte) ([]byte, error) {
	r, err := c.GetReader(key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}

// GetReader looks up the value for the given key and returns a reader to it. If
// the key does not exist then ErrNotFound is returned.
func (c *Cache) GetReader(key []byte) (io.ReadCloser, error) {
	h := hash(key)
	path := filepath.Join(c.Path, h)
	r, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSuchKey
		}
		return nil, err
	}
	return r, nil
}

// Exists flushs whether the key exists.
func (c *Cache) Exists(key []byte) bool {
	h := hash(key)
	path := filepath.Join(c.Path, h)
	_, err := os.Stat(path)
	return err == nil
}

// Put adds a key/value pair to the cache.
func (c *Cache) Put(key []byte, val []byte) error {
	c.bytesSinceFlush += int64(len(val))
	if c.bytesSinceFlush > c.opts.BytesUntilFlush {
		if err := c.flushCapacity(int64(len(val))); err != nil {
			return fmt.Errorf("error cleaning up cache: %v", err)
		}
		c.bytesSinceFlush = 0
	}

	h := hash(key)
	path := filepath.Join(c.Path, h)
	return ioutil.WriteFile(path, val, 0777)
}

// PutWriter adds a key/value pair to the cache via a io.WriteCloser
func (c *Cache) PutWriter(key []byte) (io.WriteCloser, error) {
	h := hash(key)
	path := filepath.Join(c.Path, h)
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return &putWriter{w: f, c: c}, nil
}

type putWriter struct {
	w       io.WriteCloser
	c       *Cache
	written int64
}

func (p *putWriter) Write(buf []byte) (int, error) {
	p.written += int64(len(buf))
	return p.w.Write(buf)
}

func (p *putWriter) Close() error {
	p.c.bytesSinceFlush += p.written
	if p.c.bytesSinceFlush > p.c.opts.BytesUntilFlush {
		if err := p.c.flushCapacity(p.written); err != nil {
			log.Printf("error cleaning up cache in putWriter: %v", err)
		} else {
			p.c.bytesSinceFlush = 0
		}
	}
	return p.w.Close()
}

// flushCapacity deletes old entries until there are at least n bytes left
// in the cache budget.
func (c *Cache) flushCapacity(n int64) error {
	files, err := ioutil.ReadDir(c.Path)
	if err != nil {
		return err
	}

	var sum int64
	for _, f := range files {
		sum += f.Size()
	}

	if sum+n <= c.opts.MaxSize {
		return nil
	}

	sort.Sort(byModTime(files))

	for _, f := range files {
		err := os.Remove(filepath.Join(c.Path, f.Name()))
		if err != nil {
			return err
		}
		sum -= f.Size()
		if sum+n <= c.opts.MaxSize {
			break
		}
	}
	return nil
}

type byModTime []os.FileInfo

func (xs byModTime) Len() int           { return len(xs) }
func (xs byModTime) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byModTime) Less(i, j int) bool { return xs[i].ModTime().Before(xs[j].ModTime()) }

func hash(key []byte) string {
	return hex.EncodeToString(bufutil.UintToBytes(spooky.Hash64(key)))
}
