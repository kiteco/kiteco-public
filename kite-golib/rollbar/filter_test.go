package rollbar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_FilterSlowRate(t *testing.T) {
	// accept every message, send out at most one message every 10s
	isAccepted := newRollbarLimiter(1, 10*time.Second)

	assert.True(t, isAccepted(), "the first message should've been accepted")

	accepted := 0
	for i := 0; i < 10000; i++ {
		if isAccepted() {
			accepted++
		}
	}
	assert.EqualValues(t, 0, accepted, "only the first message should've been accepted")
}
