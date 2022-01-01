package fs

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// DirExists returns true if a directory exists and is accessible.
// It returns false if the path is a file, but not a directory.
// It may return false when the path exists but is inaccessible, the disk is failing, etc.
// fixme what about symlinks?
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// FileExists returns true if a file exists and is accessible.
// It returns false if the path is a directory, but not a file.
// It may return false when the path exists but is inaccessible, the disk is failing, etc.
// fixme what about symlinks?
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// KeepExistingFiles returns only those files of the input slice which actually exist on disk
func KeepExistingFiles(paths []string) []string {
	var result []string
	for _, path := range paths {
		if FileExists(path) {
			result = append(result, path)
		}
	}
	return result
}

// CopyDir copies directory 'source' to 'target'. It recursively copies
// all files and directories. Due to it's recursive implementation it's not suitable
// to handle very deep directory structures.
// The target directory is created and must not exist when this function is called.
// It returns an error if either source isn't found or if target already exists.
// It also returns an error when it failed to copy one of the files or subdirectories
func CopyDir(source string, target string) error {
	if !DirExists(source) {
		return errors.Errorf("source directory already exists: %s", source)
	}
	if DirExists(target) {
		return errors.Errorf("target directory already exists: %s", target)
	}

	// create target with the same permissions as the source dir
	if sourceInfo, err := os.Stat(source); err != nil {
		return errors.Errorf("unable to stat source dir %s: $v", source, err)
	} else if err := os.Mkdir(target, sourceInfo.Mode()); err != nil {
		return errors.Errorf("unable to create target dir: %v", err)
	}

	// now copy entries from source into target
	entries, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}

	for _, f := range entries {
		sourcePath := filepath.Join(source, f.Name())
		targetPath := filepath.Join(target, f.Name())

		if f.IsDir() {
			if err := CopyDir(sourcePath, targetPath); err != nil {
				return err
			}
		} else {
			// for now, not supporting symlinks
			if !f.Mode().IsRegular() {
				log.Printf("skipping %s", f.Name())
				continue
			}

			if err := CopyFile(sourcePath, targetPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyFile copies a file from path 'source' to path 'target'
// An error is returned if target already exists.
func CopyFile(source, target string) error {
	if _, err := os.Stat(target); err == nil {
		return errors.Errorf("target file already exists: %v", err)
	} else if !os.IsNotExist(err) {
		return errors.Errorf("error accessing target file: %v", err)
	}

	r, err := os.Open(source)
	if err != nil {
		return errors.Errorf("unable to open file: %v", err)
	}

	stat, err := os.Stat(source)
	if err != nil {
		return errors.Errorf("unable to stat file: %v", err)
	}

	w, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE, stat.Mode())
	if err != nil {
		return errors.Errorf("unable to create target file: %v", err)
	}

	if _, err = io.Copy(w, r); err != nil {
		return errors.Errorf("error writing into target file: %v", err)
	}

	_ = r.Close()
	if err = w.Close(); err != nil {
		return errors.Errorf("error closing target file: %v", err)
	}
	return nil
}

// MoveOrCopyDir moves a directory from source to target
// if an error occurs, then source and target will be left intact (i.e. source still exists and target doesn't exist)
func MoveOrCopyDir(source, target string) error {
	if !DirExists(source) {
		return errors.Errorf("missing source dir: %s", source)
	}
	if DirExists(target) {
		return errors.Errorf("target dir already exists: %s", target)
	}

	// try moving first, might fail when crossing partitions on Linux, for example
	// we assume that this is an atomic operation and that it either fails or succeeded
	if err := os.Rename(source, target); err == nil {
		// success
		return nil
	}

	// move failed, try to copy and then remove
	// fixme find ways to test if target is locked before trying to replace it

	if err := CopyDir(source, target); err != nil {
		_ = os.RemoveAll(target)
		return err
	}

	if err := os.RemoveAll(source); err != nil {
		// restore source with the copy in target
		// for example, RemoveAll() might have removed a few files, but not all
		if err := restoreDir(target, source); err != nil {
			log.Printf("error while restoring dir %s after failed copy: %v", source, err)
		}

		return errors.Errorf("failed to remove source dir %s after copy", source)
	}
	return nil
}

// restoreDir tries to restore everything in target with the files and directories found in source.
// the implementation assumes that files were either removed or are still intact. Therefore, files which still are in
// target are not copied from source into the target dir
func restoreDir(source, target string) error {
	if !DirExists(source) {
		return errors.Errorf("source dir not found: %s", source)
	}

	if !DirExists(target) {
		if err := os.MkdirAll(target, 0700); err != nil {
			return errors.Errorf("unable to create target dir: %s", target)
		}
	}

	var err error
	var sourceEntries, targetEntries []os.FileInfo

	if sourceEntries, err = ioutil.ReadDir(source); err != nil {
		return errors.Errorf("unable to retrieve entries of source dir %s: %v", source, err)
	}
	if targetEntries, err = ioutil.ReadDir(target); err != nil {
		return errors.Errorf("unable to retrieve entries of target dir %s: %v", target, err)
	}

	for _, sourceFile := range sourceEntries {
		sourceFilePath := filepath.Join(source, sourceFile.Name())
		targetFilePath := filepath.Join(target, sourceFile.Name())

		if sourceFile.IsDir() {
			// recursively restore subdirectories, even if it already exists
			if err := restoreDir(sourceFilePath, targetFilePath); err != nil {
				return err
			}
		}

		// not supporting symlinks, only regular files
		// the JetBrains plugin has to symlinks
		if !sourceFile.Mode().IsRegular() {
			continue
		}

		// copy source file -> target file.
		// skip files, which exist in target
		inTarget := false
		for _, t := range targetEntries {
			if sourceFile.Name() == t.Name() {
				inTarget = true
				break
			}
		}

		if !inTarget {
			if err := CopyFile(sourceFilePath, targetFilePath); err != nil {
				// only log, don't stop
				// we must continue to restore remaining files
				log.Printf("error copying %s to %s: %v", sourceFilePath, targetFilePath, err)
			}
		}
	}

	return nil
}
