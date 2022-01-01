package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/mitchellh/cli"
)

type terminateCommand struct{}

func (d *terminateCommand) Help() string {
	return "run this command with a region and release, e.g 'terminate westus release_0'"
}

func (d *terminateCommand) Synopsis() string {
	return "terminate a deployment in a region"
}

func (d *terminateCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 2 {
		return cli.RunResultHelp
	}

	// read command line args
	region, ok := Regions[args[0]]
	if !ok {
		ui.Error(fmt.Sprintf("region %s does not exist", args[0]))
	}
	release := args[1]

	// use region for logging prefix
	prefix := fmt.Sprintf("[%s] ", region.Location)
	ui = &cli.PrefixedUi{
		AskPrefix:       prefix,
		AskSecretPrefix: prefix,
		OutputPrefix:    prefix,
		InfoPrefix:      prefix,
		ErrorPrefix:     prefix,
		WarnPrefix:      prefix,
		Ui:              ui,
	}

	// check if release exists
	exists, err := releaseExists(region, release)
	if err != nil {
		ui.Error(fmt.Sprintf("%v", err))
	}
	// if it doesn't, do nothing and return success (this is mainly for multi-region terminate when
	// some regions may not have had a deployment)
	if !exists {
		ui.Info(fmt.Sprintf("release %s not found in region %s", release, region.Location))
		return 0
	}

	// check if we're trying to terminate a release that's running on prod
	isProd, err := checkIfProd(region, release)
	if err != nil {
		ui.Error(fmt.Sprintf("%v", err))
		return 1
	}
	// don't terminate if it's prod
	if isProd {
		ui.Error(fmt.Sprintf("%s is part of prod and cannot be terminated!", release))
		return 1
	}

	// if it's not, we can terminate
	ui.Info(fmt.Sprintf("terminating release %s for region %s", release, region.Location))
	if err := terminateRelease(region, release); err != nil {
		ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	ui.Info(fmt.Sprintf("terminated release %s", release))
	return 0
}

func terminateRelease(region Region, release string) error {
	// init client
	client := resources.NewGroupsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)
	// terminate
	rgName := releaseRGName(release, region.Location)
	resp, err := client.Delete(context.Background(), rgName)
	_ = resp

	if err != nil {
		return err
	}

	return nil
}

// checks if the load balancer running the given release is connected to the prod appgate
func checkIfProd(region Region, release string) (bool, error) {
	fmt.Printf("check prod: %s\n", release)

	muxIPToReleaseMap := make(map[string]string)

	prodHAproxyIPs, err := getHAProxyIPsForRegion(region, "prod")
	if err != nil {
		return false, err
	}

	releaseNames, err := getReleaseNamesFromRegion(region)
	if err != nil {
		return false, err
	}

	for _, releaseName := range releaseNames {
		prodMuxIPs, err := getUsermuxIPsForRelease(region, releaseName, "prod")
		if err != nil {
			return false, err
		}
		for _, muxIP := range prodMuxIPs {
			muxIPToReleaseMap[muxIP] = releaseName
		}
	}

	if len(prodHAproxyIPs) > 0 {
		// only check the first proxy in the set, they should all match
		releaseName, err := getReleaseNameFromHAProxyIP(prodHAproxyIPs[0], muxIPToReleaseMap)
		if err != nil {
			return false, err
		}
		if releaseName == release {
			return true, nil
		}
	}

	// if release isn't found in the pools, return false
	return false, nil
}

func releaseExists(region Region, release string) (bool, error) {
	// init client
	client := resources.NewGroupsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	// get resource group using release name
	rgName := releaseRGName(release, region.Location)
	_, err := client.Get(context.Background(), rgName)
	if err != nil {
		// check if the error is a 404
		if strings.Contains(err.Error(), "StatusCode=404") {
			return false, nil
		}
		// otherwise, return the error
		return false, err
	}

	// if no error, return true
	return true, nil
}
