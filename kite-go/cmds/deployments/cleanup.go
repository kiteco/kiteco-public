package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goamz/goamz/autoscaling"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/ec2"
	"github.com/mitchellh/cli"
)

var (
	cleanupCommandFactory = func() (cli.Command, error) {
		return &cleanupCommand{}, nil
	}
)

type cleanupCommand struct{}

func (l *cleanupCommand) Help() string {
	return "run this command with a region, e.g 'deployments cleanup [region]'"
}

func (l *cleanupCommand) Synopsis() string {
	return "cleanup (terminate) unused deployments in a region"
}

func (l *cleanupCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 1 {
		return cli.RunResultHelp
	}

	region := args[0]
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

	var instances []string
	autoscalingClient := autoscaling.New(auth, awsRegion)
	for _, alb := range []string{"prod", "staging"} {
		asg, err := albTargetScalingGroupName(regionInfo, alb)
		if err != nil {
			ui.Error(fmt.Sprintf("error finding autoscaling group for %s: %s", alb, err))
			return 1
		}
		if asg == "" {
			continue
		}

		desc, err := autoscalingClient.DescribeAutoScalingGroups([]string{asg}, 1, "")
		if err != nil {
			ui.Error(fmt.Sprintf("error describing autoscaling group %s: %s", asg, err))
			return 1
		}

		for _, group := range desc.AutoScalingGroups {
			if group.AutoScalingGroupName != asg {
				continue
			}
			for _, instance := range group.Instances {
				instances = append(instances, instance.InstanceId)
			}
		}
	}

	client := ec2.New(auth, awsRegion)
	resp, err := client.DescribeInstances(instances, nil)
	if err != nil {
		ui.Error(fmt.Sprintf("error querying instances in region: %s, vpc: %s: %s", awsRegion.Name, regionInfo.VpcID, err))
		return 1
	}

	activeReleases := make(map[string]bool)
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			for _, tag := range instance.Tags {
				switch tag.Key {
				case "Release":
					activeReleases[tag.Value] = true
				}
			}
		}
	}

	filter := ec2.NewFilter()
	filter.Add("vpc-id", regionInfo.VpcID)
	resp, err = client.DescribeInstances(nil, filter)
	if err != nil {
		ui.Error(fmt.Sprintf("error querying instances in region: %s, vpc: %s: %s", awsRegion.Name, regionInfo.VpcID, err))
		return 1
	}

	allReleases := make(map[string]bool)
	newReleases := make(map[string]bool)
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			var release string
			var deployedAt time.Time
			for _, tag := range instance.Tags {
				switch tag.Key {
				case "Release":
					release = tag.Value
					allReleases[tag.Value] = true
				case "DeployedAt":
					ts, err := strconv.ParseInt(tag.Value, 10, 64)
					if err != nil {
						ui.Error(fmt.Sprintf("cannot parse DeployedAt: %s", tag.Value))
						break
					}
					deployedAt = time.Unix(ts, 0)
					if time.Since(deployedAt) < time.Hour*48 {
						newReleases[release] = true
					}
				}
			}
		}
	}

	for release := range allReleases {
		if _, active := activeReleases[release]; active {
			continue
		}

		releaseName := release
		if pos := strings.Index(release, ":"); pos > 0 {
			releaseName = release[:pos]
		}

		ui.Info(fmt.Sprintf("terminating %s... you have 5 seconds to stop this...", releaseName))
		time.Sleep(5 * time.Second)

		terminateCmd := &terminateCommand{}
		ret := terminateCmd.Run([]string{awsRegion.Name, release})
		if ret != 0 {
			ui.Error(fmt.Sprintf("error terminating %s in %s, aborting", release, awsRegion.Name))
			return 1
		}
	}

	ui.Info("looking for orphaned machines...")
	var orphaned []string
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			var name, release string
			for _, tag := range instance.Tags {
				switch tag.Key {
				case "Name":
					name = tag.Value
				case "Release":
					release = tag.Value
				}
			}
			switch name {
			case "user-mux", "user-node", "user-node-proxy", "local-code-worker", "local-index-viewer", "sandbox-node":
				if release == "" {
					orphaned = append(orphaned, instance.InstanceId)
				}
			}
		}
	}

	ui.Info(fmt.Sprintf("found %d orphaned instances", len(orphaned)))
	if len(orphaned) > 0 {
		ui.Info(fmt.Sprintf("terminating %s", strings.Join(orphaned, ", ")))
		ec2client := ec2.New(auth, awsRegion)
		_, err := ec2client.TerminateInstances(orphaned)
		if err != nil {
			ui.Error(fmt.Sprintf("error terminating instances: %s", err))
			return 1
		}
	}

	return 0
}
