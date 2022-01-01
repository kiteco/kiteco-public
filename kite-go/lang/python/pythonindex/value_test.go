package pythonindex

// TODO(naman) unused: rm unless we decide to turn local code search back on

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasePaths(t *testing.T) {
	var input, output, expected []string

	input = []string{
		"/src/project/foo.py",
		"/src/project/bar.py",
		"/src/project/module/baz.py",
	}
	output = BasePaths(input)
	expected = []string{"/src/project/"}
	assert.Equal(t, output, expected)

	input = []string{
		"/src/project/__init__.py",
		"/src/project/module/foo.py",
		"/src/project/module/bar.py",
	}
	output = BasePaths(input)
	expected = []string{"/src/"}
	assert.Equal(t, output, expected)

	input = []string{
		"/foo.py",
		"/src/project/bar.py",
		"/src/project/module/baz.py",
	}
	output = BasePaths(input)
	expected = []string{"/"}
	assert.Equal(t, output, expected)

	input = []string{
		"/__init__.py",
		"/src/project/bar.py",
		"/src/project/module/baz.py",
	}
	output = BasePaths(input)
	expected = []string{"/"}
	assert.Equal(t, output, expected)

	input = []string{
		"/src/foo/a.py",
		"/src/bar/b.py",
	}
	output = BasePaths(input)
	expected = []string{
		"/src/foo/",
		"/src/bar/",
	}
	assert.Equal(t, output, expected)

	input = []string{
		"/src/foo/__init__.py",
		"/src/bar/b.py",
	}
	output = BasePaths(input)
	expected = []string{"/src/"}
	assert.Equal(t, output, expected)
}
