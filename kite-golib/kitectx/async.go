package kitectx

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// Go starts a new goroutine with rollbar & kitectx handlers
// which will send an error to the returned (buffered) channel upon termination
func Go(fn func() error) chan error {
	errC := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				rollbar.PanicRecovery(r)
			}
		}()

		err := func() (err error) {
			defer recoverAbort(nil, &err)
			return fn()
		}()
		errC <- err
	}()
	return errC
}

// Abort waits until ctx is expired and then aborts; it will block forever if ctx never aborts, so be careful
func (ctx Context) Abort() {
	ctx.waitExpiry()
	abort(ctx.context.Err())
}

// AbortChan returns a channel that will be closed if ctx is expired
// the caller must subsequently call Abort to abort (not CheckAbort, as CheckAbort is only eventually consistent)
func (ctx Context) AbortChan() <-chan struct{} {
	return ctx.Context().Done()
}

// GoWithTimeout spins off a goroutine to execute the given function. If the parent context expires or the
// timeout is reached, an error is returned. This function blocks until either `fn` returns or atleast a duration
// of `timeout` has passed.
func GoWithTimeout(ctx Context, timeout time.Duration, fn func(ctx Context) error) error {
	// NOTE: we need to buffer the channel here so that
	// if we return before our worker goroutine finishes
	// then the worker goroutine does not block forever.
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				rollbar.PanicRecovery(r)
			}
		}()

		// IMPORTANT: we explicitly need to use a separate background context here.
		//
		// This is because the incoming kitectx (lets call it parent_ctx) is typically derived from an http request,
		// and its underlying contex.Context is cancelled when the http request returns.
		//
		// This means that if we derive a child ctx (lets call it child_ctx) from parent_ctx and then do
		// child_ctx.CheckAbort() and the child context has timed out then ultimately as part of the abort process
		// for child_ctx we will check if parent_ctx has been cancelled, and since the http request has returned
		// the parent_ctx will be cancelled and we will bubble the abort signal back up to the parent's
		// handler.
		//
		// However, the goroutine that contains child_ctx (e.g this goroutine) does not have a handler
		// for the parent's abort signal and thus the abort signal (a panic) will not be caught in this goroutine
		// and that will cause this goroutine to crash with an unhandled panic which will ultimately bring the entire
		// process down.
		err := Background().WithTimeout(timeout, fn)
		errChan <- err
	}()

	t := time.NewTimer(timeout)
	defer t.Stop()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Context().Done():
		return ContextExpiredError{}
	case <-t.C:
		return ContextExpiredError{}
	}
}
