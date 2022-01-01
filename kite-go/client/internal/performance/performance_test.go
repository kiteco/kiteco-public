package performance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_PerformanceValues(t *testing.T) {
	mem := MemoryUsage()
	assert.True(t, mem > 0)

	version := OsVersion()
	assert.True(t, version != "")

	// start cpu-intensive goroutine
	go func() {
		fib(50)
	}()

	time.Sleep(1 * time.Second)
	cpu := CPUUsage()
	assert.NotZero(t, cpu)
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}
