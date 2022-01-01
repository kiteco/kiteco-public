package predict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPartialRun_ReserveRelease(t *testing.T) {
	hp := HParams{NumPredictionSlots: 20}
	pr := newPartialRunManager(nil, hp)
	assert.Equal(t, 20, pr.SlotsRemaining())
	assert.True(t, pr.Reserve(15))
	assert.False(t, pr.Reserve(10))
	assert.True(t, pr.Reserve(5))
	assert.Equal(t, 0, pr.SlotsRemaining())
	assert.True(t, pr.Release(5))
	assert.Equal(t, 5, pr.SlotsRemaining())
	assert.True(t, pr.Reserve(2))
	assert.False(t, pr.Reserve(10))
	assert.True(t, pr.Reserve(3))
	assert.Equal(t, 0, pr.SlotsRemaining())
	assert.False(t, pr.Release(21))
}

func TestPartialRun_MatchAndReserve(t *testing.T) {
	hp := HParams{NumPredictionSlots: 20, ContextSize: 4}
	search := SearchConfig{Window: 2}

	pr := PartialRunModel{
		hp:  hp,
		mgr: newPartialRunManager(nil, hp),
	}

	pr.initialContext = []int64{1, 2}

	_, _, match := pr.MatchAndReserve([]int64{1, 2}, 1, search, 0)

	assert.True(t, match)
	pr.mgr.Release(1)

	_, _, match = pr.MatchAndReserve([]int64{1, 2}, 3, search, 0)
	assert.False(t, match)

}
