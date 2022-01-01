package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	newaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/ec2"
)

type listCommand struct{}

func (l *listCommand) Help() string {
	return ""
}

func (l *listCommand) Synopsis() string {
	return "list all deployments in all regions"
}

func (l *listCommand) Run(args []string) int {
	var regions []aws.Region
	for awsRegion := range Regions {
		regions = append(regions, awsRegion)
	}
	sort.Sort(regionsByName(regions))

	for _, awsRegion := range regions {
		region := Regions[awsRegion]
		client := ec2.New(auth, awsRegion)

		filter := ec2.NewFilter()
		filter.Add("vpc-id", region.VpcID)
		resp, err := client.DescribeInstances(nil, filter)
		if err != nil {
			log.Printf("error querying instances in region: %s, vpc: %s: %s", awsRegion.Name, region.VpcID, err)
			return 1
		}

		releaseMap := make(map[string]bool)
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				for _, tag := range instance.Tags {
					switch tag.Key {
					case "Release":
						releaseMap[tag.Value] = true
					}
				}
			}
		}

		var releases []string
		for release := range releaseMap {
			releases = append(releases, release)
		}

		sess, err := session.NewSession()
		if err != nil {
			log.Printf("error building aws session: %s", err)
			return 1
		}

		autoscalingClient := autoscaling.New(sess, newaws.NewConfig().WithRegion(awsRegion.Name))

		descResp, err := autoscalingClient.DescribeAutoScalingGroups(nil)
		if err != nil {
			log.Printf("error describing autoscaling groups: %s", err)
			return 1
		}

		var prodASG, stagingASG string
		for _, group := range descResp.AutoScalingGroups {
			if containsString(region.ProdUserNodeTargetGroupARN, group.TargetGroupARNs) {
				prodASG = valueForTag("Release", group.Tags)
			}
			if containsString(region.StagingUserNodeTargetGroupARN, group.TargetGroupARNs) {
				stagingASG = valueForTag("Release", group.Tags)
			}
		}

		fmt.Println("region:", awsRegion.Name)
		fmt.Println("\tactive on prod-alb:", prodASG)
		fmt.Println("\tactive on staging-alb:", stagingASG)
		fmt.Println("\tavailable releases:\n", "\t\t"+strings.Join(releases, "\n\t\t"))
	}

	return 0
}
