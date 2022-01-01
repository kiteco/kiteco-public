package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/goamz/goamz/aws"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/mitchellh/cli"
)

type deployregionsCommand struct {
	templates *templateset.Set
}

func (d *deployregionsCommand) Help() string {
	return "run this command with a release, e.g 'deployments deployregions release [replication]'"
}

func (d *deployregionsCommand) Synopsis() string {
	return "deploy release to all supported regions"
}

func (d *deployregionsCommand) Run(args []string) int {
	if len(args) != 1 {
		return cli.RunResultHelp
	}

	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	release := args[0]

	var regions []aws.Region
	for awsRegion := range Regions {
		regions = append(regions, awsRegion)
	}
	sort.Sort(regionsByName(regions))

	for _, awsRegion := range regions {
		deployCmd, err := newDeployCommand()
		if err != nil {
			ui.Error(fmt.Sprintf("error building deploy command: %s", err))
			return 1
		}
		ret := deployCmd.Run([]string{awsRegion.Name, release})
		if ret != 0 {
			ui.Error(fmt.Sprintf("error deploying to %s, aborting", awsRegion.Name))
			return 1
		}
	}

	return 0
}
