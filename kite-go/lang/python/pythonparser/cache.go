package pythonparser

import (
	"sync"
	"time"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-golib/collections"
)

const (
	// parseCacheSize specifies the max number of parsed files to cache
	parseCacheSize = 1000
	// staleCutoff specifies when cache entries are considered stale
	staleCutoff = 10 * time.Minute
)

var (
	lock       sync.Mutex
	parseCache = collections.NewOrderedMap(parseCacheSize + 1)
)

type parseEntry struct {
	lastAccessTs time.Time
	mod          *pythonast.Module
	err          error
}

// PurgeParseCache purges the parse cache
func PurgeParseCache() {
	lock.Lock()
	defer lock.Unlock()
	parseCache.RangeInc(func(k, v interface{}) bool {
		parseCache.Delete(k)
		return true
	})
}

// --

func getCachedParse(contents []byte) (*parseEntry, bool) {
	hash := hashContents(contents)
	lock.Lock()
	defer lock.Unlock()
	entry, ok := parseCache.Get(hash)
	removeStaleCacheEntriesLocked()
	if !ok {
		return nil, false
	}
	// update last access ts if entry existed
	cacheParseLocked(hash, entry.(*parseEntry).mod, entry.(*parseEntry).err)
	return entry.(*parseEntry), ok
}

func cacheParse(contents []byte, mod *pythonast.Module, err error) {
	lock.Lock()
	defer lock.Unlock()
	removeStaleCacheEntriesLocked()
	hash := hashContents(contents)
	cacheParseLocked(hash, mod, err)
}

func cacheParseLocked(hash uint64, mod *pythonast.Module, err error) {
	entry, ok := parseCache.Delete(hash)
	if entry == nil || !ok {
		entry = &parseEntry{}
	}

	*(entry.(*parseEntry)) = parseEntry{
		lastAccessTs: time.Now(),
		mod:          mod,
		err:          err,
	}
	parseCache.Set(hash, entry)
}

// removeStaleCacheEntriesLocked removes entries with a timestamp greater than the cutoff
func removeStaleCacheEntriesLocked() {
	parseCache.RangeDec(func(k, v interface{}) bool {
		if parseCache.Len() > parseCacheSize || time.Since(v.(*parseEntry).lastAccessTs) > staleCutoff {
			parseCache.Delete(k)
			return true
		}
		return false
	})
}

// --

func hashContents(contents []byte) uint64 {
	return spooky.Hash64(contents)
}
