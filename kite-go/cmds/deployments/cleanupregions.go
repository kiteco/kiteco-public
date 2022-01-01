package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/goamz/goamz/aws"
	"github.com/mitchellh/cli"
)

type cleanupregionsCommand struct{}

func (l *cleanupregionsCommand) Help() string {
	return "run this command with no arguments e.g 'deployments cleanupregions'"
}

func (l *cleanupregionsCommand) Synopsis() string {
	return "cleanup (terminate) unused deployments in all regions"
}

func (l *cleanupregionsCommand) Run(args []string) int {
	if len(args) > 0 {
		return cli.RunResultHelp
	}

	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	var regions []aws.Region
	for awsRegion := range Regions {
		regions = append(regions, awsRegion)
	}
	sort.Sort(regionsByName(regions))

	for _, awsRegion := range regions {
		cleanupCmd := &cleanupCommand{}
		ret := cleanupCmd.Run([]string{awsRegion.Name})
		if ret != 0 {
			ui.Error(fmt.Sprintf("error cleaning up in %s, aborting", awsRegion.Name))
			return 1
		}
	}

	return 0
}
