package kitectx

import (
	"testing"
)

// WaitExpiry waits until ctx's is expired.
// It is useful for testing, as expiry is only "eventually consistent" in kitectx,
// so clients cannot count on e.g. a cancel() to immediately expire the Context.
func (ctx Context) WaitExpiry(_ testing.TB) {
	ctx.waitExpiry()
}

// Sync forces ctx to be immediately consistent with any timeouts or cancellations.
// It returns the corresponding error if the context is expired.
func (ctx Context) Sync(_ testing.TB) error {
	if err := ctx.context.Err(); err != nil {
		ctx.waitExpiry()
		return ContextExpiredError{err}
	}
	return nil
}
