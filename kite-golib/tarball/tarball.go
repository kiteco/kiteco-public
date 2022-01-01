package tarball

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

// Pack creates a tarball from a file or directory
func Pack(inpath string, w io.Writer) error {
	tarWr := tar.NewWriter(w)
	defer tarWr.Close()

	// Walk the filesystem
	err := filepath.Walk(inpath, func(itemPath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("cannot pack symlinks in tarballs: %s", itemPath)
		}

		relpath, err := filepath.Rel(inpath, itemPath)
		if err != nil {
			return err
		}

		// Create the header
		header, err := tar.FileInfoHeader(fileInfo, "")
		header.Name = relpath

		err = tarWr.WriteHeader(header)
		if err != nil {
			return err
		}

		// Only write data if we have a regular file
		if !fileInfo.IsDir() {
			// Open the file
			f, err := os.Open(itemPath)
			if err != nil {
				return err
			}
			defer f.Close()

			// Copy bytes from the file stream to the tar stream
			_, err = io.Copy(tarWr, f)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

// Visitor is the type of function called for each entry in a tar archive encounted by the Walk method.
type Visitor func(header *tar.Header, r io.Reader) error

// Walk takes an io.Reader for a tar archive and invokes the provided Vistor function
// to all the entries in the archive.
func Walk(r io.Reader, vf Visitor) error {
	t := tar.NewReader(r)

	for {
		header, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		err = vf(header, t)
		if err != nil {
			return err
		}
	}

	return nil
}

// Unpack creates a file or directory at the given path from the given tarball
func Unpack(outpath string, r io.Reader) error {
	extractFunc := func(header *tar.Header, r io.Reader) error {
		var err error
		// Get filename for this item
		filename := path.Join(outpath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// It's a directory so run mkdir
			err = os.MkdirAll(filename, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

		case tar.TypeReg:
			// It's a file so write contents to file
			w, err := os.Create(filename)
			if err != nil {
				return err
			}

			_, err = io.Copy(w, r)
			if err != nil {
				return err
			}

			err = os.Chmod(filename, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			w.Close()
		default:
			return fmt.Errorf("Unable to untar type %c in file %s", header.Typeflag, filename)
		}

		return nil
	}

	return Walk(r, extractFunc)
}

// PackGzipBytes creates a tarball from a file or directory
func PackGzipBytes(path string) ([]byte, error) {
	buf := &bytes.Buffer{}
	comp := gzip.NewWriter(buf)
	err := Pack(path, comp)
	if err != nil {
		return nil, err
	}
	err = comp.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnpackGzipBytes creates a file or directory at the given path from the given tarball
func UnpackGzipBytes(path string, buf []byte) error {
	decomp, err := gzip.NewReader(bytes.NewBuffer(buf))
	if err != nil {
		return err
	}
	return Unpack(path, decomp)
}
