package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	tickerDelay = 250 * time.Millisecond
	maxDelay    = 120 * time.Second
)

// waitFor return when either condition returns true or if waiting for true timed out. The max timeout is 2s.
// if the condition is still returning true after the timeout then the current tests is stopped with 'failNow()'
func waitFor(t *testing.T, condition func() bool, msgAndArgs ...interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), maxDelay)
	defer cancel()

	ticker := time.NewTicker(tickerDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if !condition() {
				require.FailNow(t, "waitFor timed out: ", msgAndArgs...)
			}

		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}
