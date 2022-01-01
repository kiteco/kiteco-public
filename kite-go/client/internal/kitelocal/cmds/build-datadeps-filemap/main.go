package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

var (
	dataFile   = "data.blob"
	offsetFile = "offsets.gob"
	cacheroot  = envutil.GetenvDefault("KITE_S3CACHE", "/var/kite/s3cache")
)

func main() {
	var outputDir string
	var verifyOnly bool
	flag.StringVar(&outputDir, "output", "", "output directory")
	flag.BoolVar(&verifyOnly, "verify", false, "verify data is in sync")
	flag.Parse()

	if outputDir == "" && !verifyOnly {
		log.Fatal(fmt.Errorf("output directory flag or verify flag required"))
	}

	// Nuke cache
	if err := os.RemoveAll(cacheroot); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cacheroot, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	// Forces models to load immediately vs lazily
	tensorflow.ForceLoadCycle = true

	// Create file map writer
	writer := fileutil.NewFileMapWriter()

	// Load datasets
	_, err := kitelocal.LoadPythonServices(context.Background(), kitelocal.LoadOptions{DatadepsMode: true})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions.WithRemoteModels(lexicalmodels.DefaultRemoteHost))
	if err != nil {
		log.Fatalln(err)
	}

	// Walk cache and add sorted files
	log.Println("starting cache walk")

	start := time.Now()

	var files []string
	err = filepath.Walk(cacheroot, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".s3cache-checksum") {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		log.Fatalln(err)
	}

	sort.Strings(files)

	for _, f := range files {
		r, err := os.Open(f)
		if err != nil {
			log.Printf("error opening file %s: %v", f, err)
			continue
		}
		path := strings.TrimPrefix(f, cacheroot)
		path = strings.TrimPrefix(path, "/")
		err = writer.AddFile(path, r)
		if err != nil {
			log.Printf("error adding file %s: %v", f, err)
			continue
		}
		err = r.Close()
		if err != nil {
			log.Printf("error closing file %s: %v", f, err)
			continue
		}
	}

	log.Printf("cache walk took: %s", time.Since(start))

	// Create data and offset buffers
	var data, offsets bytes.Buffer
	if err := writer.WriteOffsets(&offsets); err != nil {
		log.Fatal(err)
	}
	if err := writer.WriteData(&data); err != nil {
		log.Fatal(err)
	}

	// Compare hash of data to hash in existing bindata
	newHash := datadeps.Hash(data.Bytes(), offsets.Bytes())
	oldHash, err := datadeps.CurrentHash()
	if err != nil {
		log.Fatal(err)
	}

	if verifyOnly {
		if oldHash == newHash {
			log.Println("kitelocal datadeps matches :)")
			return
		}
		log.Println("kitelocal datadeps mismatch :(")
		os.Exit(1)
	}

	if oldHash == newHash {
		log.Println("kitelocal datadeps already up to date")
		// Signals not to update the bindata
		os.Exit(1)
	}

	// Clean up old local data
	if err := os.RemoveAll(outputDir); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	// Update data and offset files
	dataPath := filepath.Join(outputDir, dataFile)
	err = ioutil.WriteFile(dataPath, data.Bytes(), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	offsetPath := filepath.Join(outputDir, offsetFile)
	err = ioutil.WriteFile(offsetPath, offsets.Bytes(), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("new kitelocal datadeps have been written to", outputDir)
}
