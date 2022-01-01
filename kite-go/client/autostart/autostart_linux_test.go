// +build linux

package autostart

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SetDisabled(t *testing.T) {
	input := []byte("")
	output := setDisabled(input, "true")
	assert.Equal(t, output, []byte("\n"+disabledKey+"true"))

	output = setDisabled(output, "false")
	assert.Equal(t, output, []byte("\n"+disabledKey+"false"))

	output = setDisabled(output, "true")
	assert.Equal(t, output, []byte("\n"+disabledKey+"true"))
}
