package main

import (
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/local-pipelines/python-import-exploration/internal/docker"
)

type deleteImageArgs struct {
	In       string `arg:"positional,help:name of the dockerimage to delete"`
	Cert     string `arg:"help:path to docker certification"`
	Machine  string `arg:"help:name of docker machine to build image in"`
	Registry string `arg:"help:docker registry to push image to"`
}

var deleteImageCmd = cmdline.Command{
	Name:     "deleteimage",
	Synopsis: "Delete a dockerimage",
	Args: &deleteImageArgs{
		Cert:    defaultDockerCerts,
		Machine: defaultMachine,
	},
}

func (args *deleteImageArgs) Handle() (err error) {
	start := time.Now()

	machine, status, err := maybeStart(args.Machine, args.Cert)
	if err != nil {
		return err
	}

	defer func() {
		maybeJoinErrors(err, cleanupMachine(machine, status))
	}()

	// delete the image
	if err := docker.DeleteImage(machine, args.In); err != nil {
		return err
	}
	fmt.Printf("done! took %v\n", time.Since(start))

	return nil
}

type deleteImagesArgs struct {
	In       string `arg:"positional,required,help:dockerimages to delete either a file with image names or a directory with dockerfiles matching the image names to delete"`
	Cert     string `arg:"help:path to docker certification"`
	Machine  string `arg:"help:name of docker machine to build image in"`
	Registry string `arg:"help:docker registry to push image to"`
}

var deleteImagesCmd = cmdline.Command{
	Name:     "deleteimages",
	Synopsis: "delete a set of dockerimages",
	Args: &deleteImagesArgs{
		Cert:    defaultDockerCerts,
		Machine: defaultMachine,
	},
}

func (args *deleteImagesArgs) Handle() error {
	start := time.Now()

	var dockerimages []string
	files, err := ioutil.ReadDir(args.In)
	if err != nil {
		return fmt.Errorf("error reading dir `%s`: %v", args.In, err)
	}

	for _, f := range files {
		if !f.Mode().IsRegular() {
			continue
		}
		dockerimages = append(dockerimages, f.Name())
	}
	sort.Strings(dockerimages)

	// build and start the docker machine if needed
	machine, status, err := maybeStart(args.Machine, args.Cert)
	if err != nil {
		return err
	}
	var failures []namedError
	for _, dockerimage := range dockerimages {
		if err := docker.DeleteImage(machine, dockerimage); err != nil {
			failures = append(failures, namedError{
				Name: dockerimage,
				Err:  err,
			})
		}
	}

	printFinish("images failed to delete", start, failures)

	return cleanupMachine(machine, status)
}
