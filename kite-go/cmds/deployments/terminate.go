package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/goamz/goamz/autoscaling"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/ec2"
	"github.com/mitchellh/cli"
)

var (
	terminateCommandFactory = func() (cli.Command, error) {
		return &terminateCommand{}, nil
	}
)

type terminateCommand struct{}

func (l *terminateCommand) Help() string {
	return "run this command with a region and release, e.g 'deployments terminate [region] [release]'"
}

func (l *terminateCommand) Synopsis() string {
	return "terminate a deployment in a region"
}

func (l *terminateCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 2 {
		return cli.RunResultHelp
	}

	region, release := args[0], args[1]
	awsRegion, exists := aws.Regions[region]
	if !exists {
		ui.Error(fmt.Sprintf("region %s does not exist", region))
		return 1
	}

	regionInfo, exists := Regions[awsRegion]
	if !exists {
		ui.Error(fmt.Sprintf("region %s is not supported", region))
		return 1
	}

	prefix := fmt.Sprintf("[%s] ", awsRegion.Name)
	ui = &cli.PrefixedUi{
		AskPrefix:       prefix,
		AskSecretPrefix: prefix,
		OutputPrefix:    prefix,
		InfoPrefix:      prefix,
		ErrorPrefix:     prefix,
		WarnPrefix:      prefix,
		Ui:              ui,
	}

	client := ec2.New(auth, awsRegion)

	filter := ec2.NewFilter()
	filter.Add("vpc-id", regionInfo.VpcID)
	filter.Add("tag-key", "Release")
	filter.Add("tag-value", release)
	resp, err := client.DescribeInstances(nil, filter)
	if err != nil {
		ui.Error(fmt.Sprintf("error querying instances in region: %s, vpc: %s: %s", awsRegion.Name, regionInfo.VpcID, err))
		return 1
	}

	instanceMap := make(map[string]bool)
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			instanceMap[instance.InstanceId] = true
		}
	}

	var isLive bool

	// Check if production applicaiton load balancer is pointing to this release
	prodASG, err := albTargetScalingGroupName(regionInfo, "prod")
	if err != nil {
		ui.Error(fmt.Sprintf("error finding autoscaling group for prod: %s", err))
		return 1
	}
	if prodASG != "" {
		if release == releaseFromGroupName(prodASG) {
			isLive = true
		}
	}

	if isLive {
		resp, err := ui.Ask("production load balancer still pointing to this release. are you sure you want to continue? [yes/no]")
		if err != nil || resp != "yes" {
			ui.Info("aborting!")
			return 1
		}
		ui.Info("continuing with termination!")
	}

	if len(instanceMap) == 0 {
		ui.Info("no matching instances found")
	}

	// Delete the autoscaling groups, or the instances will just get re-launched
	scalingClient := autoscaling.New(auth, awsRegion)

	for _, process := range []string{"user-node", "user-mux", "local-code-worker", "sandbox-node"} {
		name := groupName(release, process)
		desc, err := scalingClient.DescribeAutoScalingGroups([]string{name}, 1, "")
		if err != nil {
			ui.Error(fmt.Sprintf("error quering %s autoscaling group in region %s: %s", process, awsRegion.Name, err))
			continue
		}

		var found bool
		var removed []string
		for _, group := range desc.AutoScalingGroups {
			if group.AutoScalingGroupName != name {
				continue
			}

			found = true
			for _, instance := range group.Instances {
				if _, ok := instanceMap[instance.InstanceId]; !ok {
					ui.Error(fmt.Sprintf("found unexpected instance in autoscaling group: %s, expected one of: %v", instance.InstanceId, instanceMap))
					return 1
				}
				// remove from list of ids to terminate manually
				delete(instanceMap, instance.InstanceId)
				removed = append(removed, instance.InstanceId)
			}
		}

		if found {
			// force delete auto scaling group and instances it contains
			name := groupName(release, process)

			ui.Info(fmt.Sprintf("about to terminate autoscaling group %s with %d %s instances: %s", name, len(removed), process, strings.Join(removed, ",")))

			_, err := scalingClient.DeleteAutoScalingGroup(name, true)
			if err != nil {
				ui.Error(fmt.Sprintf("error deleting autoscaling group %s: %s", name, err))
				return 1
			}

			// delete the launch configuration too
			_, err = scalingClient.DeleteLaunchConfiguration(name)
			if err != nil {
				ui.Error(fmt.Sprintf("error deleting launch configuration %s: %s", name, err))
				return 1
			}
		}
	}

	// Remove anything left over
	var ids []string
	for id := range instanceMap {
		ids = append(ids, id)
	}

	if len(ids) > 0 {
		ui.Info(fmt.Sprintf("about to terminate instances: %s", strings.Join(ids, ", ")))
		_, err = client.TerminateInstances(ids)
		if err != nil {
			ui.Error(fmt.Sprintf("error terminating instances: %s", err))
			return 1
		}
	}

	return 0
}
