package sandbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimple(t *testing.T) {
	code := `print('foobar')`
	stdout, stderr, err := RunPythonCode(code, &Limits{})
	assert.NoError(t, err)
	assert.Empty(t, stderr)
	assert.Equal(t, "foobar\n", stdout)
}

func TestExitZero(t *testing.T) {
	code := `import sys; sys.exit(0)`
	stdout, stderr, err := RunPythonCode(code, &Limits{})
	assert.NoError(t, err)
	assert.Empty(t, stderr)
	assert.Empty(t, stdout)
}

func TestExitNonzero(t *testing.T) {
	code := `import sys; sys.exit(1)`
	stdout, stderr, err := RunPythonCode(code, &Limits{})
	assert.Error(t, err)
	assert.Empty(t, stderr)
	assert.Empty(t, stdout)
}

func TestHardAbort(t *testing.T) {
	code := `import os; os.abort()`
	stdout, stderr, err := RunPythonCode(code, &Limits{})
	assert.Error(t, err)
	assert.Empty(t, stderr)
	assert.Empty(t, stdout)
}

func TestTimeout(t *testing.T) {
	code := `while(True): pass`
	_, _, err := RunPythonCode(code, &Limits{Timeout: time.Millisecond})
	assert.IsType(t, &TimeLimitExceeded{}, err)
}

func TestStdoutLineLimit(t *testing.T) {
	code := `while(True): print("x")`
	_, _, err := RunPythonCode(code, &Limits{MaxLines: 10})
	require.IsType(t, &OutputLimitExceeded{}, err)
	assert.Equal(t, "stdout", err.(*OutputLimitExceeded).Stream)
}

func TestStdoutLimitWithLongTimeout(t *testing.T) {
	code := `while(True): print("x\n")`
	_, _, err := RunPythonCode(code, &Limits{MaxLines: 10, Timeout: time.Hour})
	require.IsType(t, &OutputLimitExceeded{}, err)
	assert.Equal(t, "stdout", err.(*OutputLimitExceeded).Stream)
}

func TestStderrLimit(t *testing.T) {
	code := `import sys
while True: sys.stderr.write("x\n")`
	_, _, err := RunPythonCode(code, &Limits{MaxLines: 10})
	require.IsType(t, &OutputLimitExceeded{}, err)
	assert.Equal(t, "stderr", err.(*OutputLimitExceeded).Stream)
}

func TestStderrLimitWithLongTimeout(t *testing.T) {
	code := `import sys
while True: sys.stderr.write("x\n")`
	_, _, err := RunPythonCode(code, &Limits{MaxLines: 10, Timeout: time.Hour})
	require.IsType(t, &OutputLimitExceeded{}, err)
	assert.Equal(t, "stderr", err.(*OutputLimitExceeded).Stream)
}

func TestStderrLimitOneLine(t *testing.T) {
	code := `import sys
while True: sys.stderr.write("x")`
	_, _, err := RunPythonCode(code, &Limits{MaxBytes: 20})
	require.IsType(t, &OutputLimitExceeded{}, err)
	assert.Equal(t, "stderr", err.(*OutputLimitExceeded).Stream)
}

func TestStdoutLimitOneLine(t *testing.T) {
	code := `import sys
while True: sys.stdout.write("x")`
	_, _, err := RunPythonCode(code, &Limits{MaxBytes: 20})
	require.IsType(t, &OutputLimitExceeded{}, err)
	assert.Equal(t, "stdout", err.(*OutputLimitExceeded).Stream)
}

func TestAlternatingOutput(t *testing.T) {
	code := `import sys
while True:
	for s in [sys.stdout, sys.stderr]:
		s.write('x\n')`
	_, _, err := RunPythonCode(code, &Limits{MaxLines: 10, Timeout: time.Hour})
	assert.IsType(t, &OutputLimitExceeded{}, err)
}

func TestRunProgram_Files(t *testing.T) {
	code := `print(open("spam").read())`
	program := NewPythonProgram(code)

	process, err := program.Start(&ProgramOptions{
		Files: map[string][]byte{
			"spam": []byte("shazam"),
		}})
	require.NoError(t, err)
	defer process.Cleanup()

	stdout, stderr, err := process.Wait()
	require.NoError(t, err)
	assert.Len(t, stderr, 0)
	assert.Equal(t, "shazam\n", string(stdout))
}

func TestPrintUnicode(t *testing.T) {
	code := `print('\u2713')`
	stdout, stderr, err := RunPythonCode(code, &Limits{})
	assert.NoError(t, err)
	assert.Empty(t, stderr)
	assert.Equal(t, "✓\n", stdout)
}

func TestEnvironmentVariables(t *testing.T) {
	code := `import os; print(os.environ['foo'])`
	program := NewPythonProgram(code)
	program.EnvironmentVariables["foo"] = "bar"

	stdout, stderr, err := runConsoleApparatus(program, &Limits{})
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.Equal(t, "bar\n", stdout)
}

func TestSupportingFiles(t *testing.T) {
	code := `print(open("foo").read())`
	program := NewPythonProgram(code)
	program.SupportingFiles["foo"] = []byte("bar")

	stdout, stderr, err := runConsoleApparatus(program, &Limits{})
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.Equal(t, "bar\n", stdout)
}
