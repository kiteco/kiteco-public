package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	oldscaling "github.com/goamz/goamz/autoscaling"
	"github.com/goamz/goamz/ec2"
)

func groupName(release, process string) string {
	return fmt.Sprintf("%s-%s", release, process)
}

func releaseFromGroupName(name string) string {
	parts := strings.Split(name, "-")
	if len(parts) > 3 {
		return strings.Join(parts[:len(parts)-2], "-")
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func albTargetScalingGroupName(region Region, albName string) (string, error) {
	var targetGroupARN string
	switch albName {
	case "prod":
		targetGroupARN = region.ProdUserNodeTargetGroupARN
	case "staging":
		targetGroupARN = region.StagingUserNodeTargetGroupARN
	default:
		panic("invalid load balancer name (use prod or staging)")
	}

	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}

	autoscalingClient := autoscaling.New(sess, aws.NewConfig().WithRegion(region.AwsRegion.Name))

	descResp, err := autoscalingClient.DescribeAutoScalingGroups(nil)
	if err != nil {
		return "", err
	}

	for _, group := range descResp.AutoScalingGroups {
		if containsString(targetGroupARN, group.TargetGroupARNs) {
			return *group.AutoScalingGroupName, nil
		}
	}

	return "", nil
}

func scalingGroupInstances(region Region, name string) ([]ec2.Instance, error) {
	client := oldscaling.New(auth, region.AwsRegion)
	desc, err := client.DescribeAutoScalingGroups([]string{name}, 1, "")
	if err != nil {
		return nil, err
	}

	for _, group := range desc.AutoScalingGroups {
		if group.AutoScalingGroupName != name {
			continue
		}

		var ids []string
		for _, instance := range group.Instances {
			ids = append(ids, instance.InstanceId)
		}

		ec2client := ec2.New(auth, region.AwsRegion)
		resp, err := ec2client.DescribeInstances(ids, nil)
		if err != nil {
			return nil, err
		}

		var instances []ec2.Instance
		for _, reservation := range resp.Reservations {
			instances = append(instances, reservation.Instances...)
		}

		return instances, nil
	}

	return nil, fmt.Errorf("scaling group not found")
}

func waitTargetScalingGroupState(region Region, groupARN, name, state string) error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	autoscalingClient := autoscaling.New(sess, aws.NewConfig().WithRegion(region.AwsRegion.Name))

	start := time.Now()
	var prevState string
	for range time.NewTicker(10 * time.Second).C {
		if time.Since(start) > 5*time.Minute {
			return fmt.Errorf("timed out waiting for %s to enter %s state", name, state)
		}

		descInput := &autoscaling.DescribeLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: &name,
		}
		descGroupResp, err := autoscalingClient.DescribeLoadBalancerTargetGroups(descInput)
		if err != nil {
			return err
		}

		var found bool
		var inDesiredState bool
		for _, group := range descGroupResp.LoadBalancerTargetGroups {
			if *group.LoadBalancerTargetGroupARN == groupARN {
				if prevState != *group.State {
					fmt.Print(*group.State + ".")
					prevState = *group.State
				} else {
					fmt.Print(".")
				}

				if *group.State == state {
					inDesiredState = true
				}
				found = true
			}
		}

		// If the desired state is Removed, the TargetGroup may no longer report
		// the state of the autoscaling group (since we are removing it!).
		// We need to handle this case explicitly.
		if !found && state == "Removed" {
			fmt.Println("Removed.")
			break
		}

		if inDesiredState {
			fmt.Println()
			break
		}
	}

	return nil
}
