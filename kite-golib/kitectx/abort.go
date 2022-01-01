package kitectx

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// - aborts and recovery

type checkAbortPanic struct {
	err error
}

func abort(err error) {
	panic(checkAbortPanic{err})
}

func recoverAbort(parentCheck func(), err *error) {
	if v := recover(); v != nil {
		switch v := v.(type) {
		case checkAbortPanic:
			if parentCheck != nil {
				parentCheck() // continue unwinding if the parent Context is also expired
			}
			// otherwise, set the error
			*err = ContextExpiredError{v.err}

			// record metrics
			if globalMetrics != nil {
				globalMetrics.hit(v.err)
			}
		default:
			// all other panics should continue unwinding
			panic(v)
		}
	}
}

// - API

// ContextExpiredError is returned a computation is aborted due context expiry
type ContextExpiredError struct {
	Err error
}

// Error implements error
func (c ContextExpiredError) Error() string {
	return fmt.Sprintf("kitectx.Context expired: %s", c.Err)
}

// CheckAbort aborts if ctx is expired
func (ctx Context) CheckAbort() {
	// NOTE: this code is duplicated in Context.CheckAbort, CallContext.CheckAbort
	// NOTE: it should be kept in sync as the duplication allows for inlining
	if ctx.expired != nil { // a zero Context is not expired
		errPtr := (*error)(atomic.LoadPointer(ctx.expired))
		if errPtr != nil {
			abort(*errPtr)
		}
	}
}

// FromContext calls a function with a Context that expires when a given context.Context expires.
// The provided context.Context should expire; otherwise FromContext will leak a goroutine.
func FromContext(std context.Context, f func(Context) error) (err error) {
	if std == nil {
		panic("kitectx.FromContext called on nil context.Context")
	}

	if err := std.Err(); err != nil {
		return ContextExpiredError{err}
	}

	defer recoverAbort(nil, &err)
	ctx := Background().withContext(std)
	err = f(ctx)
	return
}

// WithTimeout is equivalent to WithDeadline called on time.Now().Add(timeout)
func (ctx Context) WithTimeout(timeout time.Duration, f func(Context) error) error {
	return ctx.WithDeadline(time.Now().Add(timeout), f)
}

// WithDeadline calls a function with a Context that expires at a given deadline.
func (ctx Context) WithDeadline(deadline time.Time, f func(Context) error) (err error) {
	defer recoverAbort(ctx.CheckAbort, &err)

	newStd, cancel := context.WithDeadline(ctx.Context(), deadline)
	defer cancel()
	if err := newStd.Err(); err != nil {
		return ContextExpiredError{err}
	}

	err = f(ctx.withContext(newStd))
	return
}

// CancelFunc = context.CancelFunc
type CancelFunc = context.CancelFunc

// ClosureWithCancel is semantically equivalent to the standard context.WithCancel.
// It returns a closure that will executed the provided callback with a cancellable Context.
// Either the returned closure or CancelFunc must be called to avoid resource leaks.
func (ctx Context) ClosureWithCancel(f func(Context) error) (func() error, CancelFunc) {
	newStd, cancel := context.WithCancel(ctx.Context())
	newCtx := ctx.withContext(newStd)
	return func() (err error) {
			defer recoverAbort(ctx.CheckAbort, &err)

			defer cancel()

			err = f(newCtx)
			return
		}, func() {
			cancel()
			newCtx.waitExpiry()
		}
}

// WithCancel is semantically equivalent to the standard context.WithCancel
func (ctx Context) WithCancel(f func(Context, CancelFunc) error) error {
	var cancel CancelFunc
	g, cancel := ctx.ClosureWithCancel(func(ctx Context) error {
		return f(ctx, cancel)
	})
	return g()
}
