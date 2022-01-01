package deployments

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

var (
	awsRegionToProdARN = map[string]string{
		"us-west-1": "arn:aws:elasticloadbalancing:us-west-1:XXXXXXX:targetgroup/prod-usernodes/c53b241d25e7fb88",
		"us-east-1": "arn:aws:elasticloadbalancing:us-east-1:XXXXXXX:targetgroup/prod-usernodes/fd115f7497a0bf43",
		"eu-west-1": "arn:aws:elasticloadbalancing:eu-west-1:XXXXXXX:targetgroup/prod-usernodes/3ead9981b75b6a87",
	}
	awsRegionToStagingARN = map[string]string{
		"us-west-1": "arn:aws:elasticloadbalancing:us-west-1:XXXXXXX:targetgroup/staging-usernodes/df1b31495b21b0a4",
		"us-east-1": "arn:aws:elasticloadbalancing:us-east-1:XXXXXXX:targetgroup/staging-usernodes/1c2aaddfa41793e6",
		"eu-west-1": "arn:aws:elasticloadbalancing:eu-west-1:XXXXXXX:targetgroup/staging-usernodes/4133d514c24b88a1",
	}
	awsRegionToVPC = map[string]string{
		"us-west-1": "vpc-109def75",
		"us-east-1": "vpc-b5ca3fd2",
		"eu-west-1": "vpc-2c1bda48",
	}
)

func releaseFromGroupName(name string) string {
	parts := strings.Split(name, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func albTargetScalingGroupName(region, albName string) (string, error) {
	var targetGroupARN string
	switch albName {
	case "prod":
		targetGroupARN = awsRegionToProdARN[region]
	case "staging":
		targetGroupARN = awsRegionToStagingARN[region]
	default:
		panic("invalid load balancer name (use prod or staging)")
	}

	autoscalingClient := autoscaling.New(awsSess, aws.NewConfig().WithRegion(region))

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

func containsString(r string, vals []*string) bool {
	for _, val := range vals {
		if r == *val {
			return true
		}
	}
	return false
}
