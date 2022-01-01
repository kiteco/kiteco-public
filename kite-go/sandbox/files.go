package sandbox

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func createFiles(path string, files map[string][]byte) error {
	for relpath, data := range files {
		if strings.HasPrefix(relpath, "/") {
			return fmt.Errorf("paths should be relative but got " + relpath)
		}

		fullpath := filepath.Join(path, relpath)
		dir := filepath.Dir(fullpath)
		if err := os.MkdirAll(dir, 0777); err != nil {
			return fmt.Errorf("could not create %s: %v", dir, err)
		}

		if err := ioutil.WriteFile(fullpath, data, 0777); err != nil {
			return fmt.Errorf("could not create %s: %v", fullpath, err)
		}
	}
	return nil
}

func collectFiles(rootpath string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error while enumerating %s: %v", rootpath, err)
			return nil
		}

		// Do not include anything other than regular files in the OutputFiles output files
		if !info.Mode().IsRegular() {
			return nil
		}

		// Get the relative path
		relpath, err := filepath.Rel(rootpath, path)
		if err != nil {
			log.Printf("Unable to get path for %s relative to %s: %v\n", path, rootpath, err)
			return nil
		}

		// Use the full path so that we locate the file correctly
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("Unable to read contents of %s: %v\n", path, err)
			return nil
		}
		files[relpath] = data
		return nil
	})
	if err != nil {
		log.Printf("Error walking %s: %v\n", rootpath, err)
	}
	return files, nil
}
