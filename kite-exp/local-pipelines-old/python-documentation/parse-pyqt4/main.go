package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
)

var (
	errNoEntities = errors.New("no entities found")
	debugCoverage bool
)

func main() {
	var root, output string
	flag.StringVar(&root, "root", "", "directory containing PyQt4 html files")
	flag.StringVar(&output, "output", "", "where to output parsed docs")
	flag.BoolVar(&debugCoverage, "coverage", false, "if specified, will output colored parser coverage html")
	flag.Parse()

	out, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	comp := gzip.NewWriter(out)
	defer comp.Close()
	enc := json.NewEncoder(comp)

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			err := parseHTML(path, enc)
			if err != nil && err != errNoEntities {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func parseHTML(path string, enc *json.Encoder) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	module := pythondocs.ParsePyQt4HTML(f, path, debugCoverage)
	if module == nil {
		return nil
	}

	err = module.Encode(enc)
	if module.Entities() == 0 {
		return errNoEntities
	}

	if err != nil {
		return err
	}
	return nil
}
