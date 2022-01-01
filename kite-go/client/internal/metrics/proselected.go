package metrics

import (
	"sync"

	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

type smartSelectedByEditor map[data.Editor]uint

func (p smartSelectedByEditor) flatten(out map[string]interface{}) map[string]interface{} {
	for editor, val := range p {
		// save space by not writing 0 values
		if val != 0 {
			out[smartSelectedKey(editor)] = val
		}
	}

	return out
}

func smartSelectedKey(editor data.Editor) string {
	return "smart_selected_" + editor.String()
}

// SmartSelectedMetrics holds a lock on smartSelectedByEditor
type SmartSelectedMetrics struct {
	sync.Mutex
	store smartSelectedByEditor
}

// NewSmartSelectedMetrics creates a SmartSelectedMetrics
func NewSmartSelectedMetrics() *SmartSelectedMetrics {
	return &SmartSelectedMetrics{store: make(smartSelectedByEditor)}
}

// Initialize implements component.Initializer in completions.onComplSelecter
func (c *SmartSelectedMetrics) Initialize(opts component.InitializerOptions) {
	return
}

// OnComplSelect implements completions.OnComplSelect
func (c *SmartSelectedMetrics) OnComplSelect(compl data.RCompletion, editor data.Editor) {
	if !compl.Smart {
		return
	}
	c.Lock()
	defer c.Unlock()
	c.store[editor]++
}

// ReadAndFlatten metrics to send
func (c *SmartSelectedMetrics) ReadAndFlatten(clear bool, out map[string]interface{}) map[string]interface{} {
	if out == nil {
		out = make(map[string]interface{})
	}

	return c.read(clear).flatten(out)
}

func (c *SmartSelectedMetrics) read(clear bool) smartSelectedByEditor {
	c.Lock()
	defer c.Unlock()

	out := make(smartSelectedByEditor)
	if clear {
		c.store, out = out, c.store
	} else {
		for k, v := range c.store {
			out[k] = v
		}
	}

	return out
}
