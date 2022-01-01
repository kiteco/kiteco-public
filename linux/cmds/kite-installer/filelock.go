package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/unix"
)

const staleLockFileAge = 24 * time.Hour

// fileLock is a filePath based lock
type fileLock struct {
	filePath string

	ctx        context.Context
	ctxCancel  func()
	signalChan chan os.Signal
	file       *os.File
}

func newFileLock(filePath string) *fileLock {
	ctx, cancel := context.WithCancel(context.Background())
	return &fileLock{
		filePath:   filePath,
		ctx:        ctx,
		ctxCancel:  cancel,
		signalChan: make(chan os.Signal, 1),
	}
}

// Lock tries to create the file.
// It installs required signal handlers to make sure that Unlock
// is called when the program terminates unexpectedly.
// It returns an error if the file already exists or if the creation of the file failed.
// The lock is never left on disk if an error is returned.
func (f *fileLock) Lock() error {
	fd, err := os.OpenFile(f.filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		// it's still possible that this is a stale lock file
		// for example, termination by SIGKILL could have caused this
		// we continue if the lock file is at least a day old
		if stat, statErr := os.Stat(f.filePath); statErr == nil && stat != nil && time.Since(stat.ModTime()) >= staleLockFileAge {
			_ = os.Remove(f.filePath)
			if fd, err = os.OpenFile(f.filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	defer fd.Close()

	go func() {
		for {
			select {
			case <-f.ctx.Done():
				return
			case s := <-f.signalChan:
				log.Println("\ncaught signal", s.String())
				f.Unlock()
				os.Exit(1)
			}
		}
	}()

	signal.Notify(f.signalChan, os.Interrupt, unix.SIGQUIT, unix.SIGTERM)
	return nil
}

// Unlock removes the lock file
func (f *fileLock) Unlock() {
	defer f.ctxCancel()
	defer signal.Stop(f.signalChan)

	_ = os.Remove(f.filePath)
}
