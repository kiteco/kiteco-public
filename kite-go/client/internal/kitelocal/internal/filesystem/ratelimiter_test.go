package filesystem

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_rate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	//allows 100ms duty per second
	var ratelimit = newDutycyclelimiter(0.1, time.Second)
	var count int64

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ratelimit.Take()
				atomic.AddInt64(&count, 1)
				time.Sleep(600 * time.Millisecond)
			}
		}
	}()

	time.Sleep(2 * time.Second)
	cancel()

	assert.EqualValues(t, 2, atomic.LoadInt64(&count), "only allow one Take() if the operation takes longer than the dutycycle")
}
