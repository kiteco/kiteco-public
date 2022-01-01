package unsupported

import (
	"fmt"
	"path"

	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/diff"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Driver implements core.FileDriver
type Driver struct {
	file   core.FileDriver
	output []interface{}
}

// NewDriver creates a new driver for unsupported files
func NewDriver(filename string) *Driver {
	return &Driver{
		file: diff.NewBufferDriver(),
	}
}

// HandleEvent implements lang.Driver.
func (d *Driver) HandleEvent(ctx kitectx.Context, evt *event.Event) string {
	ctx.CheckAbort()

	state := d.file.HandleEvent(ctx, evt)
	name := evt.GetFilename()
	resp := &response.Root{
		Type: response.Awareness,
		Description: fmt.Sprintf("No results for %s shown because Kite currently only supports Python.",
			path.Base(name)),
	}
	if event.IsEditor(evt) {
		resp.Editor = evt.GetSource()
	}
	d.output = append(d.output, resp)
	return state
}

// CollectOutput implements lang.Driver.  If the current file is active,
// we return a response.Awareness response.
func (d *Driver) CollectOutput() []interface{} {
	r := d.output
	d.output = d.output[:0]
	return r
}

// Bytes implements core.FileDriver
func (d *Driver) Bytes() []byte {
	return d.file.Bytes()
}

// Cursor implements core.FileDriver
func (d *Driver) Cursor() int64 {
	return d.file.Cursor()
}

// SetContents implements core.FileDriver.
func (d *Driver) SetContents(buf []byte) {
	d.file.SetContents(buf)
}

// ResendText implements core.FileDriver.
func (d *Driver) ResendText() bool {
	return d.file.ResendText()
}
