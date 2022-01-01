package metrics

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/stretchr/testify/assert"
)

func TestProSelected_ReadAndFlatten(t *testing.T) {
	out := make(map[string]interface{})
	met := NewSmartSelectedMetrics()

	var tests = []struct {
		editor  data.Editor
		nprosel int
	}{
		{
			editor:  data.VimEditor,
			nprosel: 10,
		},
		{
			editor:  data.VSCodeEditor,
			nprosel: 20,
		},
		{
			editor:  data.SublimeEditor,
			nprosel: 35,
		},
	}

	for _, test := range tests {
		for i := 0; i < test.nprosel; i++ {
			met.OnComplSelect(data.RCompletion{Smart: true}, test.editor)
		}
	}

	met.ReadAndFlatten(false, out)
	for _, test := range tests {
		assert.EqualValues(t, test.nprosel, out[smartSelectedKey(test.editor)], "Metrics out should match number pro selected")
	}

	// Clears for subsequent set of tests
	out = met.ReadAndFlatten(true, nil)
	assert.NotNil(t, out, "ReadAndFlatten should create a map when passed out=nil")
	for _, test := range tests {
		assert.EqualValues(t, test.nprosel, out[smartSelectedKey(test.editor)], "Metrics must not be cleared when passed clear=false")
	}

	out = met.ReadAndFlatten(false, nil)
	for _, test := range tests {
		assert.Nil(t, out[smartSelectedKey(test.editor)], "Metrics must be cleared when passed clear=true")
	}
}
