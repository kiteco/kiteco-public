package codebase

import (
	"context"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type terminator struct {
	terminated bool
	cancel     context.CancelFunc
	m          *sync.Mutex
}

func newTerminator() terminator {
	return terminator{
		m: new(sync.Mutex),
	}
}

func (t *terminator) terminate() {
	t.m.Lock()
	defer t.m.Unlock()

	t.terminated = true
	if t.cancel != nil {
		t.cancel()
	}
}

func (t terminator) wasTerminated() bool {
	t.m.Lock()
	defer t.m.Unlock()
	return t.terminated
}

func (t *terminator) closureWithCancel(fn func(ctx kitectx.Context) error) (func() error, kitectx.CancelFunc) {
	t.m.Lock()
	defer t.m.Unlock()

	var closure func() error
	closure, t.cancel = kitectx.Background().ClosureWithCancel(fn)
	return closure, t.cancel
}
