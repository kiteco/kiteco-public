package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/goamz/goamz/autoscaling"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/ec2"
	"github.com/goamz/goamz/s3"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/mitchellh/cli"
)

const (
	commonProductionInstanceProfile = "common-production"
	tagAttempts                     = 3
)

type deployCommand struct {
	templates *templateset.Set
}

func newDeployCommand() (*deployCommand, error) {
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(staticfs, "templates", nil)
	if err := templates.Validate(); err != nil {
		return nil, err
	}
	return &deployCommand{templates: templates}, nil
}

func (d *deployCommand) Help() string {
	return "run this command with a region and release, e.g 'deployments deploy region release [tag]'"
}

func (d *deployCommand) Synopsis() string {
	return "deploy release to specified region"
}

func (d *deployCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 2 && len(args) != 3 {
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

	bucket := s3.New(auth, aws.Regions["us-west-1"]).Bucket("kite-deploys")
	listResp, err := bucket.List(release+"/", "/", "", 0)
	if err != nil {
		ui.Error(fmt.Sprintf("error listing binaries for release %s", release))
		return 1
	}

	contents := make(map[string]bool)
	for _, key := range listResp.Contents {
		contents[key.Key] = true
	}

	required := []string{"user-mux", "user-node"}
	for _, req := range required {
		key := fmt.Sprintf("%s/%s", release, req)
		if !contents[key] {
			ui.Error(fmt.Sprintf("could not find %s for %s", req, release))
			return 1
		}
	}

	// Optional release tag
	if len(args) == 3 {
		release = fmt.Sprintf("%s-%s", release, args[2])
	}

	ui.Info(fmt.Sprintf("release tag: %s", release))

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

	var instances []ec2.Instance
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance)
		}
	}

	if len(instances) > 0 {
		ui.Error("there are already instances with this release! run describe for more info, or terminate before redeploying")
		return 1
	}

	ts := time.Now()

	ui.Info("creating autoscaling group for user-node...")
	nodes, err := d.createAutoScalingGroup(release, regionInfo, "user-node", regionInfo.UserNodeReplication, regionInfo.UserNodeInstanceType, ts)
	if err != nil {
		ui.Error(fmt.Sprintf("error creating user-node autoscaling group: %s", err))
		return 1
	}
	ui.Info(fmt.Sprintf("created %d user-node instances in autoscaling group: %s", len(nodes), strings.Join(instanceIps(nodes), ", ")))

	ui.Info("creating autoscaling group for user-mux...")
	muxes, err := d.createAutoScalingGroup(release, regionInfo, "user-mux", regionInfo.UserMuxReplication, regionInfo.UserMuxInstanceType, ts)
	if err != nil {
		ui.Error(fmt.Sprintf("error creating user-mux autoscaling group: %s", err))
		return 1
	}
	ui.Info(fmt.Sprintf("created %d user-mux instances in autoscaling group: %s", len(muxes), strings.Join(instanceIps(muxes), ", ")))

	return 0
}

// Create autoscaling group for a process
func (d *deployCommand) createAutoScalingGroup(release string, region Region, process string, count int, instanceType string, ts time.Time) ([]ec2.Instance, error) {
	client := autoscaling.New(auth, region.AwsRegion)
	name := groupName(release, process)

	lc := &autoscaling.LaunchConfiguration{
		LaunchConfigurationName: name,
		InstanceType:            instanceType,
		KeyName:                 region.KeyPair,
		ImageId:                 region.AMIImage,
		SecurityGroups:          []string{region.AllowAllVPN, region.AllowSSH, region.AllowUsernode, region.AllowUsernodeDebug},
		UserData:                string(commonUserData(d.templates, process, release, region)),
		IamInstanceProfile:      commonProductionInstanceProfile,
	}

	_, err := client.CreateLaunchConfiguration(lc)
	if err != nil {
		return nil, err
	}

	group := &autoscaling.CreateAutoScalingGroupParams{
		AutoScalingGroupName:    name,
		LaunchConfigurationName: name,
		VPCZoneIdentifier:       strings.Join([]string{region.PrivateSubnetID}, ","),
		DefaultCooldown:         600, // 10 minutes between scaling events
		DesiredCapacity:         count,
		MaxSize:                 count,
		MinSize:                 count,
		HealthCheckGracePeriod:  600, // 10 minutes
		HealthCheckType:         "EC2",
		Tags: []autoscaling.Tag{
			autoscaling.Tag{
				Key:               "Name",
				Value:             process,
				PropagateAtLaunch: true,
			},
			autoscaling.Tag{
				Key:               "Release",
				Value:             release,
				PropagateAtLaunch: true,
			},
			autoscaling.Tag{
				Key:               "DeployedAt",
				Value:             fmt.Sprintf("%d", ts.Unix()),
				PropagateAtLaunch: true,
			},
		},
	}

	_, err = client.CreateAutoScalingGroup(group)
	if err != nil {
		return nil, err
	}

	// Check to see if the autoscaling group has launched the desired number of instances

	start := time.Now()
	for range time.NewTicker(10 * time.Second).C {
		if time.Since(start) > time.Minute {
			return nil, fmt.Errorf("timed out waiting for scaling group to start")
		}

		desc, err := client.DescribeAutoScalingGroups([]string{name}, 1, "")
		if err != nil {
			return nil, err
		}

		for _, group := range desc.AutoScalingGroups {
			if group.AutoScalingGroupName != name {
				continue
			}
			if len(group.Instances) != count {
				break
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
	}

	return nil, fmt.Errorf("unexpected error occurred")
}

func commonUserData(templates *templateset.Set, process, release string, region Region) []byte {
	type data struct {
		ReleaseBin    string
		Release       string
		ReleaseNoDots string
		ReleaseMD5    string
		Region        string
		Process       string
		IsProduction  bool
	}

	parts := strings.Split(release, "-")
	releaseBin := parts[0]

	tmplData := data{
		Region:        region.AwsRegion.Name,
		ReleaseBin:    releaseBin,
		Release:       release,
		ReleaseNoDots: strings.Replace(release, ".", "_", -1),
		ReleaseMD5:    fmt.Sprintf("%x", md5.Sum([]byte(release))),
		Process:       process,
		IsProduction:  strings.HasPrefix(release, "release_"),
	}

	buf := &bytes.Buffer{}
	if err := templates.RenderText(buf, "common-userdata.sh", tmplData); err != nil {
		log.Fatalln("error building common user data:", err)
	}

	return buf.Bytes()
}
