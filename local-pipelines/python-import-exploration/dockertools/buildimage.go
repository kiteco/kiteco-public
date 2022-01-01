package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-golib/cmdline"
)

type buildImageArgs struct {
	In      string `arg:"positional,help:dockerfile to build image from"`
	Cert    string `arg:"help:path to docker certification"`
	Machine string `arg:"help:name of docker machine to build image in"`
}

var buildImageCmd = cmdline.Command{
	Name:     "buildimage",
	Synopsis: "build a dockerimage for the specified file and push the image to dockerhub",
	Args: &buildImageArgs{
		Cert:    defaultDockerCerts,
		Machine: defaultMachine,
	},
}

func (args *buildImageArgs) Handle() (err error) {
	start := time.Now()

	// build and start the docker machine if needed
	machine, status, err := maybeStart(args.Machine, args.Cert)
	if err != nil {
		return err
	}
	defer func() {
		err = maybeJoinErrors(err, cleanupMachine(machine, status))
		fmt.Printf("Done! took %v\n", time.Since(start))
	}()

	// build the image
	if err := buildImage(context.Background(), machine, args.In); err != nil {
		return err
	}

	return nil
}

type buildImagesArgs struct {
	In            string `arg:"positional,required,help:dockerfiles"`
	Cert          string `arg:"help:path to docker certification"`
	Machine       string `arg:"help:name of docker machine to build image in"`
	NumGoroutines int    `arg:"help:number of go routines to use to build images"`
}

var buildImagesCmd = cmdline.Command{
	Name:     "buildimages",
	Synopsis: "build a dockerimage for the specified file and push the image to dockerhub",
	Args: &buildImagesArgs{
		Cert:          defaultDockerCerts,
		Machine:       defaultMachine,
		NumGoroutines: 4,
	},
}

func (args *buildImagesArgs) Handle() (err error) {
	start := time.Now()

	files, err := ioutil.ReadDir(args.In)
	if err != nil {
		return fmt.Errorf("error reading files `%s`: %v", args.In, err)
	}

	var dockerfiles []string
	for _, file := range files {
		if !file.Mode().IsRegular() || file.Name() == ".ds_store" {
			continue
		}
		dockerfiles = append(dockerfiles, filepath.Join(args.In, file.Name()))
	}
	sort.Strings(dockerfiles)

	// build and start the docker machine if needed
	machine, status, err := maybeStart(args.Machine, args.Cert)
	if err != nil {
		return err
	}
	defer func() {
		err = maybeJoinErrors(err, cleanupMachine(machine, status))
	}()

	errs := buildImages(args.NumGoroutines, machine, dockerfiles)

	printFinish("images failed to build", start, errs)

	return nil
}
