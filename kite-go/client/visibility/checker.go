package visibility

import (
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// trackSidebar will periodically check whether the sidebar is visible, and sets
// visibleRecently accordingly. visibleRecently means that the app was visibile in the last
// 10 seconds. This is controlled by the interval and recencyThreshold. The reason for having a recencyThreshold
// vs setting the interval to 10 is because we want visibleRecently to reflect whether the sidebar was visible
// at all in the last 10 seconds. If we simply use a 10sec interval, we still have the original race.
//
//                                        user closes sidebar at (t+19)
//                                                |
//                   <------(t)-------(t+10)------x-(t+20)-x------->
//  visibleRecently         YES        YES            NO   |
//                                                         code checks visibleRecently
//
// In this case, visibleRecently reports NO even though the sidebar has been visible in the previous 10 seconds.
// We need to be able to absorb a few "not visible" checks to actually span the entire 10 second history we want.
//
//             user closes sidebar at (t+19)           visibleRecently switches to NO, after having been YES for 11 seconds
//                       |                                |
//                   <---x-(t+20)-x-------(t+25)--------(t+30)----------(t+35)--------->
//  visibleRecently         YES   |         YES            NO             NO
//  notVisibleCount          1    |          2             3              4
//                                value is as expected
//
// This allows us to report visibleRecently correctly. More specifically, visibleRecently will remain YES for 10-15 seconds
// after the sidebar was last visible.
//
const (
	recencyThreshold = 2
	Interval         = 5 * time.Second
)

// A Listener receives visibility updates
type Listener func(visibleNow, visibleRecently bool)

// checker periodically checks for visibility of the Kite window and reports
// visible and occlude events to mixpanel.
type checker struct {
	Recently     bool // whether the sidebar has been visible in the past 10 seconds
	count        int  // number of consecutive occluded results
	ticker       *time.Ticker
	listeners    map[string]Listener
	listenerLock sync.Mutex
}

// newChecker creates a sidebar visibility checker
func newChecker() *checker {
	checker := checker{
		ticker:    time.NewTicker(Interval),
		listeners: make(map[string]Listener),
	}
	go checker.loop()
	return &checker
}

// Listen registers a callback that will receive visibility notifications. Any
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

// loop runs the visibility polling loop
func (c *checker) loop() {
	for range c.ticker.C {
		c.check()
	}
}

// check performs a single visibility check
func (c *checker) check() {
	defer func() {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
		}
	}()

	// test for visibility (uses platform-specific methods)
	visibleNow := windowVisible()

	// update the visible-recently state
	if visibleNow {
		c.Recently = true
		c.count = 0
	} else {
		c.count++
		if c.count > recencyThreshold {
			c.Recently = false
		}
	}

	// notify the listeners
	c.listenerLock.Lock()
	defer c.listenerLock.Unlock()
	for _, listener := range c.listeners {
		listener(visibleNow, c.Recently)
	}
}
