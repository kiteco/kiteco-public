package sandbox

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartAndWait(t *testing.T) {
	p, err := StartSubprocess(&ProcessOptions{
		Command: "echo",
		Args:    []string{"foo"},
	})
	require.NoError(t, err)
	defer p.Cleanup()

	stdout, stderr, err := p.Wait()
	if !assert.NoError(t, err) {
		return
	}
	assert.Len(t, stderr, 0)
	assert.Equal(t, "foo\n", string(stdout))
}

func TestStartAndCancel(t *testing.T) {
	p, err := StartSubprocess(&ProcessOptions{
		Command: "yes",
	})
	require.NoError(t, err)
	defer p.Cleanup()

	_, stderr := p.Cancel()
	assert.Len(t, stderr, 0)
}

func TestStartAndRescind(t *testing.T) {
	p, err := StartSubprocess(&ProcessOptions{
		Command: "yes",
	})
	require.NoError(t, err)
	defer p.Cleanup()

	rescindChan := make(chan error)
	go func() {
		rescindChan <- errors.New("we decided to cancel")
	}()

	_, stderr, err := p.RescindableWait(rescindChan)
	assert.Len(t, stderr, 0)
	assert.EqualError(t, err, "we decided to cancel")
}
