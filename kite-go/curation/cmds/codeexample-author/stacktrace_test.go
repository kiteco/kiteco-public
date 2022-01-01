package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStacktrace(t *testing.T) {
	stacktrace := `
Traceback (most recent call last):
  File "<stdin>", line 5, in <module>
  File "<stdin>", line 3, in foo
NameError: global name 'a' is not defined
`

	err := findLastErrorForPath("<stdin>", stacktrace)
	if assert.NotNil(t, err) {
		assert.Equal(t, 4, err.Line) // line number is expected to be 0-based
		assert.Equal(t, "NameError: global name 'a' is not defined", err.Message)
	}

	assert.Nil(t, findLastErrorForPath("foo", stacktrace))
}

func TestStacktraceNegative(t *testing.T) {
	stacktrace := "xxx\nyyy"
	assert.Nil(t, findLastErrorForPath("foo", stacktrace))
}

func TestStacktraceComplicated(t *testing.T) {
	stacktrace := `
Traceback (most recent call last):
  File "foo.py", line 11, in <module>
    import matplotlib.pyplot as plt; plt.plot(range(5), {})
  File "/Applications/anaconda/lib/python3.4/site-packages/matplotlib/pyplot.py", line 3099, in plot
    ret = ax.plot(*args, **kwargs)
  File "/Applications/anaconda/lib/python3.4/site-packages/matplotlib/axes/_axes.py", line 1373, in plot
    for line in self._get_lines(*args, **kwargs):
  File "/Applications/anaconda/lib/python3.4/site-packages/matplotlib/axes/_base.py", line 304, in _grab_next_args
    for seg in self._plot_args(remaining, kwargs):
  File "/Applications/anaconda/lib/python3.4/site-packages/matplotlib/axes/_base.py", line 282, in _plot_args
    x, y = self._xy_from_xy(x, y)
  File "/Applications/anaconda/lib/python3.4/site-packages/matplotlib/axes/_base.py", line 223, in _xy_from_xy
    raise ValueError("x and y must have same first dimension")
ValueError: x and y must have same first dimension
`

	err := findLastErrorForPath("foo.py", stacktrace)
	if assert.NotNil(t, err) {
		assert.Equal(t, 10, err.Line) // line number is expected to be 0-based
		assert.Equal(t, "ValueError: x and y must have same first dimension", err.Message)
	}
}

func TestStacktraceDistractors(t *testing.T) {
	stacktrace := `
ham
spam

Traceback (most recent call last):
  File "ham.py", line 15, in <module>
  File "ham.py", line 20, in foo
<some error message>
`

	err := findLastErrorForPath("ham.py", stacktrace)
	if assert.NotNil(t, err) {
		assert.Equal(t, 14, err.Line) // line number is expected to be 0-based
		assert.Equal(t, "<some error message>", err.Message)
	}

	assert.Nil(t, findLastErrorForPath("foo", stacktrace))

}
