package source

import (
	"sync/atomic"
	"time"
)

// CountQuiter ...
func CountQuiter(count *int64, max int64, interval time.Duration) chan struct{} {
	quit := make(chan struct{})
	go func() {
		if max == 0 {
			return
		}
		for range time.Tick(interval) {
			if atomic.LoadInt64(count) >= max {
				close(quit)
				return
			}
		}
	}()

	return quit
}
