package main

import (
	"archive/tar"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// modified from tar.FileInfoHeader to discard undesirable metadata
func fileInfoHeader(root, path string, fi os.FileInfo) (*tar.Header, error) {
	if fi == nil {
		return nil, errors.Errorf("FileInfo is nil for path %s", path)
	}

	relpath, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}

	fm := fi.Mode()
	h := &tar.Header{
		Name: relpath,
		Mode: int64(fm.Perm()), // or'd with c_IS* constants later
	}

	switch {
	case fm.IsRegular():
		h.Typeflag = tar.TypeReg
		h.Size = fi.Size()
	case fm.IsDir():
		h.Typeflag = tar.TypeDir
		h.Name += "/"
	case fm&os.ModeSymlink != 0:
		h.Typeflag = tar.TypeSymlink
		link, err := os.Readlink(path)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read symbolic link at path %s", path)
		}
		h.Linkname = link
	default:
		return nil, errors.Errorf("unsupported file mode %s for path %s", fm.String(), path)
	}
	return h, nil
}

func extract(dst string, r io.Reader) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch err {
		case nil:
			// keep going
		case io.EOF:
			return nil
		default:
			return err
		}

		path := filepath.Join(dst, header.Name)
		fm := header.FileInfo().Mode()

		switch {
		case fm.IsDir():
			// assume all parent directories already exist,
			// which should hold if the archive was created with this binary.
			if _, err := os.Stat(path); err != nil {
				if err := os.Mkdir(path, fm); err != nil {
					return err
				}
			}
		case fm.IsRegular():
			err := func() error {
				f, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_RDWR, fm)
				if err != nil {
					return err
				}
				defer f.Close()

				_, err = io.Copy(f, tr)
				return err
			}()
			if err != nil {
				return err
			}
		case fm&os.ModeSymlink != 0:
			if err := os.Symlink(header.Linkname, path); err != nil {
				return err
			}
		default:
			return errors.Errorf("unsupported file mode %s for path %s", fm, path)
		}
	}
}

func archive(src string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// no need to include "." in the archive
		if src == path {
			if info.IsDir() {
				return nil
			}
			return errors.Errorf("must archive a directory")
		}

		hdr, err := fileInfoHeader(src, path, info)
		if err != nil {
			return err
		}

		tw.WriteHeader(hdr)
		if hdr.Typeflag != tar.TypeReg {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return errors.Errorf("failed to create archive: %s", err)
	}
	return nil
}

func main() {
	log.SetFlags(0)
	if len(os.Args) != 3 {
		log.Printf("usage: %s [archive|extract] path/to/dir\n", os.Args[0])
		os.Exit(1)
	}
	cmd := os.Args[1]
	dirpath := os.Args[2]

	var err error
	switch cmd {
	case "archive":
		err = archive(dirpath, os.Stdout)
	case "extract":
		err = extract(dirpath, os.Stdin)
	}
	if err != nil {
		log.Println("error:", err)
		os.Exit(1)
	}
}
