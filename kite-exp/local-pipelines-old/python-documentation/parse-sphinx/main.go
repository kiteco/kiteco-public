package main

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"encoding/gob"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/tarball"
)

var (
	errNoEntities = errors.New("no entities found")
	debugCoverage bool
)

func main() {
	var root, output string
	flag.StringVar(&root, "root", "", "directory containing rtd zip files")
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
	enc := gob.NewEncoder(comp)

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatalln("Error loading import graph:", err)
	}

	parser := pythondocs.NewDocParser(graph)
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".zip":
			err := parseZIP(path, enc, parser)
			if err == errNoEntities {
				log.Println("no entries found in", path)
			} else if err != nil {
				return err
			}
		case ".bz2":
			// HACK: Special case for parsing the stdlib (the only bz2 file currently)
			err := parseStdLib(path, enc, parser)
			if err != nil {
				return err
			}
		case ".html":
			r, err := os.Open(path)
			if err != nil {
				return err
			}
			defer r.Close()

			module := parser.ParseSphinxHTML(r, path, debugCoverage)
			if module == nil {
				log.Println("Parsing single HTML: module is nil")
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func parseZIP(path string, enc *gob.Encoder, parser *pythondocs.DocParser) error {
	zipArc, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zipArc.Close()

	var entities int
	for _, f := range zipArc.File {
		err := func() error {
			if filepath.Ext(f.Name) != ".html" {
				return nil
			}
			r, err := f.Open()
			if err != nil {
				return err
			}
			defer r.Close()

			module := parser.ParseSphinxHTML(r, f.Name, debugCoverage)
			if module == nil {
				return nil
			}

			err = module.EncodeGob(enc)
			entities += module.Entities()
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	if entities == 0 {
		return errNoEntities
	}

	log.Println(path, "yielded", entities, "entities")
	return nil
}

func parseStdLib(path string, enc *gob.Encoder, parser *pythondocs.DocParser) error {
	parseFn := func(header *tar.Header, r io.Reader) error {
		// Ignore non-file entries
		if header.Typeflag != tar.TypeReg {
			return nil
		}

		if matched, _ := filepath.Match("*/library/*.html", header.Name); matched {
			module := parser.ParseSphinxHTML(r, header.Name, debugCoverage)
			if module == nil {
				return nil
			}
			err := module.EncodeGob(enc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Bzip reader for input
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	decomp := bzip2.NewReader(in)
	err = tarball.Walk(decomp, parseFn)
	if err != nil {
		return err
	}

	return nil
}
