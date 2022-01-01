package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/local-pipelines/python-import-exploration/internal/docker"
)

const defaultMachine = "default"

var (
	defaultDockerCerts = filepath.Join(envutil.MustGetenv("HOME"), ".docker/machine/machines/default")
	defaultOut         = filepath.Join(envutil.MustGetenv("GOPATH"), "/src/github.com/kiteco/kiteco/local-pipelines/python-import-exploration/tmp")
)

func explore(ctx context.Context, machine *docker.Machine, out string, image string) error {
	run := []string{"-i", "--rm", "-v", out + ":/host", "-w", "/host", image}

	if err := docker.RunWithContext(ctx, machine, run...); err != nil {
		return fmt.Errorf("error running image `%s`: %v", image, err)
	}

	return nil
}

type packageArgs struct {
	Image   string `arg:"positional,required,help:name of the docker image to run"`
	Out     string `arg:"help:directory to write output to must be an absolute path"`
	Cert    string `arg:"help:path to docker certification"`
	Machine string `arg:"help:name of docker machine to build image in"`
}

var packageCmd = cmdline.Command{
	Name:     "package",
	Synopsis: "explore a package",
	Args: &packageArgs{
		Out:     defaultOut,
		Cert:    defaultDockerCerts,
		Machine: defaultMachine,
	},
}

func (args *packageArgs) Handle() (err error) {
	start := time.Now()

	if !filepath.IsAbs(args.Out) {
		return fmt.Errorf("output path must be absolute: %v", args.Out)
	}

	if err := os.MkdirAll(args.Out, os.ModePerm); err != nil {
		return fmt.Errorf("error making output dir `%s`: %v", args.Out, err)
	}

	// build and start the docker machine if needed
	machine, status, err := maybeStart(args.Machine, args.Cert)
	if err != nil {
		return err
	}

	defer func() {
		err = maybeJoinErrors(err, cleanupMachine(machine, status))
		fmt.Printf("Done! took %v\n", time.Since(start))
	}()

	if err := explore(context.TODO(), machine, args.Out, args.Image); err != nil {
		return err
	}

	return nil
}

type packagesArgs struct {
	Images        string `arg:"positional,required,help:names of the docker images to run"`
	Out           string `arg:"help:directory to write output to must be an absolute path"`
	Cert          string `arg:"help:path to docker certification"`
	Machine       string `arg:"help:name of docker machine to run images in"`
	NumGoroutines int    `arg:"help:number of go routines to use to explore images"`
}

var packagesCmd = cmdline.Command{
	Name:     "packages",
	Synopsis: "explore a set of packages",
	Args: &packagesArgs{
		Out:           defaultOut,
		Cert:          defaultDockerCerts,
		Machine:       defaultMachine,
		Images:        "../dockertools/dockerfiles",
		NumGoroutines: 4,
	},
}

func (args *packagesArgs) Handle() (err error) {
	start := time.Now()
	if !filepath.IsAbs(args.Out) {
		return fmt.Errorf("output path must be absolute: %v", args.Out)
	}

	if err := os.MkdirAll(args.Out, os.ModePerm); err != nil {
		return fmt.Errorf("error making output dir `%s`: %v", args.Out, err)
	}

	files, err := ioutil.ReadDir(args.Images)
	if err != nil {
		return fmt.Errorf("error reading images dir `%s`: %v", args.Images, err)
	}

	var images []string
	for _, f := range files {
		if !f.Mode().IsRegular() || f.Name() == ".ds_store" {
			continue
		}
		images = append(images, f.Name())
	}
	sort.Strings(images)

	machine, status, err := maybeStart(args.Machine, args.Cert)
	if err != nil {
		return err
	}
	defer func() {
		err = maybeJoinErrors(err, cleanupMachine(machine, status))
		fmt.Printf("Done! took %v\n", time.Since(start))
	}()

	errs := exploreImages(args.NumGoroutines, machine, args.Out, images)

	if len(errs) > 0 {
		fmt.Printf("%d images failed:", len(errs))
		for _, err := range errs {
			fmt.Println(strings.Repeat("*", 20))
			fmt.Println(err.Name)
			fmt.Println(err.Err.Error())
		}
	} else {
		fmt.Println("All images ran succesfully!")
	}

	return nil
}

type namedError struct {
	Name string
	Err  error
}

func (n namedError) Error() string {
	return fmt.Sprintf("name: %s\n%v", n.Name, n.Err)
}

func maybeStart(machine, cert string) (*docker.Machine, docker.Status, error) {
	var m *docker.Machine
	var status docker.Status
	if runtime.GOOS != "linux" {
		var err error
		m = docker.NewMachine(machine, cert)
		status, err = m.Status()
		if err != nil {
			return nil, docker.Unknown, err
		}

		switch status {
		case docker.Stopped:
			if err := m.Start(); err != nil {
				return nil, docker.Unknown, err
			}
			if err := m.SetEnv(); err != nil {
				return nil, docker.Unknown, err
			}
		case docker.Running:
			if err := m.SetEnv(); err != nil {
				return nil, docker.Unknown, err
			}
		default:
			return nil, docker.Unknown, fmt.Errorf("docker machine in invalid state: %v", status)
		}
	}
	return m, status, nil
}

func maybeJoinErrors(err1, err2 error) error {
	switch {
	case err1 != nil && err2 != nil:
		return fmt.Errorf("%v: and error: %v", err1, err2)
	case err2 != nil:
		return err2
	case err1 != nil:
		return err1
	default:
		return nil
	}
}

func cleanupMachine(machine *docker.Machine, status docker.Status) error {
	if machine != nil && status == docker.Stopped {
		return machine.Stop()
	}
	return nil
}

func main() {
	cmdline.MustDispatch(packageCmd, packagesCmd)
}
