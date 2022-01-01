//go:generate go-bindata -pkg $GOPACKAGE -prefix rootless-system-data -o bindata.go rootless-system-data/...
package main

import (
	"log"
	"os"
	"os/user"

	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/mitchellh/cli"
)

var version string
var rootMsg = `This installer should not be run as root! All the files installed by this binary are within
the user's home directory.`

func main() {
	log.SetFlags(0)

	u, err := user.Current()
	if err != nil {
		log.Printf("unable to determine current user: %s", err.Error())
		os.Exit(1)
	}

	if u.Uid == "0" {
		log.Print(rootMsg)
	}

	localManager := newLocalManager()
	localVersion, _ := localManager.currentVersion()

	c := cli.NewCLI("kite-installer", version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"install": func() (cli.Command, error) {
			if localVersion != "" {
				log.Println("Found an installation of Kite. Calling update instead.")
				return &updateCommand{localManager: localManager}, nil
			}
			return &installCommand{localManager: localManager}, nil
		},
		"update": func() (cli.Command, error) {
			if localVersion == "" {
				log.Println("No installation of Kite was found. Calling install instead.")
				return &installCommand{localManager: localManager}, nil
			}
			return &updateCommand{localManager: localManager}, nil
		},
		"self-update": func() (cli.Command, error) {
			return &updateCommand{localManager: localManager, selfUpdate: true}, nil
		},
		"update-system-data": func() (cli.Command, error) {
			return &updateSystemDataCommand{}, nil
		},
		"uninstall": func() (cli.Command, error) {
			return &uninstallCommand{}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	rollbar.Wait()

	os.Exit(exitStatus)
}
