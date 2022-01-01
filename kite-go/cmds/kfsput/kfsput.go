package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/goamz/goamz/s3"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func main() {
	var args struct {
		Local       string `arg:"positional,required,help:Local path"`
		Remote      string `arg:"positional,required,help:Remote path (insert %t for timestamp)"`
		ContentType string
		Region      string
		ACL         s3.ACL
	}
	arg.MustParse(&args)

	// Avoid polluting output stream with log
	log.SetOutput(ioutil.Discard)

	// Add timestamp if requested
	timestamp := time.Now().Format("2006-01-02_03-04-05-PM")
	dest := strings.Replace(args.Remote, "%t", timestamp, -1)
	fmt.Printf("%s -> %s\n", args.Local, dest)

	fi, err := os.Stat(args.Local)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	start := time.Now()
	var numUploaded int
	defer func() {
		fmt.Printf("Done, took %v to upload %d files\n", time.Since(start), numUploaded)
	}()

	if fi.IsDir() {
		err := filepath.Walk(args.Local, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(args.Local, path)
			if err != nil {
				return fmt.Errorf("error relativizing path %s to %s: %v", path, args.Local, err)
			}
			numUploaded++

			remote := fileutil.Join(dest, rel)

			if err := put(remote, path); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	numUploaded++
	if err := put(dest, args.Local); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func put(remote string, local string) error {
	rf, err := fileutil.NewBufferedWriter(remote)
	if err != nil {
		return fmt.Errorf("error creating writer for remote %s: %v", remote, err)
	}

	lf, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("error opening local file %s: %v", local, err)
	}
	defer lf.Close()

	if _, err := io.Copy(rf, lf); err != nil {
		return fmt.Errorf("error copying %s to %s: %v", local, remote, err)
	}

	if err := rf.Close(); err != nil {
		return fmt.Errorf("error closing remote file %s: %v", remote, err)
	}
	return nil
}
