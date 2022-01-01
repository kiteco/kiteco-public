package sandbox

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	// The name of the python interpreter to invoke
	pythonInterpreter = "python3"
	// The name of the source file in which to store python code
	sourceFilename = "main.py"
	// The working dir for python programs within the container
	pythonScratchPath = "/scratch"
)

// Environment variables to be sent to python programs
var pythonEnvs = map[string]string{
	"PYTHONIOENCODING": "utf8", // instructs the python intepreter to encode output as utf8
}

// --

// PythonProgram represents python code and its environment
type PythonProgram struct {
	// Code is the python source to run
	Code string
	// Environment variables for this program
	EnvironmentVariables map[string]string
	// Files for this program
	SupportingFiles map[string][]byte
}

// Options for invoking the Python interpreter
func (p *PythonProgram) options(interpreter string, opts *ProgramOptions) *ProcessOptions {
	subprocessOpts := ProcessOptions{
		Files:                make(map[string][]byte),
		EnvironmentVariables: make(map[string]string),
		StandardInput:        opts.StandardInput,
		Port:                 opts.Port,
		Limits:               opts.Limits,
		WorkingDir:           pythonScratchPath,
	}

	if opts.Command != "" {
		split := strings.Split(opts.Command, " ")
		subprocessOpts.Command = split[0]
		if len(split) > 1 {
			subprocessOpts.Args = split[1:]
		}
	} else {
		subprocessOpts.Command = interpreter
		subprocessOpts.Args = []string{sourceFilename}
	}

	// Take the union of the files and environmentVariables vars from the program and the options
	subprocessOpts.Files[sourceFilename] = []byte(p.Code)
	for k, v := range opts.Files {
		subprocessOpts.Files[k] = v
	}
	for k, v := range p.SupportingFiles {
		subprocessOpts.Files[k] = v
	}
	for k, v := range opts.EnvironmentVariables {
		subprocessOpts.EnvironmentVariables[k] = v
	}
	for k, v := range p.EnvironmentVariables {
		subprocessOpts.EnvironmentVariables[k] = v
	}

	return &subprocessOpts
}

// --

// NativePythonProgram represents python code that runs inside the default system python interpreter
type NativePythonProgram struct {
	PythonProgram
}

// NewPythonProgram creates a program that runs a python script in the default python interpreter
func NewPythonProgram(code string) *NativePythonProgram {
	return &NativePythonProgram{
		PythonProgram: PythonProgram{
			Code:                 code,
			EnvironmentVariables: copyEnv(pythonEnvs),
			SupportingFiles:      make(map[string][]byte),
		},
	}
}

// Start launches the program in a subprocess
func (p *NativePythonProgram) Start(opts *ProgramOptions) (Process, error) {
	// Resolve path to python interpreter
	interpreterPath, err := exec.LookPath(pythonInterpreter)
	if err != nil {
		return nil, fmt.Errorf("could not find python interpreter: %v", err)
	}

	return StartSubprocess(p.options(interpreterPath, opts))
}

// --

// ContainerizedPythonProgram represents a block of python code that runs within a docker container
type ContainerizedPythonProgram struct {
	PythonProgram
	// Image is the name of the docker image to run the program in, or empty to use the native
	Image string
}

// NewContainerizedPythonProgram creates a program that runs a python script in a docker container
func NewContainerizedPythonProgram(code string, image string) *ContainerizedPythonProgram {
	return &ContainerizedPythonProgram{
		Image: image,
		PythonProgram: PythonProgram{
			Code:                 code,
			EnvironmentVariables: copyEnv(pythonEnvs),
			SupportingFiles:      make(map[string][]byte),
		},
	}
}

// Start launches the program in a docker container
func (p *ContainerizedPythonProgram) Start(opts *ProgramOptions) (Process, error) {
	return StartContainer(p.Image, p.options(pythonInterpreter, opts))
}
