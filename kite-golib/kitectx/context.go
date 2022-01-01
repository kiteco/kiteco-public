// Package kitectx encapsulates the capability to abort computations.
//
// kitectx.Context is analogous to the built-in context.Context, with some helper methods
// to easily define abort-able computations. ctx.CheckAbort() must be called sufficiently
// frequently during a computation in order for the abort condition to be checked.
//
// As a rule of thumb, ctx.CheckAbort should be called either directly or indirectly (by calling
// another function that calls it) at the start of every function that accepts a kitectx.Context.
//
// NOTE: Context is not go routine safe in the sense that a child Context should never be created
// in a separate go routine from the parent Context. The issue is that the child context will typically not
// have a handler to catch if the parent context panics, this means that if the parent context is cancelled
// before the child context then when the child context checks the abort condition it will see that
// the parent context has been cancelled and will bubble up the panic, but since the child context
// is in a separate go routine there will typically not be a handler to catch the parent context's panic
// and the go routine that contains the child context will be brought down.
// SEE: kite-go/lang/python/pythoncall/model.go for an example of why this does not work
package kitectx

import (
	"context"
	"sync/atomic"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/kitelog"
)

// Context manages an abort condition and a Kite logger
// it typically should be passed explicitly to functions rather than stored in another type
// if a function accepts a Context, it should typically also call Context.CheckAbort at the top
type Context struct {
	context context.Context
	expired *unsafe.Pointer // pointer to unsafe.Pointer to expiry error
	Logger  *kitelog.Logger
}

// waitExpiry waits until ctx's underlying context.Context is expired, and sets the expired flag
func (ctx Context) waitExpiry() {
	stdctx := ctx.Context()
	if done := stdctx.Done(); done != nil {
		<-done
		err := stdctx.Err()
		atomic.StorePointer(ctx.expired, unsafe.Pointer(&err))
	}
}

// withContext handles asynchronously setting the expired flag
func (ctx Context) withContext(std context.Context) Context {
	ctx.context = std
	ctx.expired = new(unsafe.Pointer)
	go ctx.waitExpiry()
	return ctx
}

// Background returns a context that doesn't expire
func Background() Context {
	return Context{
		Logger: kitelog.Basic,
	}
}

// TODO returns a context that doesn't expire
func TODO() Context {
	return Background()
}

// WithLogger returns a new Context with the provided kitelog.Logger set
func (ctx Context) WithLogger(l *kitelog.Logger) Context {
	ctx.Logger = l
	return ctx
}

// Context returns a context.Context for use with libraries/packages that don't support kitectx
func (ctx Context) Context() context.Context {
	if ctx.context == nil {
		return context.Background()
	}
	return ctx.context
}

// IsDeadlineExceeded checks if the error is a context expired error
func IsDeadlineExceeded(err error) bool {
	switch err {
	case context.DeadlineExceeded, ContextExpiredError{context.DeadlineExceeded}:
		return true
	}
	return false
}
