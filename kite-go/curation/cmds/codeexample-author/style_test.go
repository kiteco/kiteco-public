package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStyleOutput(t *testing.T) {
	output := `/tmp/src.py:12: [C0111(missing-docstring), ] Missing module docstring`
	violations := parseLinterOutput(output)
	if assert.Len(t, violations, 1) {
		assert.Equal(t, 11, violations[0].Line) // 11 not 12 because lines in string are 1-based
		assert.Equal(t, "C0111", violations[0].RuleCode)
		assert.Equal(t, "missing-docstring", violations[0].RuleKey)
		assert.Equal(t, "Missing module docstring", violations[0].Message)
	}
}
