package main

import "github.com/goamz/goamz/aws"

var (
	// DefaultParams ...
	DefaultParams = Params{
		UserNodeReplication:  2,
		UserNodeInstanceType: "r5.large",
		UserMuxReplication:   2,
		UserMuxInstanceType:  "m5.large",
	}
	// Regions contains a map from aws region to region-specific constants
	Regions = map[aws.Region]Region{
		aws.Regions["us-west-1"]: {
			AwsRegion:                     aws.Regions["us-west-1"],
			KeyPair:                       "kite-prod",
			VpcID:                         "vpc-XXXXXXX",
			AvailabilityZone:              "us-west-1b",
			ProdUserNodeTargetGroupARN:    "arn:aws:elasticloadbalancing:us-west-1:XXXXXXX:targetgroup/prod-usernodes/XXXXXXX",
			StagingUserNodeTargetGroupARN: "arn:aws:elasticloadbalancing:us-west-1:XXXXXXX:targetgroup/staging-usernodes/XXXXXXX",
			PublicSubnetID:                "subnet-XXXXXXX",
			PrivateSubnetID:               "subnet-XXXXXXX",
			AMIImage:                      "ami-XXXXXXX",
			AllowAllVPN:                   "sg-XXXXXXX",
			AllowHTTP:                     "sg-XXXXXXX",
			AllowHTTPS:                    "sg-XXXXXXX",
			AllowSSH:                      "sg-XXXXXXX",
			AllowUsernode:                 "sg-XXXXXXX",
			AllowUsernodeDebug:            "sg-XXXXXXX",
			Params:                        DefaultParams,
		},
		aws.Regions["us-east-1"]: {
			AwsRegion:                     aws.Regions["us-east-1"],
			KeyPair:                       "kite-prod",
			VpcID:                         "vpc-XXXXXXX",
			AvailabilityZone:              "us-east-1a",
			ProdUserNodeTargetGroupARN:    "arn:aws:elasticloadbalancing:us-east-1:XXXXXXX:targetgroup/prod-usernodes/XXXXXXX",
			StagingUserNodeTargetGroupARN: "arn:aws:elasticloadbalancing:us-east-1:XXXXXXX:targetgroup/staging-usernodes/XXXXXXX",
			PublicSubnetID:                "subnet-XXXXXXX",
			PrivateSubnetID:               "subnet-XXXXXXX",
			AMIImage:                      "ami-XXXXXXX",
			AllowAllVPN:                   "sg-XXXXXXX",
			AllowHTTP:                     "sg-XXXXXXX",
			AllowHTTPS:                    "sg-XXXXXXX",
			AllowSSH:                      "sg-XXXXXXX",
			AllowUsernode:                 "sg-XXXXXXX",
			AllowUsernodeDebug:            "sg-XXXXXXX",
			Params:                        DefaultParams,
		},
		aws.Regions["eu-west-1"]: {
			AwsRegion:                     aws.Regions["eu-west-1"],
			KeyPair:                       "kite-prod",
			VpcID:                         "vpc-XXXXXXX",
			AvailabilityZone:              "eu-west-1a",
			ProdUserNodeTargetGroupARN:    "arn:aws:elasticloadbalancing:eu-west-1:XXXXXXX:targetgroup/prod-usernodes/XXXXXXX",
			StagingUserNodeTargetGroupARN: "arn:aws:elasticloadbalancing:eu-west-1:XXXXXXX:targetgroup/staging-usernodes/XXXXXXX",
			PublicSubnetID:                "subnet-XXXXXXX",
			PrivateSubnetID:               "subnet-XXXXXXX",
			AMIImage:                      "ami-XXXXXXX",
			AllowAllVPN:                   "sg-XXXXXXX",
			AllowHTTP:                     "sg-XXXXXXX",
			AllowHTTPS:                    "sg-XXXXXXX",
			AllowSSH:                      "sg-XXXXXXX",
			AllowUsernode:                 "sg-XXXXXXX",
			AllowUsernodeDebug:            "sg-XXXXXXX",
			Params:                        DefaultParams,
		},
	}
)

// Params encapsulates deployment parameters that are not necessarily tied to a specific region
type Params struct {
	UserNodeReplication  int
	UserNodeInstanceType string
	UserMuxReplication   int
	UserMuxInstanceType  string
}

// Region contains region-specific object ides such as VPC ID, subnet IDs,
// security group IDs and AMI image ids.
type Region struct {
	AwsRegion aws.Region
	KeyPair   string

	// Networking
	VpcID            string
	AvailabilityZone string
	PublicSubnetID   string
	PrivateSubnetID  string

	// Application Load Balancer user-node target groups
	ProdUserNodeTargetGroupARN    string
	StagingUserNodeTargetGroupARN string

	AMIImage string

	// Security Groups
	AllowAllVPN        string
	AllowHTTP          string
	AllowHTTPS         string
	AllowSSH           string
	AllowUsernode      string
	AllowUsernodeDebug string

	Params
}

type regionsByName []aws.Region

func (r regionsByName) Len() int           { return len(r) }
func (r regionsByName) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r regionsByName) Less(i, j int) bool { return r[i].Name < r[j].Name }
