package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/mitchellh/cli"
)

type listCommand struct{}

type deploymentInfo struct {
	prod      string
	staging   string
	available []string
}

var errNoRelease = errors.New("load balancer has no active release")

func (l *listCommand) Help() string {
	return "list takes no arguments"
}

func (l *listCommand) Synopsis() string {
	return "list all deployments in all regions"
}

func (l *listCommand) Run(args []string) int {
	// init cli ui
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	// map each region to map of appgates/available to deployments
	regionInfo := make(map[string]deploymentInfo)
	var regions []string
	for regionKey := range Regions {
		regions = append(regions, regionKey)
	}
	sort.Strings(regions)

	for _, regionKey := range regions {
		region := Regions[regionKey]
		// get deployment info for region
		deploymentInfo, err := getDeploymentInfo(region)
		if err != nil {
			ui.Error(fmt.Sprintf("error getting info for region %s: %v", region.Location, err))
			return 1
		}
		regionInfo[region.Location] = deploymentInfo
	}

	// print regionInfo
	fmt.Println(formatInfo(regionInfo))
	return 0
}

// getDeploymentInfo returns a map of releases on prod and staging as well as all available
// releases
func getDeploymentInfo(region Region) (deploymentInfo, error) {
	var info deploymentInfo
	muxIPToReleaseMap := make(map[string]string)

	stagingHAproxyIPs, err := getHAProxyIPsForRegion(region, "staging")
	if err != nil {
		return info, err
	}

	prodHAproxyIPs, err := getHAProxyIPsForRegion(region, "prod")
	if err != nil {
		return info, err
	}

	releaseNames, err := getReleaseNamesFromRegion(region)
	if err != nil {
		return info, err
	}
	info.available = releaseNames

	for _, releaseName := range releaseNames {
		stageMuxIPs, err := getUsermuxIPsForRelease(region, releaseName, "staging")
		if err != nil {
			// log error here and continue on
			// this case can occur when we're in the middle of cleaning
			// up a region and the resource group still exists but the
			// user-mux vmss has already been deleted.
			log.Println(err)
		} else {
			// fill map of mux IP -> release name so we can look up the release from haproxy status
			for _, muxIP := range stageMuxIPs {
				muxIPToReleaseMap[muxIP] = releaseName
			}
		}

		prodMuxIPs, err := getUsermuxIPsForRelease(region, releaseName, "prod")
		if err != nil {
			log.Println(err)
		} else {
			for _, muxIP := range prodMuxIPs {
				muxIPToReleaseMap[muxIP] = releaseName
			}
		}
	}

	if len(stagingHAproxyIPs) > 0 {
		// only check the first proxy in the set, they should all match
		releaseName, err := getReleaseNameFromHAProxyIP(stagingHAproxyIPs[0], muxIPToReleaseMap)
		if err != nil {
			return info, err
		}
		if releaseName != "" {
			info.staging = releaseName
		}
	}

	if len(prodHAproxyIPs) > 0 {
		// only check the first proxy in the set, they should all match
		releaseName, err := getReleaseNameFromHAProxyIP(prodHAproxyIPs[0], muxIPToReleaseMap)
		if err != nil {
			return info, err
		}
		if releaseName != "" {
			info.prod = releaseName
		}
	}

	return info, nil
}

// formatInfo formats the info map
func formatInfo(info map[string]deploymentInfo) string {
	buf := &bytes.Buffer{}
	for region, info := range info {
		fmt.Fprintf(buf, "region: %s\n", region)
		fmt.Fprintf(buf, "\tprod: %s\n", info.prod)
		fmt.Fprintf(buf, "\tstaging: %s\n", info.staging)
		fmt.Fprintf(buf, "\tavailable:\n\t\t%s\n", strings.Join(info.available, "\n\t\t"))
	}
	return strings.TrimSpace(buf.String())
}
