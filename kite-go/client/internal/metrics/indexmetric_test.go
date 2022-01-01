package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	m := IndexMetric{}
	var stats IndexSnapshot

	stats = m.Read()
	assert.Equal(t, 0, stats.EventsWithIndex)
	assert.Equal(t, 0, stats.EventsWithoutIndex)

	m.EventHandled(true)
	m.EventHandled(false)

	stats = m.Read()
	assert.Equal(t, 1, stats.EventsWithIndex)
	assert.Equal(t, 1, stats.EventsWithoutIndex)
	stats = m.Read()
	assert.Equal(t, 1, stats.EventsWithIndex)
	assert.Equal(t, 1, stats.EventsWithoutIndex)

	m.EventHandled(true)
	m.EventHandled(false)

	stats = m.Read()
	assert.Equal(t, 2, stats.EventsWithIndex)
	assert.Equal(t, 2, stats.EventsWithoutIndex)
	stats = m.Read()
	assert.Equal(t, 2, stats.EventsWithIndex)
	assert.Equal(t, 2, stats.EventsWithoutIndex)
}

func TestReadAndClear(t *testing.T) {
	m := IndexMetric{}
	var stats IndexSnapshot

	stats = m.ReadAndClear()
	assert.Equal(t, 0, stats.EventsWithIndex)
	assert.Equal(t, 0, stats.EventsWithoutIndex)

	m.EventHandled(true)
	m.EventHandled(false)

	stats = m.ReadAndClear()
	assert.Equal(t, 1, stats.EventsWithIndex)
	assert.Equal(t, 1, stats.EventsWithoutIndex)
	stats = m.ReadAndClear()
	assert.Equal(t, 0, stats.EventsWithIndex)
	assert.Equal(t, 0, stats.EventsWithoutIndex)

	m.EventHandled(true)
	m.EventHandled(false)

	stats = m.ReadAndClear()
	assert.Equal(t, 1, stats.EventsWithIndex)
	assert.Equal(t, 1, stats.EventsWithoutIndex)
	stats = m.ReadAndClear()
	assert.Equal(t, 0, stats.EventsWithIndex)
	assert.Equal(t, 0, stats.EventsWithoutIndex)
}
