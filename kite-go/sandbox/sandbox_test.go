package sandbox

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Python code for a simple HTTP server. This server is executed and probed during various unittests below.
// We use the standard library's http.server package (rather than e.g. flask) because it introduces
// the smallest number of dependencies (we do not want to install flask just to be able to run unittests).
var httpServerCode = `
import os
import sys
import time
import http.server
import socketserver


class Handler(http.server.SimpleHTTPRequestHandler):
    def handle_hello(self):
        return "Hello World!", 200

    def handle_error(self):
        return "There was trouble.", 500

    def handle_echo(self, body):
        return body, 200

    def handle_exit(self):
        print("xyz")
        sys.stdout.flush()
        os.abort()   # cannot use sys.exit here

    def handle_hang(self):
        print("xyz")
        sys.stdout.flush()
        time.sleep(1000000)
        return "", 200

    def do_GET(self):
        if self.path == '/hello':
            data, status = self.handle_hello()
        elif self.path == '/error':
            data, status = self.handle_error()
        elif self.path == '/echo':
            data, status = self.handle_echo()
        elif self.path == '/exit':
            data, status = self.handle_exit()
        elif self.path == '/hang':
            data, status = self.handle_hang()
        else:
            data = "Path not found"
            status = 404

        self.send_response(status)
        self.end_headers()
        self.wfile.write(data.encode())

    def do_POST(self):
        length = int(self.headers['Content-Length'])
        body = self.rfile.read(length)

        if self.path == '/echo':
            data, status = self.handle_echo(body)
        else:
            data = "Path not found"
            status = 404

        self.send_response(status)
        self.end_headers()
        self.wfile.write(data)


if __name__ == '__main__':
    port = int(os.environ.get("PORT", "8000"))
    httpd = socketserver.TCPServer(('', port), Handler)
    httpd.allow_reuse_address = True
    httpd.serve_forever()
`

func NewHTTPApparatus(request *http.Request, limits *Limits) *Apparatus {
	port, _ := UnusedPort()
	action := &DoThenCancel{&HTTPAction{request}}
	return NewApparatus(action, limits, port, "")
}

func TestHTTPApparatusGetRequest(t *testing.T) {
	program := NewPythonProgram(httpServerCode)

	request, err := http.NewRequest("GET", "http://localhost/hello", nil)
	require.NoError(t, err)

	apparatus := NewHTTPApparatus(request, &Limits{})

	result, err := apparatus.Run(program)
	if result != nil && !result.Succeeded {
		t.Log("Python said:\n" + string(result.Stderr))
	}

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Succeeded)
	require.Len(t, result.HTTPOutputs, 1)

	httpOutput := result.HTTPOutputs[0]
	assert.Equal(t, http.StatusOK, httpOutput.Response.StatusCode)
	assert.Equal(t, "Hello World!", string(httpOutput.ResponseBody))
}

func TestHTTPApparatusErrorStatus(t *testing.T) {
	program := NewPythonProgram(httpServerCode)

	request, err := http.NewRequest("GET", "http://localhost/error", nil)
	require.NoError(t, err)

	apparatus := NewHTTPApparatus(request, &Limits{})
	result, err := apparatus.Run(program)
	if result != nil && !result.Succeeded {
		t.Log("Python said:\n" + string(result.Stderr))
	}

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Succeeded)
	require.Len(t, result.HTTPOutputs, 1)

	httpOutput := result.HTTPOutputs[0]
	assert.Equal(t, http.StatusInternalServerError, httpOutput.Response.StatusCode)
	assert.Equal(t, "There was trouble.", string(httpOutput.ResponseBody))
}

func TestHTTPApparatusPost(t *testing.T) {
	program := NewPythonProgram(httpServerCode)

	body := "abc"
	request, err := http.NewRequest("POST", "http://localhost/echo", bytes.NewBuffer([]byte(body)))
	require.NoError(t, err)

	apparatus := NewHTTPApparatus(request, &Limits{})
	result, err := apparatus.Run(program)
	if result != nil && !result.Succeeded {
		t.Log("Python said:\n" + string(result.Stderr))
	}

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Succeeded)
	require.Len(t, result.HTTPOutputs, 1)

	httpOutput := result.HTTPOutputs[0]
	assert.Equal(t, http.StatusOK, httpOutput.Response.StatusCode)
	assert.Equal(t, body, string(httpOutput.ResponseBody))
}

func TestHTTPApparatusTimeout(t *testing.T) {
	t.Skip("skipping since this test is flaky")
	program := NewPythonProgram(httpServerCode)

	request, err := http.NewRequest("GET", "http://localhost/hang", nil)
	require.NoError(t, err)

	apparatus := NewHTTPApparatus(request, &Limits{Timeout: 200 * time.Millisecond})
	result, err := apparatus.Run(program)
	assert.IsType(t, &TimeLimitExceeded{}, result.SandboxError)
	assert.NotNil(t, result)
	assert.Equal(t, "xyz\n", string(result.Stdout))
}

// This server waits one second before opening a port, which is used in tests below.
var delayedStartServerCode = `
import os
import time
import http.server
import socketserver

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'abc')

time.sleep(1)
port = int(os.environ.get("PORT", "8000"))
httpd = socketserver.TCPServer(('', port), Handler)
httpd.allow_reuse_address = True
httpd.serve_forever()
`

func TestHTTPApparatusRetry(t *testing.T) {
	// In this test, the python code does not open a TCP port for 1 second. This exercises the retry behavior
	// of HTTPApparatus, which should eventually make a connection to the server.
	program := NewPythonProgram(delayedStartServerCode)

	request, err := http.NewRequest("GET", "http://localhost/", nil)
	require.NoError(t, err)

	apparatus := NewHTTPApparatus(request, &Limits{})
	result, err := apparatus.Run(program)
	if result != nil && !result.Succeeded {
		t.Log("Python said:\n" + string(result.Stderr))
	}

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Succeeded)
	require.Len(t, result.HTTPOutputs, 1)

	httpOutput := result.HTTPOutputs[0]
	require.NotNil(t, httpOutput)
	require.NotNil(t, httpOutput.Response)
	assert.Equal(t, http.StatusOK, httpOutput.Response.StatusCode)
	assert.Equal(t, "abc", string(httpOutput.ResponseBody))
}

func TestHTTPApparatusRetryTimout(t *testing.T) {
	// In this test, the python code does not open a TCP port for 1 second, but the apparatus timeout is 0.5 seconds.
	// This exercises the retry behavior of HTTPApparatus, which should timeout and return an error.
	program := NewPythonProgram(delayedStartServerCode)

	request, err := http.NewRequest("GET", "http://localhost/", nil)
	require.NoError(t, err)

	apparatus := NewHTTPApparatus(request, &Limits{
		Timeout: 100 * time.Millisecond,
	})
	result, err := apparatus.Run(program)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Succeeded)

	assert.Error(t, result.SandboxError, "expected a timeout")
	assert.IsType(t, &TimeLimitExceeded{}, result.SandboxError)
}
