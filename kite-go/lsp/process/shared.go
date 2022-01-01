package process

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/shirou/gopsutil/process"
)

// IsRunning checks if Kite is running.
func IsRunning(name string) (bool, error) {
	list, err := process.Processes()
	if err != nil {
		log.Printf("error retrieving process list: %s", err.Error())
		return true, err
	}

	for _, p := range list {
		curName, err := p.Name()
		if err != nil {
			continue
		}
		if curName == name {
			return true, nil
		}
	}
	return false, nil
}

func startProcess(name string, additionalEnv []string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = attributes
	if additionalEnv != nil {
		cmd.Env = append(os.Environ(), additionalEnv...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return nil, errors.WrapfOrNil(err, "error running %s %q\nstdout: %s\nstderr: %s", name, strings.Join(arg, " "), stdout.String(), stderr.String())
	}
	return stdout.Bytes(), nil
}

func homeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}
