package javascript

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/diff"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Driver implements core.FileDriver and handles file events
// and maintaining the state of the current file and context.
type Driver struct {
	debug       bool
	diagnostics io.Writer

	file core.FileDriver

	lastEvent *event.Event

	// mediates between HTTP endpoints, which are read-only, and HandleEvent, which
	// modifies the state of this driver
	lock sync.RWMutex
}

// NewDriver creates a javascript driver
func NewDriver() *Driver {
	return &Driver{
		file: diff.NewBufferDriver(),
	}
}

// HandleEvent implements lang.Driver
func (d *Driver) HandleEvent(ctx kitectx.Context, evt *event.Event) string {
	ctx.CheckAbort()

	d.lock.Lock()
	defer d.lock.Unlock()
	return d.file.HandleEvent(ctx, evt)
}

// CollectOutput implements lang.Driver
func (d *Driver) CollectOutput() []interface{} {
	// Note: this function holds a read-only lock, so it must not modify the driver state
	d.lock.RLock()
	defer d.lock.RUnlock()
	return nil
}

// StartDiagnostics implements the core.Diagnoser interface
func (d *Driver) StartDiagnostics(w io.Writer) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.diagnostics = w
}

// Bytes implements core.FileDriver.
func (d *Driver) Bytes() []byte {
	// Note: this function holds a read-only lock, so it must not modify the driver state
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.file.Bytes()
}

// Cursor implements core.FileDriver.
func (d *Driver) Cursor() int64 {
	// Note: this function holds a read-only lock, so it must not modify the driver state
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.file.Cursor()
}

// SetContents implements core.FileDriver.
func (d *Driver) SetContents(buf []byte) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.file.SetContents(buf)
}

// ResendText implements core.FileDriver.
func (d *Driver) ResendText() bool {
	// Note: this function holds a read-only lock, so it must not modify the driver state
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.file.ResendText()
}

func (d *Driver) printf(str string, values ...interface{}) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.diagnostics != nil {
		fmt.Fprintf(d.diagnostics, str, values...)
		if !strings.HasSuffix(str, "\n") {
			fmt.Fprint(d.diagnostics, "\n")
		}
	}
	if d.debug {
		fmt.Printf(str+"\n", values...)
		if !strings.HasSuffix(str, "\n") {
			fmt.Fprint(d.diagnostics, "\n")
		}
	}
}

func (d *Driver) println(parts ...interface{}) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.diagnostics != nil {
		fmt.Fprintln(d.diagnostics, parts...)
	}
	if d.debug {
		fmt.Println(parts...)
	}
}
