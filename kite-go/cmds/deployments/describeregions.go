package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/goamz/goamz/aws"
	"github.com/mitchellh/cli"
)

type describeregionsCommand struct{}

func (l *describeregionsCommand) Help() string {
	return "run this command with a release and optional json format, e.g 'describeregions release_0 [json]"
}

func (l *describeregionsCommand) Synopsis() string {
	return "describe a deployment in all regions"
}

func (l *describeregionsCommand) Run(args []string) int {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 1 && len(args) != 2 {
		return cli.RunResultHelp
	}

	// read args
	release := args[0]
	jsonFormat := ""
	if len(args) == 2 && args[1] == "json" {
		jsonFormat = "json"
	}

	var regions []aws.Region
	for awsRegion := range Regions {
		regions = append(regions, awsRegion)
	}
	sort.Sort(regionsByName(regions))

	for _, awsRegion := range regions {
		describeCmd := &describeCommand{}
		ret := describeCmd.Run([]string{awsRegion.Name, release, jsonFormat})
		if ret != 0 {
			ui.Error(fmt.Sprintf("error describing in %s, aborting", awsRegion.Name))
			return 1
		}
	}

	return 0
}
