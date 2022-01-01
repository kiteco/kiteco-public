package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/goamz/goamz/aws"
	"github.com/mitchellh/cli"
)

type terminateregionsCommand struct{}

func (l *terminateregionsCommand) Help() string {
	return "run this command with release, and elb e.g 'deployments terminateregions release'"
}

func (l *terminateregionsCommand) Synopsis() string {
	return "terminate a deployment in all regions"
}

func (l *terminateregionsCommand) Run(args []string) int {
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
		terminateCmd := &terminateCommand{}
		ret := terminateCmd.Run([]string{awsRegion.Name, release})
		if ret != 0 {
			ui.Error(fmt.Sprintf("error terminating %s in %s, aborting", release, awsRegion.Name))
			return 1
		}
	}

	return 0
}
