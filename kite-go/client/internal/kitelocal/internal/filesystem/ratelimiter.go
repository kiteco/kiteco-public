package filesystem

import (
	"sync"
	"time"
)

// dutycyclelimiter is a basic bursty ratelimiter that will allow ops to go through
// a certain percentage of every cycle.
type dutycyclelimiter struct {
	duty  time.Duration
	cycle time.Duration

	m     sync.Mutex
	start time.Time
}

func newDutycyclelimiter(duty float64, cycle time.Duration) *dutycyclelimiter {
	return &dutycyclelimiter{
		duty:  time.Duration(duty * float64(cycle)),
		cycle: cycle,
	}
}

func (r *dutycyclelimiter) Take() {
	r.m.Lock()
	defer r.m.Unlock()

	// Set start time on first pass
	if r.start.IsZero() {
		r.start = time.Now()
		return
	}

	// If we are within the duty cycle, allow
	sinceStart := time.Since(r.start)
	if sinceStart < r.duty {
		return
	}

	time.Sleep(r.cycle - r.duty)
	r.start = time.Now()
}
