package main

import (
	"fmt"
	"os"

	newaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/ec2"
	"github.com/kiteco/kiteco/kite-golib/scalinggroups"
	"github.com/mitchellh/cli"
)

type switchCommand struct{}

func (l *switchCommand) Help() string {
	return "run this command with a region, release, and alb e.g 'deployments switch [region] [release] [staging|prod]'"
}

func (l *switchCommand) Synopsis() string {
	return "switch application load balancer to point to a deployment in a region"
}

func (l *switchCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 3 {
		return cli.RunResultHelp
	}

	region, release, albName := args[0], args[1], args[2]
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

	var targetGroupARN string
	switch albName {
	case "prod":
		targetGroupARN = regionInfo.ProdUserNodeTargetGroupARN
	case "staging":
		targetGroupARN = regionInfo.StagingUserNodeTargetGroupARN
	default:
		ui.Error("invalid load balancer name (use prod or staging)")
		return 1
	}

	ui.Info(fmt.Sprintf("switching release for %s", albName))
	ui.Info("checking to see if release " + release + " is ready...")

	client := ec2.New(auth, awsRegion)

	filter := ec2.NewFilter()
	filter.Add("vpc-id", regionInfo.VpcID)
	filter.Add("tag:Release", release)
	resp, err := client.DescribeInstances(nil, filter)
	if err != nil {
		ui.Error(fmt.Sprintf("error querying instances in region: %s, vpc: %s: %s", awsRegion.Name, regionInfo.VpcID, err))
		return 1
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			var name string
			for _, tag := range instance.Tags {
				switch tag.Key {
				case "Name":
					name = tag.Value
				}
			}

			var status string
			switch name {
			case "user-node", "user-mux", "local-code-worker", "sandbox-node":
				status = scalinggroups.Status(instance.PrivateIPAddress)
				ui.Info(fmt.Sprintf("%s at %s is %s", name, instance.PrivateIPAddress, status))
				if status != "ready" {
					ui.Error(fmt.Sprintf("%s not ready, aborting!", release))
					return 1
				}
			}
		}
	}

	ui.Info("release looks good! lets do this")

	oldASG, err := albTargetScalingGroupName(regionInfo, albName)
	if err != nil {
		ui.Error(fmt.Sprintf("error finding autoscaling group for %s: %s", albName, err))
		return 1
	}

	newASG := groupName(release, "user-mux")
	if oldASG == newASG {
		ui.Error(fmt.Sprintf("%s alb already pointing to %s", albName, oldASG))
		return 0
	}

	sess, err := session.NewSession()
	if err != nil {
		ui.Error(fmt.Sprintf("error building session: %s", err))
		return 1
	}

	autoscalingClient := autoscaling.New(sess, newaws.NewConfig().WithRegion(awsRegion.Name))

	ui.Info(fmt.Sprintf("attaching %s to target group for %s", newASG, albName))

	attachInput := &autoscaling.AttachLoadBalancerTargetGroupsInput{
		AutoScalingGroupName: &newASG,
		TargetGroupARNs:      []*string{&targetGroupARN},
	}

	_, err = autoscalingClient.AttachLoadBalancerTargetGroups(attachInput)
	if err != nil {
		ui.Error(fmt.Sprintf("error attaching %s to %s alb: %s", newASG, albName, err))
		return 1
	}

	ui.Info(fmt.Sprintf("waiting for %s to be InService..", newASG))

	err = waitTargetScalingGroupState(regionInfo, targetGroupARN, newASG, "Added")
	if err != nil {
		ui.Error(fmt.Sprintf("error waiting for target group to become 'InService': %s", err))
		return 1
	}

	if oldASG != "" {
		ui.Info(fmt.Sprintf("detaching previous release (%s) from target group for %s", oldASG, albName))

		detachInput := &autoscaling.DetachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: &oldASG,
			TargetGroupARNs:      []*string{&targetGroupARN},
		}
		_, err = autoscalingClient.DetachLoadBalancerTargetGroups(detachInput)
		if err != nil {
			ui.Error(fmt.Sprintf("error detaching previous release (%s) from %s alb: %s", oldASG, albName, err))
			return 1
		}

		ui.Info(fmt.Sprintf("waiting for %s to be out of InService..", oldASG))

		err := waitTargetScalingGroupState(regionInfo, targetGroupARN, oldASG, "Removed")
		if err != nil {
			ui.Error(fmt.Sprintf("error waiting for target group to become 'Removed': %s", err))
			return 1
		}
	}

	return 0
}

func containsString(r string, vals []*string) bool {
	for _, val := range vals {
		if r == *val {
			return true
		}
	}
	return false
}

func valueForTag(key string, tags []*autoscaling.TagDescription) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}
