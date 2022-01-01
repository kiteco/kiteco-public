package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
)

type commandController struct {
	m       sync.Mutex
	cmd     *exec.Cmd
	process *process.Process

	outf *os.File
}

func newCommandController(name string, arg ...string) *commandController {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return &commandController{
		cmd: cmd,
	}
}

func (t *commandController) log(dir, name string) error {
	t.m.Lock()
	defer t.m.Unlock()

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	ts := time.Now().Format("01-02-2006-15:04:05")
	outf, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s-%s.log", ts, name)))
	if err != nil {
		return err
	}

	t.outf = outf

	stdout := io.MultiWriter(os.Stdout, outf)
	stderr := io.MultiWriter(os.Stderr, outf)

	t.cmd.Stdout = stdout
	t.cmd.Stderr = stderr

	return nil
}

func (t *commandController) start() error {
	log.Println("starting", t.cmd.String())
	t.m.Lock()
	defer t.m.Unlock()

	err := t.cmd.Start()
	if err != nil {
		return err
	}

	process, err := process.NewProcess(int32(t.cmd.Process.Pid))
	if err != nil {
		return err
	}

	t.process = process
	return nil
}

func (t *commandController) stop() error {
	log.Println("stopping", t.cmd.String())
	t.m.Lock()
	defer t.m.Unlock()
	if t.process == nil {
		return errors.Wrap(nil, "process not started")
	}

	defer t.outf.Close()

	err := t.process.Kill()
	if err != nil {
		return err
	}
	t.process = nil

	err = t.cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (t *commandController) running() (bool, error) {
	t.m.Lock()
	defer t.m.Unlock()
	if t.process == nil {
		return false, nil
	}

	running, err := t.process.IsRunning()
	if err != nil {
		return false, errors.Wrap(err, "failed to determine if process is running")
	}

	return running, nil
}

func (t *commandController) wait() error {
	log.Println("waiting", t.cmd.String())
	defer t.outf.Close()
	return t.cmd.Wait()
}
