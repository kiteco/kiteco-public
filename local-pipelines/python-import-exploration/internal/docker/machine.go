package docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Status of a docker machine
type Status string

const (
	// Unknown docker status
	Unknown = "unknown"
	// Running docker machine
	Running = "running"
	// Stopped docker machine
	Stopped = "stopped"
	// Nonexistent docker machine
	Nonexistent = "nonexistent"
)

// Machine represents a docker machine running on the local machine,
// the interactions with the docker client are carried out via
// shelling out to the command line.
type Machine struct {
	name string   // name of the machine
	cert string   // cert file for the machine
	env  []string // env variables for the machine
}

// NewMachine with the specified name and certifications, does not start the machine.
func NewMachine(name, cert string) *Machine {
	return &Machine{
		name: name,
		cert: cert,
	}
}

// Status returns the status of the docker machine.
func (d *Machine) Status() (Status, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("docker-machine", "status", d.name)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Unknown, fmt.Errorf("error getting status of `%s`: %v, stdout:\n%s\n, stderr:\n%s",
			d.name, err, stdout.String(), stderr.String())
	}

	out := stdout.String()
	switch {
	case strings.Contains(out, "Running"):
		return Running, nil
	case strings.Contains(out, "Stopped"):
		return Stopped, nil
	case strings.Contains(out, "does not exist"):
		return Nonexistent, nil
	default:
		return Unknown, nil
	}
}

// Start the docker machine, returns true if the machine was already running.
// Not safe for concurrent use on multiple go routines.
func (d *Machine) Start() error {
	err := shellOut("docker-machine", "start", d.name)
	if err != nil {
		return fmt.Errorf("error starting docker machine `%s`: %v", d.name, err)
	}
	return nil
}

// SetEnv sets up the environment for the machine so that we can run commands.
// Not safe for concurrent use on multiple go routines.
func (d *Machine) SetEnv() error {
	// get the ip for the docker image
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("docker-machine", "ip", d.name)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running `docker-machine ip %s`: %v", d.name, err)
	}

	ip := strings.TrimSpace(stdout.String())

	d.env = []string{
		"DOCKER_TLS_VERIFY=1",
		fmt.Sprintf(`DOCKER_HOST=tcp://%s:2376`, ip),
		fmt.Sprintf(`DOCKER_CERT_PATH=%s`, d.cert),
		fmt.Sprintf(`DOCKER_MACHINE_NAME=%s`, d.name),
	}

	return nil
}

// Stop the docker machine
// Not safe for concurrent use on multiple go routines.
func (d *Machine) Stop() error {
	// reset the environment
	d.env = nil

	// stop the machine
	err := shellOut("docker-machine", "stop", d.name)
	switch {
	case err == nil:
		// all good
		return nil
	case strings.Contains(err.Error(), "is already stopped"):
		// docker-machine with the specified name is already stopped
		return nil
	default:
		return err
	}
}
