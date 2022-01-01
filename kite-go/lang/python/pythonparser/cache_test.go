package pythonparser

import (
	"fmt"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
)

func Test_ParseCache(t *testing.T) {
	// make sure cache is empty on start
	PurgeParseCache()
	assert.Equal(t, 0, parseCache.Len(), "parse cache should be empty on start")

	// get on an empty parse cache should not return anything
	contents := []byte("test contents")
	p, ok := getCachedParse(contents)
	assert.False(t, ok, "contents should not exist")
	assert.Nil(t, p, "contents should not exist")

	opts := Options{
		ErrorMode: Recover,
	}

	// add ten entries
	for i := 0; i < 10; i++ {
		contents = []byte(fmt.Sprintf("test contents %d", i))
		Parse(kitectx.Background(), contents, opts)
	}
	assert.Equal(t, 10, parseCache.Len(), "parse cache should have ten entries.")

	// adding the same entries should not result in more items in the cache
	for i := 0; i < 10; i++ {
		contents = []byte(fmt.Sprintf("test contents %d", i))
		Parse(kitectx.Background(), contents, opts)
	}
	assert.Equal(t, 10, parseCache.Len(), "parse cache should have ten entries.")

	// purging the cache should result in an empty cache
	PurgeParseCache()
	assert.Equal(t, 0, parseCache.Len(), "parse cache should be empty after purge")
}

func Test_StaleCacheEntries(t *testing.T) {
	// make sure cache is empty on start
	PurgeParseCache()
	assert.Equal(t, 0, parseCache.Len(), "parse cache should be empty on start")

	contents := []byte("test contents")
	hash := hashContents(contents)
	now := time.Now()
	lock.Lock()
	parseCache.Set(hash, &parseEntry{
		lastAccessTs: now.Add(-20 * time.Minute),
	})
	lock.Unlock()
	assert.Equal(t, 1, parseCache.Len(), "parse cache should have one entry")

	// getting entry that does not exist should cause stale entries to be removed
	getCachedParse([]byte("not exist"))
	assert.Equal(t, 0, parseCache.Len(), "parse cache should be empty")

	// getting entry that exists should cause stale entries to be removed
	oldContents := []byte("old test contents")
	oldHash := hashContents(oldContents)
	now = time.Now()
	lock.Lock()
	parseCache.Set(hash, &parseEntry{
		lastAccessTs: time.Now(),
	})
	parseCache.Set(oldHash, &parseEntry{
		lastAccessTs: now.Add(-20 * time.Minute),
	})
	lock.Unlock()
	assert.Equal(t, 2, parseCache.Len(), "parse cache should have two entries")
	getCachedParse(contents)
	assert.Equal(t, 1, parseCache.Len(), "parse cache should have one entry")

	// parsing should cause stale entries to be removed
	PurgeParseCache()
	opts := Options{
		ErrorMode: Recover,
	}
	now = time.Now()
	lock.Lock()
	parseCache.Set(oldHash, &parseEntry{
		lastAccessTs: now.Add(-20 * time.Minute),
	})
	lock.Unlock()
	assert.Equal(t, 1, parseCache.Len(), "parse cache should have one entry")
	// parse should add a new entry and remove the stale entry
	Parse(kitectx.Background(), contents, opts)
	assert.Equal(t, 1, parseCache.Len(), "parse cache should have one entry")
}

func Test_LimitCacheEntries(t *testing.T) {
	// make sure cache is empty on start
	PurgeParseCache()
	assert.Equal(t, 0, parseCache.Len(), "parse cache should be empty on start")

	now := time.Now()
	for i := 0; i < parseCacheSize+5; i++ {
		contents := []byte(fmt.Sprintf("test contents %d", i))
		hash := hashContents(contents)
		lock.Lock()
		parseCache.Set(hash, &parseEntry{
			lastAccessTs: now,
		})
		lock.Unlock()
	}
	assert.Equal(t, parseCacheSize+5, parseCache.Len(), "parse cache should have too many entries")

	lock.Lock()
	removeStaleCacheEntriesLocked()
	lock.Unlock()
	assert.Equal(t, parseCacheSize, parseCache.Len(), "parse cache should be at capacity")
}
