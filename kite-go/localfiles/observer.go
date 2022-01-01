package localfiles

import (
	"fmt"
	"sync"
)

var (
	observerMutex sync.RWMutex
	observers     = map[observerKey]ObserverFunc{}
)

type observerKey struct {
	uid int64
	mid string
}

// ObserverFunc is the callback function used when local file changes are observed
// for a user/machine id pair. It is passed in a list of files that have been added,
// changed or removed.
type ObserverFunc func([]string)

// Observe registers an observer for the provided user id and machine id. This is a very
// basic observer model implementation, allowing for a single observer for a user id / machine id pair.
// This will definitely need more work before it can be used generally.
func Observe(uid int64, mid string, fn ObserverFunc) error {
	observerMutex.Lock()
	defer observerMutex.Unlock()

	key := observerKey{uid, mid}
	if _, exists := observers[key]; exists {
		return fmt.Errorf("observer already registered for uid: %d mid: %s", uid, mid)
	}

	observers[key] = fn
	return nil
}

// RemoveObserver will remove an observer associated with a user id / machine id pair.
func RemoveObserver(uid int64, mid string) {
	observerMutex.Lock()
	defer observerMutex.Unlock()

	key := observerKey{uid, mid}
	delete(observers, key)
}

func triggerObservers(uid int64, mid string, files []string) {
	observerMutex.RLock()
	defer observerMutex.RUnlock()

	key := observerKey{uid, mid}
	if fn, exists := observers[key]; exists {
		fn(files)
	}
}
