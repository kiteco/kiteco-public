package lazy

import (
	"sync"
)

// Loader allows for lazily loading & unloading data
type Loader struct {
	load   func() error
	unload func()

	lock    sync.RWMutex
	once    sync.Once
	loadErr error
}

// NewLoader creates a new Loader
func NewLoader(load func() error, unload func()) *Loader {
	return &Loader{
		load:   load,
		unload: unload,
	}
}

// LoadAndLock and ensures Load has run, and locks against Unloads until Unlock is called.
// Callers should take care to immediately defer l.Unlock() after verifying that l.LoadLock() has not returned an error.
func (l *Loader) LoadAndLock() error {
	// defer unlock if l.load() panics
	deferUnlock := true
	l.lock.RLock()
	defer func() {
		if deferUnlock {
			l.lock.RUnlock()
		}
	}()

	l.once.Do(func() { l.loadErr = l.load() })
	if l.loadErr == nil {
		// l.load() ran without a panic or error, don't defer unlock
		deferUnlock = false
	}
	return l.loadErr
}

// Unlock unlocks the Loader for Unloading
func (l *Loader) Unlock() {
	l.lock.RUnlock()
}

// Unload unloads the underlying data.
func (l *Loader) Unload() {
	// ensure we're not stepping on a readers toes
	l.lock.Lock()
	defer l.lock.Unlock()
	l.once = sync.Once{}
	l.unload()
	l.loadErr = nil
}
