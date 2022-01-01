//go:generate go-bindata -o bindata.go templates
package main

import (
	"log"
	"os"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/mitchellh/cli"
)

var (
	auth aws.Auth
)

func init() {
	var err error
	auth, err = aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	c := cli.NewCLI("deployments", "1.0.0")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"list": func() (cli.Command, error) {
			return &listCommand{}, nil
		},
		"find": func() (cli.Command, error) {
			return &findCommand{}, nil
		},
		"describe": func() (cli.Command, error) {
			return &describeCommand{}, nil
		},
		"describeregions": func() (cli.Command, error) {
			return &describeregionsCommand{}, nil
		},
		"deploy": func() (cli.Command, error) {
			return newDeployCommand()
		},
		"deployregions": func() (cli.Command, error) {
			return &deployregionsCommand{}, nil
		},
		"cleanup": func() (cli.Command, error) {
			return &cleanupCommand{}, nil
		},
		"cleanupregions": func() (cli.Command, error) {
			return &cleanupregionsCommand{}, nil
		},
		"terminate": func() (cli.Command, error) {
			return &terminateCommand{}, nil
		},
		"terminateregions": func() (cli.Command, error) {
			return &terminateregionsCommand{}, nil
		},
		"switch": func() (cli.Command, error) {
			return &switchCommand{}, nil
		},
		"switchregions": func() (cli.Command, error) {
			return &switchregionsCommand{}, nil
		},
		"finduser": func() (cli.Command, error) {
			return &finduserCommand{}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
