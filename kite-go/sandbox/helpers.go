package sandbox

func runConsoleApparatus(prog Program, limit *Limits) (string, string, error) {
	// Create the apparatus
	apparatus := NewApparatus(nil, limit, 0, "")

	// Run the program in the apparatus
	result, err := apparatus.Run(prog)
	if err != nil {
		return "", "", err
	}

	// Collect the output
	var stdout, stderr string
	if result != nil {
		stdout = string(result.Stdout)
		stderr = string(result.Stderr)
		err = result.SandboxError
	}
	return stdout, stderr, err
}

// RunPythonCode runs a python script with time and output limits
func RunPythonCode(code string, limit *Limits) (string, string, error) {
	prog := NewPythonProgram(code)
	return runConsoleApparatus(prog, limit)
}

// RunPythonCodeContainerized runs a python script in a docker container with time and output limits
func RunPythonCodeContainerized(code string, image string, limit *Limits) (string, string, error) {
	prog := NewContainerizedPythonProgram(code, image)
	return runConsoleApparatus(prog, limit)
}

// RunCommand execute a commands and returns the output
func RunCommand(stdin []byte, cmd string, args ...string) (string, string, error) {
	process, err := StartSubprocess(&ProcessOptions{
		Command:       cmd,
		Args:          args,
		StandardInput: stdin,
		Limits:        *DefaultLimits,
	})
	if err != nil {
		return "", "", err
	}
	defer process.Cleanup()

	stdout, stderr, err := process.Wait()
	return string(stdout), string(stderr), err
}

func copyEnv(in map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = v
	}
	return out
}
