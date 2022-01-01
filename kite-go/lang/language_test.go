package lang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromFilename(t *testing.T) {
	assert.Equal(t, Golang, FromFilename("test.go"))
	assert.Equal(t, Python, FromFilename("test.py"))
	assert.Equal(t, JavaScript, FromFilename("test.js"))
	assert.Equal(t, Unknown, FromFilename("test.wtf"))
}

func TestFromFilenameAndContent(t *testing.T) {
	assert.Equal(t, Golang, FromFilenameAndContent("test.go", ""))
	assert.Equal(t, Golang, FromFilenameAndContent("test.go", "#!/bin/python"))

	assert.Equal(t, Python, FromFilenameAndContent("test.py", ""))
	assert.Equal(t, Python, FromFilenameAndContent("test.py", "#!/bin/python"))
	assert.Equal(t, Python, FromFilenameAndContent("test.py", "#!/bin/python\n===\n==="))
	assert.Equal(t, Python, FromFilenameAndContent("test", "#!/bin/python"))
	assert.Equal(t, Python, FromFilenameAndContent("test", "#!/bin/python\n===\n==="))
	assert.Equal(t, Python, FromFilenameAndContent("test.wtf", "#!/bin/python"))
	assert.Equal(t, Python, FromFilenameAndContent("test.wtf", "#!/bin/python\n===\n==="))
}

func TestExtsContainsExt(t *testing.T) {
	for l := range LanguageTags {
		assert.Contains(t, l.Extensions(), l.Extension())
	}
}
