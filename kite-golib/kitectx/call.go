package kitectx

import (
	"fmt"
	"sync/atomic"
)

// CallLimitError is returned by WithCallLimit if the callback is aborted due to exceeding the call limit
type CallLimitError struct{}

// Error implements error
func (c CallLimitError) Error() string {
	return fmt.Sprintf("kitectx.Context call limit reached")
}

// CallContext bundles a Context with a call limit, typically for recursive functions
type CallContext struct {
	Context Context
	limit   uint32 // the number of calls remaining before we hit the call limit
}

// CheckAbort aborts if the underlying Context is expired, or if the call limit has been reached
func (cctx CallContext) CheckAbort() {
	cctx.checkAbort(true)
}

func (cctx CallContext) checkAbort(checkLimit bool) {
	ctx := cctx.Context

	// NOTE: this code is duplicated in Context.CheckAbort, CallContext.CheckAbort
	// NOTE: it should be kept in sync as the duplication allows for inlining
	if ctx.expired != nil { // a zero Context is not expired
		errPtr := (*error)(atomic.LoadPointer(ctx.expired))
		if errPtr != nil {
			abort(*errPtr)
		}
	}

	if checkLimit && cctx.limit == 0 {
		abort(CallLimitError{})
	}
}

// WithCallLimit calls a function with a Context with a call limit (for recursive functions).
// Use cctx.Call() when passing cctx to a recursive call in order to increment the call count.
func (cctx Context) WithCallLimit(count uint32, f func(CallContext) error) (err error) {
	cctx.CheckAbort()
	defer recoverAbort(cctx.CheckAbort, &err)

	new := CallContext{
		Context: cctx,
		limit:   count,
	}
	err = f(new)
	return
}

// Call returns a new Context with a decremented call limit
// it is intended for use in recursive calls: foo(cctx.Call())
// if cctx is already at the call limit, cctx.Call() aborts
func (cctx CallContext) Call() CallContext {
	cctx.CheckAbort()
	cctx.limit--
	return cctx
}

// AtCallLimit checks if the next call to cctx.Call() would abort due to hitting the call limit.
// It also calls CheckAbort on the parent Context.
func (cctx CallContext) AtCallLimit() bool {
	cctx.checkAbort(false)
	return cctx.limit == 0
}
