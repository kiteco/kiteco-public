package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func upload(localPath string, remotePath string, recursive bool) error {
	log.Printf("%s -> %s", localPath, remotePath)

	fi, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		if !recursive {
			return fmt.Errorf("%s is a directory, can only copy recursively", localPath)
		}
		files, err := ioutil.ReadDir(localPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			local := path.Join(localPath, f.Name())
			remote := fmt.Sprintf("%s/%s", remotePath, f.Name())
			if err := upload(local, remote, true); err != nil {
				return err
			}
		}
		return nil
	}

	if fi.Size() == 0 {
		log.Printf("skipping %s, size is 0 bytes", localPath)
		return nil
	}

	r, err := fileutil.NewCachedReader(localPath)
	fail(err)
	defer r.Close()

	w, err := fileutil.NewBufferedWriter(remotePath)
	fail(err)

	_, err = io.Copy(w, r)
	fail(err)

	if err := w.Close(); err != nil {
		return fmt.Errorf("error closing %s: %v", remotePath, err)
	}

	return nil
}

func download(localPath string, remotePath string) error {
	log.Printf("%s -> %s", remotePath, localPath)

	dir := filepath.Dir(localPath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := fileutil.NewReader(remotePath)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(f, r)
	return err
}

func wait(remotePath string) error {
	for {
		exists, err := awsutil.Exists(remotePath)
		if err != nil {
			return err
		}

		if exists {
			return nil
		}

		time.Sleep(time.Minute)
	}
}
