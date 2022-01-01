package deployments

import (
	"fmt"
	"log"
	"sort"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

// Region contains the release, and IP's of user-node and local-code-worker instances in a deployment for a region
type Region struct {
	Region             string
	Release            string
	UsernodeIPs        []string
	LocalCodeWorkerIPs []string
	UserMuxIPs         []string
}

var (
	// cloud service provider and respective authentication
	provider   string
	awsSess    client.ConfigProvider
	azureAuth  *adal.ServicePrincipalToken
	azureSubID string
)

func init() {
	switch provider = envutil.MustGetenv("PROVIDER"); provider {
	case "aws":
		// authenticate aws
		awsSess = session.Must(session.NewSession())
	case "azure":
		// get azure credentials from environment
		spName, spPass, spTenant :=
			envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_NAME"),
			envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_PASSWORD"),
			envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_TENANT")

		azureSubID = envutil.MustGetenv("AZURE_SUBSCRIPTION_ID")

		// authenticate azure to get service principal token
		oauth, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, spTenant)
		if err != nil {
			log.Fatalln(err)
		}
		azureAuth, err = adal.NewServicePrincipalToken(*oauth, spName, spPass, azure.PublicCloud.ResourceManagerEndpoint)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

// Production returns a slice of Region objects corresponding to all production deployments
func Production() ([]Region, error) {
	var d []Region
	var dep Region
	var err error
	for _, region := range Regions() {
		dep, err = deploymentForRegion(region, "prod")
		if err != nil {
			log.Println(err)
			continue
		}
		d = append(d, dep)
	}
	return d, err
}

// Staging returns a slice of Region objects corresponding to all staging deployments
func Staging() ([]Region, error) {
	var d []Region
	var dep Region
	var err error
	for _, region := range Regions() {
		dep, err = deploymentForRegion(region, "staging")
		if err != nil {
			log.Println(err)
			continue
		}
		d = append(d, dep)
	}
	return d, nil
}

// Regions returns a sorted slice of strings containing the active regions for the provider
func Regions() []string {
	var r []string
	switch provider {
	case "aws":
		r = awsRegions()
	case "azure":
		r = azureRegions()
	}
	sort.Strings(r)
	return r
}

// awsRegions returns the active aws regions
func awsRegions() []string {
	var r []string
	for region := range awsRegionToProdARN {
		r = append(r, region)
	}
	return r
}

// azureRegions returns the active azure regions
func azureRegions() []string {
	var r []string
	for region := range azureRegionInfo {
		r = append(r, region)
	}
	return r
}

// deploymentForRegion checks provider and delegates to one of the provider specific functions
func deploymentForRegion(region, target string) (Region, error) {
	switch provider {
	case "aws":
		return awsDeployments(region, target)
	case "azure":
		return azureDeployments(region, target)
	default:
		return Region{}, fmt.Errorf("Invalid provider %s", provider)
	}
}

// awsDeployments returns a populated Region struct for a given aws region
func awsDeployments(region, elbName string) (Region, error) {
	vpcID, ok := awsRegionToVPC[region]
	if !ok {
		return Region{}, fmt.Errorf("no vpc found for region: %s", region)
	}
	asg, err := albTargetScalingGroupName(region, elbName)
	if err != nil {
		return Region{}, err
	}
	if asg == "" {
		return Region{}, fmt.Errorf("no deployment found for %s %s", region, elbName)
	}

	release := releaseFromGroupName(asg)
	if release == "" {
		return Region{}, fmt.Errorf("could not find release in %s", region)
	}

	client := ec2.New(awsSess, aws.NewConfig().WithRegion(region))
	filters := []*ec2.Filter{
		&ec2.Filter{Name: aws.String("vpc-id"), Values: []*string{&vpcID}},
		&ec2.Filter{Name: aws.String("tag:Release"), Values: []*string{&release}},
	}
	resp, err := client.DescribeInstances(&ec2.DescribeInstancesInput{Filters: filters})
	if err != nil {
		return Region{}, fmt.Errorf("error describing instnaces in region: %s, vpc: %s", region, vpcID)
	}

	d := Region{
		Region:  region,
		Release: release,
	}

	// Populate IP's of nodes that are part of the active release
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			var nodeName string
			var nodeRelease string
			for _, tag := range instance.Tags {
				switch key := *tag.Key; key {
				case "Release":
					nodeRelease = *tag.Value
				case "Name":
					nodeName = *tag.Value
				}
			}
			if nodeRelease == release {
				switch nodeName {
				case "user-node":
					d.UsernodeIPs = append(d.UsernodeIPs, *instance.PrivateIpAddress)
				case "local-code-worker":
					d.LocalCodeWorkerIPs = append(d.LocalCodeWorkerIPs, *instance.PrivateIpAddress)
				case "user-mux":
					d.UserMuxIPs = append(d.UserMuxIPs, *instance.PrivateIpAddress)
				}
			}
		}
	}

	return d, nil
}

// azureDeployments returns a populated Region struct for a given azure region
func azureDeployments(region, target string) (Region, error) {
	var r Region

	// get deployment info for region
	d, ok := azureRegionInfo[region]
	if !ok {
		return r, fmt.Errorf("invalid region %s", region)
	}

	// get release name for target appgate using deployment info
	release, err := getAzureRelease(target, d)
	if err != nil {
		return r, err
	}

	r.Region = region
	r.Release = release
	r.UsernodeIPs, err = azIPs("user-node", release, region)
	if err != nil {
		return r, err
	}
	r.UserMuxIPs, err = azIPs("user-mux", release, region)
	if err != nil {
		return r, err
	}
	r.LocalCodeWorkerIPs, err = azIPs("local-code-worker", release, region)
	if err != nil {
		return r, err
	}

	return r, nil
}
