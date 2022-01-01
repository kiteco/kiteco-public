package localfiles

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

var (
	//DefaultContentStoreOptions contains default options for ContentStore
	DefaultContentStoreOptions = ContentStoreOptions{
		Type:            S3ContentStore,
		MaxWriteRetries: 3,
	}
)

// ContentStoreOptions contains parameters to configure the Content Store.
type ContentStoreOptions struct {
	Type            ContentStoreType
	BucketName      string
	Region          string
	MaxWriteRetries int
}

// ContentStore wraps a file system for storing file content.
type ContentStore struct {
	opts   ContentStoreOptions
	fs     contentFileSystem
	FileDB *sqlx.DB
	Files  *FileManager
}

// NewContentStore creates a new ContentStore.
func NewContentStore(opts ContentStoreOptions, fdb *sqlx.DB) (*ContentStore, error) {
	var fs contentFileSystem
	var err error
	switch opts.Type {
	case S3ContentStore:
		fs, err = newS3ContentFileSystem(opts.BucketName, opts.Region)
	case LocalContentStore:
		fs, err = newLocalContentFileSystem("/var/kite/localcontent")
	case InMemoryContentStore:
		fs, err = newInMemoryContentFileSystem()
	}

	if err != nil {
		return nil, err
	}

	return &ContentStore{
		opts:   opts,
		fs:     fs,
		FileDB: fdb,
		Files:  NewFileManager(fdb),
	}, nil
}

// ContentStoreFromEnv creates a content store from environment variables
func ContentStoreFromEnv(driver, uri, storeType string) (*ContentStore, error) {
	fdb := FileDB(driver, uri)

	opts := DefaultContentStoreOptions
	switch ContentStoreType(storeType) {
	case S3ContentStore:
		opts.Type = S3ContentStore
		opts.BucketName = envutil.MustGetenv("LOCALFILES_S3_BUCKET")
		opts.Region = envutil.GetenvDefault("AWS_REGION", "us-west-1")
		log.Printf("local file content store using bucket %s", opts.BucketName)
	case LocalContentStore:
		log.Printf("local file content store using local filesystem")
		opts.Type = LocalContentStore
	case InMemoryContentStore:
		log.Printf("local file content store using in-memory storage")
		opts.Type = InMemoryContentStore
	default:
		return nil, fmt.Errorf("unrecognized content store type: %s", storeType)
	}

	store, err := NewContentStore(opts, fdb)
	if err != nil {
		return nil, fmt.Errorf("error creating content store: %v", err)
	}
	return store, nil
}

// Close will close the underlying database connection
func (s *ContentStore) Close() error {
	return s.FileDB.Close()
}

// Migrate sets up the proper table columns in the database.
func (s *ContentStore) Migrate() error {
	db, err := gorm.Open("postgres", s.FileDB)
	if err != nil {
		return fmt.Errorf("error creating tables: %s", err)
	}
	db.SingularTable(true)
	if err := db.AutoMigrate(&File{}).Error; err != nil {
		return fmt.Errorf("error creating tables: %s", err)
	}

	// Add uid_mid_name_uidx unique index
	_, err = s.FileDB.Exec("CREATE UNIQUE INDEX uid_mid_name_uidx ON file (user_id, machine, name)")
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("error adding index: %v", err)
		}
		log.Println(err)
	}
	// Add uid_mid_updated_idx index
	_, err = s.FileDB.Exec("CREATE INDEX uid_mid_updated_idx ON file (user_id, machine, updated_at)")
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("error adding index: %v", err)
		}
		log.Println(err)
	}
	// Add updated_at_idx index
	_, err = s.FileDB.Exec("CREATE INDEX updated_at_idx ON file (updated_at)")
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("error adding index: %v", err)
		}
		log.Println(err)
	}
	return nil
}

// Get returns the content for the file with HashedContent hash.
func (s *ContentStore) Get(hash string) ([]byte, error) {
	content, err := s.fs.read(hash)
	return content, err
}

// Put adds a file to the database and storage backend
func (s *ContentStore) Put(uid int64, machine, name string, content []byte) error {
	if content == nil {
		return nil
	}

	hash := ComputeHash(content)
	err := s.putContent(hash, content)
	if err != nil {
		return err
	}

	event := &FileEvent{
		File: &File{
			UserID:        uid,
			Machine:       machine,
			Name:          name,
			HashedContent: hash,
		},
		Content: content,
		Type:    ModifiedEvent,
	}

	err = s.Files.BatchCreateOrUpdate([]*FileEvent{event})
	if err != nil {
		return err
	}

	return nil
}

// Delete removes a file from the database.
func (s *ContentStore) Delete(uid int64, machine, name string) error {
	event := &FileEvent{
		File: &File{
			UserID:  uid,
			Machine: machine,
			Name:    name,
		},
		Type: RemovedEvent,
	}

	return s.Files.Delete(event)
}

// Exists returns whether or not the file with the hash exists.
func (s *ContentStore) Exists(hash string) (bool, error) {
	return s.fs.exists(hash)
}

// --

func (s *ContentStore) putContent(key string, content []byte) error {
	var err error
	for i := 0; i < s.opts.MaxWriteRetries; i++ {
		err = s.fs.write(key, content)
		if err != nil {
			log.Printf("putContent attempt %d for %s failed with err: %s", i, key, err)
			time.Sleep(time.Duration(math.Pow(2.0, float64(i+2))) * time.Second)
			continue
		}

		return nil
	}

	return fmt.Errorf("putContent failed after %d retries for %s: %s", s.opts.MaxWriteRetries, key, err)
}
