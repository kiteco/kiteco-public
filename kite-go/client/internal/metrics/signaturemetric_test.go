package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Metrics(t *testing.T) {
	s := &SignaturesMetric{}
	assert.EqualValues(t, 0, s.Read().Triggered)
	assert.EqualValues(t, 0, s.Read().Shown)

	s.SignatureRequested(false)
	assert.EqualValues(t, 1, s.Read().Triggered)
	assert.EqualValues(t, 0, s.Read().Shown)

	s.SignatureRequested(true)
	assert.EqualValues(t, 2, s.Read().Triggered)
	assert.EqualValues(t, 1, s.Read().Shown)

	snap := s.ReadAndClear()
	assert.EqualValues(t, 2, snap.Triggered)
	assert.EqualValues(t, 1, snap.Shown)
	assert.EqualValues(t, 0, s.Read().Triggered)
	assert.EqualValues(t, 0, s.Read().Shown)
}
