package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// An OutputFile represents the contents of a file created by a code example
type OutputFile struct {
	// Path is relative to the working directory in which the code example was executed
	Path string
	// Contents of the file
	Contents []byte
}

// An HTTPOutput represents an HTTP request sent to a code example and the response received
type HTTPOutput struct {
	Request      *http.Request
	RequestBody  []byte
	Response     *http.Response
	ResponseBody []byte
}

// A Result represents the output produced by a program
type Result struct {
	Stdin        []byte
	Stdout       []byte
	Stderr       []byte
	SandboxError error
	Succeeded    bool
	OutputFiles  []OutputFile
	HTTPOutputs  []HTTPOutput
}

// File gets the output file at the given path, or returns nil if there was no such output file
func (r *Result) File(path string) *OutputFile {
	for _, file := range r.OutputFiles {
		if file.Path == path {
			return &file
		}
	}
	return nil
}

// Apparatus runs commands and captures standard output and standard error.
type Apparatus struct {
	// Limits describes time and output limits
	limits *Limits
	// Action is something to perform while the program is running
	action Action
	// Files is a set of files to place in the working directory when running the code
	files map[string][]byte
	// Port that this apparatus will communicate on, or zero if no port is required
	port int
	// Command to execute
	command string
}

// NewApparatus constructs an aparatus with the given action and limits
func NewApparatus(action Action, limits *Limits, port int, command string) *Apparatus {
	return &Apparatus{
		action:  action,
		limits:  limits,
		files:   make(map[string][]byte),
		port:    port,
		command: command,
	}
}

// AddFile adds a file relative to the working directory in which the program will run
func (s *Apparatus) AddFile(path string, contents []byte) {
	s.files[path] = contents
}

// FileExists checks if the given file already exists in the folder
func (s *Apparatus) FileExists(path string) bool {
	_, ok := s.files[path]
	return ok
}

// File returns the contents of the named file
func (s *Apparatus) File(path string) []byte {
	return s.files[path]
}

// Action returns the action for this apparatus
func (s *Apparatus) Action() Action {
	return s.action
}

// SetLimits sets the limits on the process to run with this apparatus.
func (s *Apparatus) SetLimits(limits *Limits) {
	s.limits = limits
}

// Run executes the given program in this apparatus, and returns the program's console output
// There are two fundamentally different types of errors: ones that prevented the sandbox
// environment from starting or monitoring the code example (e.g. disk is full and the temporary
// file could not be created) and ones that resulted from a bug in the code example itself,
// such as exiting with a nonzero exit status, or exceeding a timeout.
func (s *Apparatus) Run(prog Program) (*Result, error) {
	// Start the timer. This is a time limit on the entire execution, including creating
	// the docker container, launching the python interpreter, waiting for the HTTP
	// request (and possibly retrying some number of times), and collecting the result.
	timeout := s.limits.makeTimeoutChannel()

	// Start the process
	process, err := prog.Start(&ProgramOptions{
		Limits:               *s.limits,
		Port:                 s.port,
		EnvironmentVariables: map[string]string{"PORT": strconv.Itoa(s.port)},
		Files:                s.files,
		Command:              s.command,
	})
	if err != nil {
		return nil, fmt.Errorf("error starting python program: %v", err)
	}
	// Cleaning up the process can take a while (e.g. for docker containers)
	// so run it in a goroutine and return
	defer func() { go process.Cleanup() }()

	// Run the action
	var result Result
	var actionError error
	if s.action != nil {
		actionError = s.action.Do(process, &result, timeout)
	}

	// Cancel or wait for the process
	if actionError == nil {
		// If the action succeeded (or there was no action) then wait for the process to complete
		result.Stdout, result.Stderr, result.SandboxError = process.Wait()
		result.Succeeded = result.SandboxError == nil
	} else {
		// If the action failed then cancel the process (but first give it a few moments to finish
		// writing helpful diagnostic output to stdout/stderr).
		time.Sleep(10 * time.Millisecond)
		result.Stdout, result.Stderr = process.Cancel()
		if actionError == errShouldTerminate {
			result.Succeeded = true
		} else { // some other non-nil actionError
			result.SandboxError = actionError
			result.Succeeded = false
		}
	}

	files, err := process.Files()
	if err != nil {
		log.Printf("Error collecting output files: %v\n", err)
	}
	for k, v := range files {
		result.OutputFiles = append(result.OutputFiles, OutputFile{k, v})
	}

	return &result, nil
}

// --

var errShouldTerminate = errors.New("action completed")

// Action represents something that can be performed while a code example is running
type Action interface {
	// Do performs the given action on the process
	Do(process Process, result *Result, timeout <-chan time.Time) error
}

