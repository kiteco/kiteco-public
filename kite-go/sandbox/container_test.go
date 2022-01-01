package sandbox

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const image = "kiteco/pythonsandbox"

func TestDockerEcho(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}
	opts := &ProcessOptions{
		Command: "echo",
		Args:    []string{"abc"},
	}
	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	require.NotNil(t, container)
	defer container.Cleanup()

	stdout, stderr, err := container.Wait()
	require.NoError(t, err)

	assert.Equal(t, "abc\n", string(stdout))
	assert.Equal(t, "", string(stderr))
}

func TestDockerTimeout(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}
	opts := &ProcessOptions{
		Command: "sleep",
		Args:    []string{"10"},
		Limits:  Limits{Timeout: 100 * time.Millisecond},
	}
	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	defer container.Cleanup()

	_, _, err = container.Wait()
	assert.IsType(t, &TimeLimitExceeded{}, err)
}

func TestDockerEchoAndTimeout(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}
	cmd := "echo abc; sleep 2"
	opts := &ProcessOptions{
		Command: "bash",
		Args:    []string{"-c", cmd},
		Limits:  Limits{Timeout: 500 * time.Millisecond},
	}
	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	defer container.Cleanup()

	stdout, stderr, err := container.Wait()
	assert.IsType(t, &TimeLimitExceeded{}, err)
	assert.Equal(t, "abc\n", string(stdout))
	assert.Equal(t, "", string(stderr))
}

func TestDockerEnvironmentVars(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}
	cmd := "echo $ABC $DEF"
	opts := &ProcessOptions{
		Command:              "bash",
		Args:                 []string{"-c", cmd},
		Limits:               Limits{Timeout: 500 * time.Millisecond},
		EnvironmentVariables: map[string]string{"ABC": "abc", "DEF": "def"},
	}
	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	defer container.Cleanup()

	stdout, stderr, err := container.Wait()
	require.NoError(t, err)
	assert.Equal(t, "abc def\n", string(stdout))
	assert.Equal(t, "", string(stderr))
}

func TestDockerWorkingDirectory(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}

	cmd := "pwd"
	opts := &ProcessOptions{
		Command: "bash",
		Args:    []string{"-c", cmd},
		Limits:  Limits{Timeout: 500 * time.Millisecond},
	}

	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	defer container.Cleanup()

	stdout, stderr, err := container.Wait()
	require.NoError(t, err)
	assert.Equal(t, "/scratch", strings.TrimSpace(string(stdout)))
	assert.Equal(t, "", string(stderr))
}

func TestDockerPort(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}

	code := `
import sys
import socket

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(("", 19870))
s.listen(1)
conn, addr = s.accept()
data = conn.recv(1024)
print(data)
`
	opts := &ProcessOptions{
		Command: "python",
		Args:    []string{"-c", code},
		Port:    19870,
	}

	t.Log("Starting container...")
	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	defer container.Cleanup()

	t.Log("Getting external port...")
	endpoint, err := container.Endpoint()
	require.NoError(t, err)

	t.Log("Opening TCP connection...")
	conn, err := net.Dial("tcp", endpoint)
	require.NoError(t, err)

	t.Log("Writing to TCP connection...")
	_, err = conn.Write([]byte("test"))
	require.NoError(t, err)

	t.Log("Closing TCP connection...")
	err = conn.Close()
	require.NoError(t, err)

	t.Log("Cancelling process...")
	container.Cancel()
}

func TestDockerInputFiles(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}
	opts := &ProcessOptions{
		Command: "cat",
		Args:    []string{"foo", "/bar/baz"},
		Limits:  Limits{Timeout: 500 * time.Millisecond},
		Files: map[string][]byte{
			"foo":      []byte("abc"),
			"/bar/baz": []byte("def"),
		},
	}
	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	defer container.Cleanup()

	stdout, stderr, err := container.Wait()
	require.NoError(t, err)
	assert.Equal(t, "abcdef\n", string(stdout))
	assert.Equal(t, "", string(stderr))
}

func TestDockerOutputFiles(t *testing.T) {
	if !dockerTests {
		t.Skip("use go test --docker to run tests that require docker")
	}
	opts := &ProcessOptions{
		Command: "bash",
		Args:    []string{"-c", "echo foo > bar"},
		Limits:  Limits{Timeout: 500 * time.Millisecond},
	}
	container, err := StartContainer(image, opts)
	require.NoError(t, err)
	defer container.Cleanup()

	stdout, stderr, err := container.Wait()
	require.NoError(t, err)
	assert.Equal(t, "", string(stdout))
	assert.Equal(t, "", string(stderr))

	files, err := container.Files()
	for k, v := range files {
		t.Logf("Found output files: %s (%d bytes)", k, len(v))
	}
	require.NoError(t, err)
	file, ok := files["bar"]
	require.True(t, ok, "expected to find a file named 'bar' but there was no such file")
	assert.Equal(t, "foo\n", string(file))
}
