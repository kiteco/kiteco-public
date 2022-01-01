package kitectx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFromContext(t *testing.T) {
	x, cancel := context.WithCancel(context.Background())
	err := FromContext(x, func(ctx Context) error {
		cancel()
		ctx.WaitExpiry(t)
		ctx.CheckAbort()
		return nil
	})
	require.Error(t, err)
}

func TestFromContext_Immediate(t *testing.T) {
	x, cancel := context.WithCancel(context.Background())
	cancel()
	err := FromContext(x, func(ctx Context) error {
		return nil
	})
	require.Error(t, err)
}

func TestNoWaitExpiry(t *testing.T) {
	// use retry-based test to test the asynchronous setting of the expiry flag
	// everywhere else we use the assume that code-path works and just use WaitExpiry
	err := Background().WithCancel(func(ctx Context, cancel CancelFunc) error {
		cancel()
		wait := time.Millisecond
		// wait up to 2^6 - 1 = 63 ms total
		for i := 0; i < 6; i++ {
			time.Sleep(wait)
			ctx.CheckAbort()
			wait *= 2
		}
		return nil
	})
	require.Error(t, err)
}

func TestWithCancel(t *testing.T) {
	err := Background().WithCancel(func(ctx Context, cancel CancelFunc) error {
		cancel()
		ctx.WaitExpiry(t)
		ctx.CheckAbort()
		return nil
	})
	require.Error(t, err)
}

func TestWithTimeout(t *testing.T) {
	var ran bool
	timeout := time.Millisecond
	err := Background().WithTimeout(timeout, func(ctx Context) error {
		ran = true
		ctx.WaitExpiry(t)
		ctx.CheckAbort()
		return nil
	})
	require.Error(t, err)
	require.True(t, ran)
}

func TestWithTimeout_Immediate(t *testing.T) {
	err := Background().WithTimeout(time.Duration(0), func(ctx Context) error {
		return nil
	})
	require.Error(t, err)
}

func TestError(t *testing.T) {
	err := Background().WithCancel(func(ctx Context, cancel CancelFunc) error {
		return errors.New("abort_test.TestError")
	})
	require.Error(t, err)
	require.Equal(t, err.Error(), "abort_test.TestError")
}

func TestNested_Inner(t *testing.T) {
	var innerErr, outerErr error
	outerErr = Background().WithCancel(func(outer Context, outerCancel CancelFunc) error {
		innerErr = outer.WithCancel(func(inner Context, innerCancel CancelFunc) error {
			innerCancel()
			inner.WaitExpiry(t)
			inner.CheckAbort()
			return nil
		})
		return nil
	})
	require.NoError(t, outerErr)
	require.Error(t, innerErr)
}

func TestNested_Outer(t *testing.T) {
	var innerErr, outerErr error
	outerErr = Background().WithCancel(func(outer Context, outerCancel CancelFunc) error {
		innerErr = outer.WithCancel(func(inner Context, innerCancel CancelFunc) error {
			outerCancel()
			inner.WaitExpiry(t)
			outer.WaitExpiry(t)
			inner.CheckAbort()
			return nil
		})
		return nil
	})
	require.Error(t, outerErr)
	require.NoError(t, innerErr)
}

func TestNested_InnerOuter(t *testing.T) {
	var innerErr, outerErr error
	outerErr = Background().WithCancel(func(outer Context, outerCancel CancelFunc) error {
		innerErr = outer.WithCancel(func(inner Context, innerCancel CancelFunc) error {
			innerCancel()
			inner.WaitExpiry(t)
			inner.CheckAbort()
			return nil
		})
		outerCancel()
		outer.WaitExpiry(t)
		outer.CheckAbort()
		return nil
	})
	require.Error(t, outerErr)
	require.Error(t, innerErr)
}
