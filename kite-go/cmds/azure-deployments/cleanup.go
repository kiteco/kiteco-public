package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mitchellh/cli"
)

type cleanupCommand struct{}

func (l *cleanupCommand) Help() string {
	return ""
}

func (l *cleanupCommand) Synopsis() string {
	return ""
}

func (l *cleanupCommand) Run(args []string) int {
	// init cli ui
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 1 {
		return cli.RunResultHelp
	}

	// read command line args
	region, ok := Regions[args[0]]
	if !ok {
		ui.Error(fmt.Sprintf("region %s does not exist", args[0]))
	}

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

	ui.Info("checking for idle releases")
	idleReleases, err := getIdleReleases(region)

	if err != nil {
		ui.Error(fmt.Sprintf("error getting idle releases: %v", err))
		return 1
	}

	if len(idleReleases) == 0 {
		ui.Info("no idle release found")
		return 0
	}

	for _, release := range idleReleases {
		ui.Info(fmt.Sprintf("terminating %s... you have 5 seconds to stop this...", release))
		time.Sleep(5 * time.Second)
		ui.Info(fmt.Sprintf("terminating %s", release))
		if err := terminateRelease(region, release); err != nil {
			ui.Error(fmt.Sprintf("error terminating release %s: %v", release, err))
			return 1
		}
	}

	return 0
}

// getIdleReleases gets a list of release names that are idle
func getIdleReleases(region Region) ([]string, error) {
	var idleReleases []string
	var stagingReleaseName string
	var prodReleaseName string

	muxIPToReleaseMap := make(map[string]string)

	stagingHAproxyIPs, err := getHAProxyIPsForRegion(region, "staging")
	if err != nil {
		return nil, err
	}

	prodHAproxyIPs, err := getHAProxyIPsForRegion(region, "prod")
	if err != nil {
		return nil, err
	}

	releaseNames, err := getReleaseNamesFromRegion(region)
	if err != nil {
		return nil, err
	}

	for _, releaseName := range releaseNames {
		stageMuxIPs, err := getUsermuxIPsForRelease(region, releaseName, "staging")
		if err != nil {
			fmt.Printf("error fetching staging usermux IPs for %s in region %s\n", releaseName, region.Location)
		}

		prodMuxIPs, err := getUsermuxIPsForRelease(region, releaseName, "prod")
		if err != nil {
			fmt.Printf("error fetching prod usermux IPs for %s in region %s\n", releaseName, region.Location)
		}
		// fill map of mux IP -> release name so we can look up the release from haproxy status
		for _, muxIP := range stageMuxIPs {
			muxIPToReleaseMap[muxIP] = releaseName
		}
		for _, muxIP := range prodMuxIPs {
			muxIPToReleaseMap[muxIP] = releaseName
		}
	}

	if len(stagingHAproxyIPs) > 0 {
		// only check the first proxy in the set, they should all match
		stagingReleaseName, err = getReleaseNameFromHAProxyIP(stagingHAproxyIPs[0], muxIPToReleaseMap)
		if err != nil {
			return nil, err
		}
	}

	if len(prodHAproxyIPs) > 0 {
		// only check the first proxy in the set, they should all match
		prodReleaseName, err = getReleaseNameFromHAProxyIP(prodHAproxyIPs[0], muxIPToReleaseMap)
		if err != nil {
			return nil, err
		}
	}

	for _, releaseName := range releaseNames {
		if releaseName != stagingReleaseName && releaseName != prodReleaseName {
			idleReleases = append(idleReleases, releaseName)
		}
	}

	return idleReleases, nil
}
