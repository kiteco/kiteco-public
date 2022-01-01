package event

import (
	"sync"
	"time"
)

var (
	// DefaultBlockStoreOptions contains default options for event.BlockStore
	DefaultBlockStoreOptions = BlockStoreOptions{
		Type:      S3Store,
		BlockSize: (4 << 20), // 4 MB, uncompressed
	}
)

// BlockStoreOptions contains parameters to configure the block store.
type BlockStoreOptions struct {
	Type       StoreType
	BlockSize  int
	BucketName string
}

type userMachine struct {
	userID    int64
	machineID string
}

// BlockStore is an event storage system that is backed by a simple
// block store. There is a local filesystem and s3 based block store.
type BlockStore struct {
	opts    BlockStoreOptions
	fs      blockFileSystem
	manager *MetadataManager

	mutex   sync.Mutex
	drivers map[userMachine]*Driver
}

// NewBlockStore returns a new Store with contents stored on the
// chosen file system.
func NewBlockStore(mm *MetadataManager, opts BlockStoreOptions) *BlockStore {
	var fs blockFileSystem
	switch opts.Type {
	case S3Store:
		fs = newS3BlockFileSystem(opts.BucketName)
	case LocalStore:
		fs = newLocalBlockFileSystem("/var/kite/localevents")
	case InMemoryStore:
		fs = newInMemoryBlockFileSystem()
	}

	store := &BlockStore{
		opts:    opts,
		fs:      fs,
		manager: mm,
		drivers: make(map[userMachine]*Driver),
	}

	go store.flushLoop()
	return store
}

// DriverForUser returns the driver for the given user id,
// creating it if it does not exist.
func (b *BlockStore) DriverForUser(uid int64, machine string) *Driver {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	um := userMachine{uid, machine}
	if driver, exists := b.drivers[um]; exists {
		return driver
	}

	driver := newDriver(uid, machine, b.opts.BlockSize, b.fs, b.manager)
	b.drivers[um] = driver
	return driver
}

// RemoveDriver removes the driver from the block store's drivers.
func (b *BlockStore) RemoveDriver(driver *Driver) {
	driver.Flush()
	driver.Wait()
	b.mutex.Lock()
	defer b.mutex.Unlock()
	delete(b.drivers, userMachine{driver.uid, driver.machine})
}

// Flush calls Flush and Wait for all user drivers.
func (b *BlockStore) Flush() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, driver := range b.drivers {
		driver.Flush()
	}
	for _, driver := range b.drivers {
		driver.Wait()
	}
}

// --

const (
	flushTicker = 15 * time.Second
)

// flushLoop is run as a goroutine to periodically trigger flushes for all users drivers
func (b *BlockStore) flushLoop() {
	for range time.Tick(flushTicker) {
		b.mutex.Lock()
		for _, driver := range b.drivers {
			driver.flushTicker()
		}
		b.mutex.Unlock()
	}
}
