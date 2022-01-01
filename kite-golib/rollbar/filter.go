package rollbar

import (
	"math/rand"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// newRollbarLimiter creates a new sampling, rate-limited filter
// it first samples, e.g. a rate of 3 means that every 3rd message would be accepted on average
// then the rate of messages is limited to one message in every rateLimitDelay
// it's safe for concurrent use
func newRollbarLimiter(downSampleRate int, rateLimitDelay time.Duration) func() bool {
	var mu sync.Mutex
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	rateLimiter := rate.NewLimiter(rate.Every(rateLimitDelay), 1)

	return func() bool {
		mu.Lock()
		defer mu.Unlock()
		ok := random.Int()%downSampleRate == 0

		return ok && rateLimiter.Allow()
	}
}