// DoThenCancel executes some other action then cancels the process
type DoThenCancel struct {
	Action Action
}

// Do executes some other action then cancels the process
func (a *DoThenCancel) Do(process Process, result *Result, timeout <-chan time.Time) error {
	err := a.Action.Do(process, result, timeout)
	if err != nil {
		return err
	}
	return errShouldTerminate
}

// --

// InParallel is an action that performs a set of actions in parallel goroutines
type InParallel []Action

// Do starts each action in a goroutine and waits for them all to complete
func (a InParallel) Do(process Process, result *Result, timeout <-chan time.Time) error {
	var finalError error
	var wg sync.WaitGroup
	for _, action := range a {
		wg.Add(1)
		go func(a Action) {
			defer wg.Done()
			err := a.Do(process, result, timeout)
			if finalError == nil && err != nil {
				finalError = err
			}
		}(action)
	}
	wg.Wait()
	return finalError
}

// InSequence is an action that performs a set of actions in sequence
type InSequence []Action

// Do executes each action one-by-one
func (a InSequence) Do(process Process, result *Result, timeout <-chan time.Time) error {
	for _, action := range a {
		err := action.Do(process, result, timeout)
		if err != nil {
			return err
		}
	}
	return nil
}

// --

// HTTPAction sends HTTP requests to programs and returns the HTTP response
type HTTPAction struct {
	Request *http.Request
}

// NewHTTPAction constructs a web apparatus
func NewHTTPAction(request *http.Request) *HTTPAction {
	return &HTTPAction{
		Request: request,
	}
}

func (s *HTTPAction) communicate(timeout <-chan time.Time) (*http.Response, error) {
	// Create the dialer. We are dialing localhost so set a very short timeout.
	// This timeout is the time to wait before giving up on a single HTTP request.
	// Note that the timeout channel passed into this function is a timeout on
	// the entire execution of a code example (including zero or more HTTP requests).
	dialer := &net.Dialer{
		Timeout: 50 * time.Millisecond,
	}
	transport := &http.Transport{
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 50 * time.Millisecond,
	}

	// Set up channel for returning the response
	type responseAndError struct {
		response *http.Response
		err      error
	}
	responseChan := make(chan responseAndError, 1)

	// Start HTTP communication with the process
	go func() {
		response, err := transport.RoundTrip(s.Request)
		responseChan <- responseAndError{response, err}
	}()

	// Wait for either the response to complete or the timeout to expire
	select {
	case <-timeout:
		log.Println("Cancelling request after HTTP request timed out")
		transport.CancelRequest(s.Request)
		return nil, &TimeLimitExceeded{}

	case r := <-responseChan:
		return r.response, r.err
	}
}

// Do sends an HTTP request and waits for the response
func (s *HTTPAction) Do(process Process, result *Result, timeout <-chan time.Time) error {
	// Get the externally accessible port
	endpoint, err := process.Endpoint()
	if err != nil {
		return err
	}

	// Read the body once so that retries work as expected. Otherwise, the body will be
	// consumed on the first read and then will be empty for subsequent retries.
	var body []byte
	if s.Request.Body != nil {
		body, err = ioutil.ReadAll(s.Request.Body)
		if err != nil {
			return err
		}
	}

	// Construct request
	s.Request.URL.Host = endpoint
	log.Printf("HTTP apparatus will dial %s\n", s.Request.URL.Host)

	var subprocessError error
	var response *http.Response
	var responseBody []byte

	for {
		// Attempt to communicate with the server
		if body != nil {
			s.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}
		response, err = s.communicate(timeout)
		if err != nil {
			// If the error was a connection refused then sleep and try again. For all other
			// errors, return failure.
			if _, ok := err.(*net.OpError); ok {
				log.Println("Request timed out so will wait then retry.")
				time.Sleep(100 * time.Millisecond)
				continue
			} else if err.Error() == "EOF" {
				log.Println("Received EOF so will wait then retry.")
				time.Sleep(100 * time.Millisecond)
				continue
			} else if err.Error() == "http: can't write HTTP request on broken connection" {
				log.Println("Broken connection so will wait then retry.")
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}

		log.Println("Successfully communicated with server, reading response body...")

		// Read the contents of the response body before exiting
		responseBody, err = ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("Received an error via responseChan:", err)
			if subprocessError == nil {
				subprocessError = err
			}
		}
		log.Println("Successfully read response body.")

		err = response.Body.Close()
		if err != nil {
			log.Println("Error closing HTTP body stream:", err)
			if subprocessError == nil {
				subprocessError = err
			}
		}

		result.HTTPOutputs = append(result.HTTPOutputs, HTTPOutput{
			Request:      s.Request,
			Response:     response,
			ResponseBody: responseBody,
		})

		break
	}
	return nil
}
