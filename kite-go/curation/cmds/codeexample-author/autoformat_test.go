package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This is a mock of autopep8 - used to avoid external dependencies during testing
func mockAutoformatter(code string) (string, error) {
	return "def foo():\n    x = y\n", nil
}

func TestAutoformatOneSegment(t *testing.T) {
	// This is a mock of autopep8 - used to avoid external dependencies during testing
	mockAutoformatter := func(code string) (string, error) {
		return "def foo():\n    x = y\n", nil
	}

	code := "def  foo ():\n x=y\n"
	expected := "def foo():\n    x = y\n"

	formatted, err := autoformatPythonSegmentsCustom(mockAutoformatter, code)
	if assert.NoError(t, err) && assert.Len(t, formatted, 1) {
		assert.Equal(t, expected, formatted[0])
	}
}

func TestAutoformatTwoSegments(t *testing.T) {
	// This is a mock of autopep8 - used to avoid external dependencies during testing
	mockAutoformatter := func(code string) (string, error) {
		return "def foo():\n\n    # ~~~ SENTINEL ~~~\n    x = y\n", nil
	}

	prelude := "def  foo ():\n"
	code := " x=y\n"
	expectedPrelude := "def foo():\n"
	expectedCode := "    x = y\n"

	formatted, err := autoformatPythonSegmentsCustom(mockAutoformatter, prelude, code)
	if assert.NoError(t, err) && assert.Len(t, formatted, 2) {
		assert.Equal(t, expectedPrelude, formatted[0])
		assert.Equal(t, expectedCode, formatted[1])
	}
}
