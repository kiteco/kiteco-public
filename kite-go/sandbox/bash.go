package sandbox

const bashFilename = "main.sh"

// ContainerizedBashProgram represents a block of python code that runs
// in a Docker container.
type ContainerizedBashProgram struct {
	// Bash source to run
	Code string
	// Name of the docker image in which to run the program
	Image string
	// Environment variables for this program
	EnvironmentVariables map[string]string
	// Files used by this program
	SupportingFiles map[string][]byte
}

// NewContainerizedBashProgram constructs a new program containerized bash. ;)
func NewContainerizedBashProgram(code string, image string) *ContainerizedBashProgram {
	return &ContainerizedBashProgram{
		Image:                image,
		Code:                 code,
		EnvironmentVariables: make(map[string]string),
		SupportingFiles:      make(map[string][]byte),
	}
}

func (p *ContainerizedBashProgram) options(shell string, opts *ProgramOptions) *ProcessOptions {
	subprocessOpts := ProcessOptions{
		Command:              shell,
		Args:                 []string{bashFilename},
		Files:                make(map[string][]byte),
		EnvironmentVariables: make(map[string]string),
	}

	subprocessOpts.Files[bashFilename] = []byte(p.Code)
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

// Start launches the program in a Docker container
func (p *ContainerizedBashProgram) Start(opts *ProgramOptions) (Process, error) {
	return StartContainer(p.Image, p.options("bash", opts))
}
