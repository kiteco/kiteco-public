package fileutil

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ProcessTar by unpacking the archive and then calling the provided
// function `f` on each file with the name of the file in the archive and the
// contents of the file in the provided reader.
// hat tip https://gist.github.com/indraniel/1a91458984179ab4cf80
func ProcessTar(r io.Reader, f func(string, io.Reader) error) error {
	for tr := tar.NewReader(r); ; {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error processing tar: %v", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		if err := f(header.Name, tr); err != nil {
			return err
		}
	}
	return nil
}

// ExtractTarGZFromStream deflates the archive in the gzipStream reader to the targetDir
// Warning, it currently only deflate files, empty directory won't be recreated
// To do so, the 'if' part has to process additional header.Typeflag (refer to the github gist for example)
func ExtractTarGZFromStream(gzipStream io.Reader, targetDir string) error {
	gz, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}

	return ProcessTar(gz, func(path string, r io.Reader) error {
		path = filepath.Join(targetDir, path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("error making dir '%s': %v", filepath.Dir(path), err)
		}

		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(f, r); err != nil {
			return fmt.Errorf("error copying contents: %v", err)
		}
		return nil
	})
}

// ExtractTarGZ deflates the archive in the gzipStream reader to the targetDir
// archivePath can be a s3 URL or a local filepath
// Warning, it currently only deflate files, empty directory won't be recreated
// The function ExtractTarGZFromStream needs to be updated to also process folder (see comment there)
func ExtractTarGZ(archivePath string, targetDir string) error {
	stream, err := NewCachedReader(archivePath)
	if err != nil {
		return err
	}
	return ExtractTarGZFromStream(stream, targetDir)
}
