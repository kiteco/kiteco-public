package sandbox

import "io"

// ProgramOptions represents options that determine how a program is executed
type ProgramOptions struct {
	// EnvironmentVariables is a map of environment variables to pass to the program
	EnvironmentVariables map[string]string
	// StandardInput will be written to the program's standard input stream
	StandardInput []byte
	// Port is the port number on which apparatuses will expect to communicate with the program
	Port int
	// Limits defines time and output limits for program execution
	Limits Limits
	// Files is a map where keys are filenames and values are the contents of files
	// that will be placed in the working directory of the program when it is started.
	Files map[string][]byte
	// The command to execute in the sandbox
	Command string
}

// Program represents an executable process with a cleanup operation.
type Program interface {
	// Start creates an instance of the program
	Start(options *ProgramOptions) (Process, error)
}

// A Process is a handle to a program that is currently executing.
type Process interface {
	// Wait blocks the program exits, then returns the output generated
	Wait() ([]byte, []byte, error)
	// RescindableWait blocks until either the program exits or any message is received on the given channel.
	// If a message on the rescind channel terminates the program's execution then that object is returned
	// as the third output parameter of this function.
	RescindableWait(rescind chan error) ([]byte, []byte, error)
	// Cancel terminates the underlying subprocess and returns output generated so far
	Cancel() ([]byte, []byte)
	// Endpoint returns the host:port string that can be used to communicate with the program. This
	// may differ from the port passed to Start()
	Endpoint() (string, error)
	// Files gets the files in the container's working directory. It can be called at any point after
	// a process is created until Cleanup() is called.
	Files() (map[string][]byte, error)
	// Cleanup deletes any temporary resources used by the process
	Cleanup()
}

// ProcessOptions encapsulates the parameters used to start a process
type ProcessOptions struct {
	// Command is the path to the executable to run
	Command string
	// Args is the command line arguments
	Args []string
	// EnvironmentVariables is an environment variable
	EnvironmentVariables map[string]string
	// Port is a port to be exposed to the host
	Port int
	// Image is the name of a docker image from which to construct the container
	Image string
	// StandardInput is the input to pass to the command
	StandardInput []byte
	// Limits defines time and output limits
	Limits Limits
	// Files is a map from paths to contents to add to the container
	Files map[string][]byte
	// FileReader reads a tar archive containing files to add to the container
	FileReader io.Reader
	// WorkingDir is the working dir within the container
	WorkingDir string
}
