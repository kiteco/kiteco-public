package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/goamz/goamz/aws"
	"github.com/mitchellh/cli"
)

type switchregionsCommand struct{}

func (l *switchregionsCommand) Help() string {
	return "run this command with release, and elb e.g 'deployments switchregions release [staging|prod]'"
}

func (l *switchregionsCommand) Synopsis() string {
	return "switch load balancers in all regions to point to a deployment"
}

func (l *switchregionsCommand) Run(args []string) int {
	if len(args) != 2 {
		return cli.RunResultHelp
	}

	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	release, elb := args[0], args[1]
	var regions []aws.Region
	for awsRegion := range Regions {
		regions = append(regions, awsRegion)
	}
	sort.Sort(regionsByName(regions))

	for _, awsRegion := range regions {
		switchCmd := &switchCommand{}
		ret := switchCmd.Run([]string{awsRegion.Name, release, elb})
		if ret != 0 {
			ui.Error(fmt.Sprintf("error switching alb in %s, aborting", awsRegion.Name))
			return 1
		}
	}

	return 0
}
