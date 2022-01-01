package status

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeferRecord(t *testing.T) {
	d := NewSection("foo").SampleDuration("bar")
	d.SetSampleRate(1.0)

	// to avoid having tests that are sensitive to wall clock time...
	d.DeferRecord(time.Now().Add(-200 * time.Millisecond))

	// we expect some variance due to function overhead
	expected := int64(200 * time.Millisecond)
	actual := d.Values()[0]
	delta := float64(6 * time.Millisecond)
	assert.InDelta(t, expected, actual, delta)
}
