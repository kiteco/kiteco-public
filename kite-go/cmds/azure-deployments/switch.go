package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
)

var errNoVMSS = errors.New("vmss not found")
var errSameIP = errors.New("new IP is same as old IP")

type switchCommand struct{}

func (d *switchCommand) Help() string {
	return "run this command with a region, release, and appgate, e.g 'switch westus release_0 staging'"
}

func (d *switchCommand) Synopsis() string {
	return "switch the appgate to point to the deployment with the specified release and region"
}

func (d *switchCommand) Run(args []string) int {
	// init cli ui
	var ui cli.Ui
	var haProxyIPs []string
	var err error
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 3 {
		return cli.RunResultHelp
	}

	// read command line args
	region, ok := Regions[args[0]]
	if !ok {
		ui.Error(fmt.Sprintf("region %s does not exist", args[0]))
	}
	release := args[1]
	environment := args[2]

	fmt.Printf("%s env selected\n", environment)
	if environment != "prod" && environment != "staging" {
		ui.Error(fmt.Sprintf("%s is not a valid environment, must be one of staging,prod", environment))
		return 1
	}
	haProxyIPs, err = getHAProxyIPsForRegion(region, environment)
	if err != nil {
		ui.Error(fmt.Sprintf("error fetching %s haproxy IPs: %v", environment, err))
		return 1
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

	// check that release is ready
	descriptions, err := getVMSSInfo(region, release)
	if err != nil {
		ui.Error(fmt.Sprintf("error checking release readiness: %v", err))
	}

	var muxIPStrings []string
	for _, desc := range descriptions {
		if desc.Status != "ready" {
			ui.Error(fmt.Sprintf("release %s is not ready; aborting", release))
			return 1
		}
		if desc.Name == "user-mux" {
			muxIPStrings = append(muxIPStrings, fmt.Sprintf("%s", desc.Addr))
		}
	}

	for _, haProxyIP := range haProxyIPs {
		ui.Info(fmt.Sprintf("Switching %s to %s in region %s", haProxyIP, release, region.Location))
		err = switchHAProxyIP(region, haProxyIP, release, muxIPStrings)
		if err != nil {
			ui.Error(fmt.Sprintf("error swapping haproxy release %s: %v", haProxyIP, err))
		}
	}

	return 0

}

func switchHAProxyIP(region Region, HAProxyIP string, release string, deployIPStrings []string) error {
	// convert IP list to comma delimited string
	deployIPString := strings.Join(deployIPStrings, ",")

	cmdResult, err := runCommandOnHaproxy(HAProxyIP, fmt.Sprintf("haproxy_swap_deploy %s", deployIPString))
	if err != nil {
		fmt.Printf("RES: %s", cmdResult)
		return fmt.Errorf("error swapping haproxy deploy: %v", err)
	}

	return nil
}
