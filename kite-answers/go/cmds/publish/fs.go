package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/spf13/afero"
)

// TODO(naman) get rid of this?

func dirFs(dir string) (afero.Fs, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	return afero.NewBasePathFs(afero.NewOsFs(), dir), nil
}

func tgzFs(path string) (afero.Fs, error) {
	var tgzF *os.File
	if path == "-" {
		tgzF = os.Stdin
	} else {
		var err error
		tgzF, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	defer tgzF.Close()

	r, err := gzip.NewReader(tgzF)
	if err != nil {
		return nil, errors.Wrapf(err, "could not gunzip source tarball")
	}

	fs := afero.NewMemMapFs()
	if err := extractTar(r, fs); err != nil {
		return nil, errors.Wrapf(err, "could not extract tar archive")
	}

	return fs, nil
}

func extractTar(r io.Reader, targetFs afero.Fs) error {
	target := afero.Afero{Fs: targetFs}
	tr := tar.NewReader(r)
loop:
	for {
		hdr, err := tr.Next()
		switch err {
		case nil:
		case io.EOF:
			break loop
		default:
			return err
		}

		fi := hdr.FileInfo()
		name := hdr.Name
		mode := fi.Mode()

		log.Printf("processing tar %s", name)
		if fi.IsDir() {
			if err := target.MkdirAll(fi.Name(), fi.Mode()); err != nil {
				return err
			}
		} else {
			if err := target.WriteReader(name, tr); err != nil {
				return err
			}
			if err := target.Chmod(name, mode); err != nil {
				return err
			}
			mtime := fi.ModTime()
			if err := target.Chtimes(name, mtime, mtime); err != nil {
				return err
			}
		}
	}
	return nil
}
