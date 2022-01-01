package scalinggroups

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

var (
	provider string
	region   string
)

const (
	testInstanceRelease = "test-instance"
)

func init() {
	provider = envutil.GetenvDefault("PROVIDER", "aws")

	// get region based on provider
	switch provider {
	case "aws":
		region = envutil.GetenvDefault("AWS_REGION", "us-west-1")
	case "azure":
		region = envutil.GetenvDefault("AZURE_REGION", "westus2")
	}
}

// IPs returns a list of private IPs for the scaling group machines
func IPs(process, release string) ([]string, error) {
	if release == testInstanceRelease {
		return []string{"127.0.0.1"}, nil
	}

	switch provider {
	case "azure":
		return azIPs(process, release, region)
	case "aws":
		return awsIPs(process, release, region)
	default:
		return []string{}, fmt.Errorf("invalid provider: %s", provider)
	}
}

// RegionIPs returns a list of private IPs for the scaling group machines for a specific region
func RegionIPs(process, release, r string) ([]string, error) {
	if release == testInstanceRelease {
		return []string{"127.0.0.1"}, nil
	}

	switch provider {
	case "azure":
		return azIPs(process, release, r)
	case "aws":
		return awsIPs(process, release, r)
	default:
		return []string{}, fmt.Errorf("invalid provider: %s", provider)
	}
}

// authAzure gets credentials from env and gets an auth token
func authAzure() (string, *adal.ServicePrincipalToken, error) {
	// get azure credentials from environment
	spName, spPass, spTenant :=
		os.Getenv("AZURE_SERVICE_PRINCIPAL_NAME"),
		os.Getenv("AZURE_SERVICE_PRINCIPAL_PASSWORD"),
		os.Getenv("AZURE_SERVICE_PRINCIPAL_TENANT")

	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	// authenticate to get service principal token
	oauth, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, spTenant)
	if err != nil {
		return "", nil, err
	}
	auth, err := adal.NewServicePrincipalToken(*oauth, spName, spPass, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", nil, err
	}

	return subID, auth, nil
}

// azIPs is the Azure implementation of IPs
func azIPs(process, release, region string) ([]string, error) {
	var ips []string

	// authenticate and init client
	subID, auth, err := authAzure()
	if err != nil {
		return ips, err
	}
	client := network.NewInterfacesClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	// get network interfaces from scale set
	rgName := releaseRGName(release, region)
	l, err := client.ListVirtualMachineScaleSetNetworkInterfaces(context.Background(), rgName, process)
	if err != nil {
		return ips, err
	}
	nis := l.Values()

	// iterate over network interfaces and their ip configs to get the private IPs
	for _, ni := range nis {
		if ni.IPConfigurations == nil {
			return ips, fmt.Errorf("scale set %s for %s has no ip configurations", process, release)
		}
		for _, ipc := range *ni.IPConfigurations {
			ips = append(ips, *ipc.PrivateIPAddress)
		}
	}

	return ips, nil
}

// releaseRGName returns the azure resource group name based on the release and region name
func releaseRGName(release string, location string) string {
	return fmt.Sprintf("%s-%s", release, location)
}

// awsIPs is the AWS implementation of IPs
func awsIPs(process, release, region string) ([]string, error) {
	name := awsGroupName(release, process)

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	client := autoscaling.New(sess, aws.NewConfig().WithRegion(region))

	descInput := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&name},
	}

	descOutput, err := client.DescribeAutoScalingGroups(descInput)
	if err != nil {
		return nil, err
	}

	var instanceIDs []*string
	for _, group := range descOutput.AutoScalingGroups {
		for _, instance := range group.Instances {
			instanceIDs = append(instanceIDs, instance.InstanceId)
		}
	}

	if len(instanceIDs) == 0 {
		return nil, fmt.Errorf("found no instances in autoscaling group %s", name)
	}

	ec2client := ec2.New(sess, aws.NewConfig().WithRegion(region))

	instancesInput := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	}

	instancesOutput, err := ec2client.DescribeInstances(instancesInput)
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, res := range instancesOutput.Reservations {
		for _, instance := range res.Instances {
			ips = append(ips, *instance.PrivateIpAddress)
		}
	}

	return ips, nil
}

// awsGroupName is a helper to get the autoscaling group name
func awsGroupName(release, process string) string {
	return fmt.Sprintf("%s-%s", release, process)
}
