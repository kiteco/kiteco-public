package sysidle

import (
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	interval = 30 * time.Second
)

// A Listener receives system idle updates
type Listener func(sysIdle bool)

// checker periodically checks for system idle
type checker struct {
	ticker       *time.Ticker
	listeners    map[string]Listener
	listenerLock sync.Mutex
}

// newChecker creates a sidebar visibility checker
func newChecker() *checker {
	checker := checker{
		ticker:    time.NewTicker(interval),
		listeners: make(map[string]Listener),
	}
	go checker.loop()
	return &checker
}

// Listen registers a callback that will receive system idle notifications. Any
// previous listener registered with the same key will be removed.
func (c *checker) Listen(key string, l Listener) {
	c.listenerLock.Lock()
	defer c.listenerLock.Unlock()
	c.listeners[key] = l
}

// clear all listeners
func (c *checker) clear() {
	c.listenerLock.Lock()
	defer c.listenerLock.Unlock()
	c.listeners = make(map[string]Listener)
}

// loop runs the system idle polling loop
func (c *checker) loop() {
	for range c.ticker.C {
		c.check()
	}
}

// check performs a single system idle check
func (c *checker) check() {
	defer func() {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
		}
	}()

	// test for system idle (uses platform-specific methods)
	isIdle := sysIdle()

	// notify the listeners
	c.listenerLock.Lock()
	defer c.listenerLock.Unlock()
	for _, listener := range c.listeners {
		listener(isIdle)
	}
}
