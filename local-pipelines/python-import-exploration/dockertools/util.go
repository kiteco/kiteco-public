package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kiteco/kiteco/local-pipelines/python-import-exploration/internal/docker"
)

func copyFile(src, dst string) error {
	buf, err := ioutil.ReadFile(src)
	if err != nil {
		return fmt.Errorf("error reading src file `%s`: %v", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return fmt.Errorf("error making dst dir `%s`: %v", filepath.Dir(dst), err)
	}

	if err := ioutil.WriteFile(dst, buf, os.ModePerm); err != nil {
		return fmt.Errorf("error writing dst file `%s`: %v", dst, err)
	}

	return nil
}

func maybeJoinErrors(err1, err2 error) error {
	switch {
	case err1 != nil && err2 != nil:
		return fmt.Errorf("%v: and error: %v", err1, err2)
	case err2 != nil:
		return err2
	case err1 != nil:
		return err1
	default:
		return nil
	}
}

func cleanupMachine(machine *docker.Machine, status docker.Status) error {
	if machine != nil && status == docker.Stopped {
		return machine.Stop()
	}
	return nil
}

func maybeStart(machine, cert string) (*docker.Machine, docker.Status, error) {
	var m *docker.Machine
	var status docker.Status
	if runtime.GOOS != "linux" {
		var err error
		m = docker.NewMachine(machine, cert)
		status, err = m.Status()
		if err != nil {
			return nil, docker.Unknown, err
		}

		switch status {
		case docker.Stopped:
			if err := m.Start(); err != nil {
				return nil, docker.Unknown, err
			}
			if err := m.SetEnv(); err != nil {
				return nil, docker.Unknown, err
			}
		case docker.Running:
			if err := m.SetEnv(); err != nil {
				return nil, docker.Unknown, err
			}
		default:
			return nil, docker.Unknown, fmt.Errorf("docker machine in invalid state: %v", status)
		}
	}
	return m, status, nil
}

type namedError struct {
	Name string
	Err  error
}

func printFinish(job string, start time.Time, errs []namedError) {
	if len(errs) > 0 {
		fmt.Printf("%d %s\n", len(errs), job)
		for _, err := range errs {
			fmt.Println(strings.Repeat("*", 20))
			fmt.Println(err.Name)
			fmt.Println(err.Err.Error())
		}
	}
	fmt.Printf("Took %v\n", time.Since(start))
}
