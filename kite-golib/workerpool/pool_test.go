package workerpool

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_RunJobs(t *testing.T) {
	pool := New(5)

	var jobs []Job
	var completed int32
	for i := 0; i < 15; i++ {
		jobs = append(jobs, func() error {
			time.Sleep(time.Second)
			atomic.AddInt32(&completed, 1)
			return nil
		})
	}

	pool.Add(jobs)
	pool.Wait()
	require.EqualValues(t, len(jobs), completed, "expected all jobs to be completed")
}
func Test_StopWait(t *testing.T) {
	pool := New(5)

	var jobs []Job
	for i := 0; i < 15; i++ {
		jobs = append(jobs, func() error {
			time.Sleep(time.Second)
			return nil
		})
	}

	pool.Add(jobs)
	<-time.After(time.Second)
	pool.Stop()
	pool.Wait()
}
