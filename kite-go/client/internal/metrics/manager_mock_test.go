package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MockManager(t *testing.T) {
	m := NewMockManager()

	m.SetRegion("my region")
	assert.EqualValues(t, "my region", m.GetRegion())

	assert.False(t, m.IsMenubarVisible())
	m.SetMenubarVisible(true)
	assert.True(t, m.IsMenubarVisible())
}
