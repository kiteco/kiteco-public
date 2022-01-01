package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"syscall"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

// azDepInfo stores deployment info about a region necessary for interacting with the deployment
type azDepInfo struct {
	azDepInfoCommon
	Location string
	rg       string
}

// azDepInfoCommon stores deployment info that is common across all regions
type azDepInfoCommon struct {
	agprod           string
	agstaging        string
	agpool           string
	lbpool           string
	lbnames          []string
	lbfipcname       string
	SubscriptionID   string
	HAProxyVMSSNames map[string]string
}

var (
	// source of these values are kite-go/cmds/azure-deployments/regions.go (which is sourced from
	// terraform)
	common = azDepInfoCommon{
		agprod:         "agw-prod",
		agstaging:      "agw-staging",
		agpool:         "lbpool",
		lbpool:         "deployments-pool",
		lbnames:        []string{"deployments-0", "deployments-1", "deployments-2"},
		lbfipcname:     "deployments-frontend",
		SubscriptionID: "XXXXXXX",
		HAProxyVMSSNames: map[string]string{
			"staging": "haproxy-staging",
			"prod":    "haproxy-prod",
		},
	}
	azureRegionInfo = map[string]azDepInfo{
		"westus2": azDepInfo{
			azDepInfoCommon: common,
			Location:        "westus2",
			rg:              "prod-westus2-0",
		},
		"eastus": azDepInfo{
			azDepInfoCommon: common,
			Location:        "eastus",
			rg:              "prod-eastus-0",
		},
		"westeurope": azDepInfo{
			azDepInfoCommon: common,
			Location:        "westeurope",
			rg:              "prod-westeurope-0",
		},
	}
)

// NOTE: the following is a slightly modified copy of util functions in
// kite-go/cmds/azure-deployments, along with all the necessarily utility functions, also slightly
// modified

func getHAProxyIPsForRegion(region azDepInfo, environment string) ([]string, error) {
	var haProxyIPs []string

	client := network.NewInterfacesClient(region.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(azureAuth)

	haProxyVMSSName := region.HAProxyVMSSNames[environment]

	ifaceList, err := client.ListVirtualMachineScaleSetNetworkInterfaces(context.Background(), region.rg, haProxyVMSSName)
	if err != nil {
		fmt.Printf("no haproxy vmss found in rg %s\n", region.rg)
	}
	ifaces := ifaceList.Values()

	for _, iface := range ifaces {
		if iface.IPConfigurations == nil {
			return haProxyIPs, fmt.Errorf("nil pointer: iface.IPConfigurations")
		}
		for _, ipconfig := range *iface.IPConfigurations {
			if ipconfig.PrivateIPAddress == nil {
				return haProxyIPs, fmt.Errorf("nil pointer: ipconfig.PrivateIPAddress")
			}

			haProxyIPs = append(haProxyIPs, *ipconfig.PrivateIPAddress)
		}
	}

	return haProxyIPs, nil
}

func getReleaseNamesFromRegion(region azDepInfo) ([]string, error) {
	var releaseNames []string

	groupsClient := resources.NewGroupsClient(region.SubscriptionID)
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(azureAuth)

	// if we have groups to look at
	for list, err := groupsClient.ListComplete(context.Background(), "", nil); list.NotDone(); err = list.Next() {
		if err != nil {
			return releaseNames, err
		}
		group := list.Value()
		if strings.HasPrefix(*group.Name, "release_") && (*group.Location == region.Location) {
			releaseName := strings.Split(*group.Name, "-")[0]
			releaseNames = append(releaseNames, releaseName)
		}
	}

	return releaseNames, nil
}

func getUsermuxIPsForRelease(region azDepInfo, release string, environment string) ([]string, error) {
	var usermuxIPs []string

	client := network.NewInterfacesClient(region.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(azureAuth)

	rgName := releaseRGName(release, region.Location)
	ifaceList, err := client.ListVirtualMachineScaleSetNetworkInterfaces(context.Background(), rgName, "user-mux")
	if err != nil {
		return usermuxIPs, fmt.Errorf("no user-mux vmss found in %s in region %s", release, region.rg)
	}
	ifaces := ifaceList.Values()

	for _, iface := range ifaces {
		if iface.IPConfigurations == nil {
			return usermuxIPs, fmt.Errorf("nil pointer: iface.IPConfigurations")
		}
		for _, ipconfig := range *iface.IPConfigurations {
			if ipconfig.PrivateIPAddress == nil {
				return usermuxIPs, fmt.Errorf("nil pointer: ipconfig.PrivateIPAddress")
			}

			usermuxIPs = append(usermuxIPs, *ipconfig.PrivateIPAddress)
		}
	}

	return usermuxIPs, nil
}

func runCommandOnHaproxy(haProxyIP string, command string) (string, error) {
	cmd := exec.Command("ssh", "-v", "-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("ubuntu@%s", haProxyIP), command)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("haproxy ssh error: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return "", fmt.Errorf("haproxy ssh failed with code: %d\n\nPlease ensure you can access %s and have the proper keys added to your ssh-agent\n\nTo add the key, run ssh-add <keyfile>", status.ExitStatus(), haProxyIP)
			}
		} else {
			return "", fmt.Errorf("haproxy ssh error: %v", err)
		}
	}

	return out.String(), nil
}

func getReleaseNameFromHAProxyIP(haProxyIP string, releaseMap map[string]string) (string, error) {
	// releaseMap is a map of string IP to string ReleaseName
	// TODO: refactor command to not use my hard coded key path
	var haproxyStatusResponse map[string][]string

	cmdResult, err := runCommandOnHaproxy(haProxyIP, "sudo /root/haproxy_get_release_ips.sh")
	if err != nil {
		return "", err
	}

	jsonDecoder := json.NewDecoder(strings.NewReader(cmdResult))
	err = jsonDecoder.Decode(&haproxyStatusResponse)
	if err != nil {
		return "", err
	}

	muxIPList := haproxyStatusResponse["muxIPs"]
	if len(muxIPList) == 0 {
		return "", nil
	}
	releaseName := releaseMap[muxIPList[0]]
	return releaseName, nil
}

// getAzureRelease returns the release name running on the appgate
func getAzureRelease(target string, region azDepInfo) (string, error) {

	muxIPToReleaseMap := make(map[string]string)

	haProxyIPs, err := getHAProxyIPsForRegion(region, target)
	if err != nil {
		return "", err
	}

	releaseNames, err := getReleaseNamesFromRegion(region)
	if err != nil {
		return "", err
	}

	for _, releaseName := range releaseNames {
		muxIPs, err := getUsermuxIPsForRelease(region, releaseName, target)
		if err != nil {
			log.Println(err)
		} else {
			// fill map of mux IP -> release name so we can look up the release from haproxy status
			for _, muxIP := range muxIPs {
				muxIPToReleaseMap[muxIP] = releaseName
			}
		}
	}

	if len(haProxyIPs) > 0 {
		// only check the first proxy in the set, they should all match
		releaseName, err := getReleaseNameFromHAProxyIP(haProxyIPs[0], muxIPToReleaseMap)
		if err != nil {
			return "", err
		}
		if releaseName != "" {
			return releaseName, nil
		}
	}
	return "", fmt.Errorf("%s release not found on %s", target, region.rg)
}

// az IPs returns a list of private IPs for the given process and release in a region
//
// NOTE: this is a modified version of scalinggroups.IPs
func azIPs(process, release, region string) ([]string, error) {
	var ips []string

	client := network.NewInterfacesClient(azureSubID)
	client.Authorizer = autorest.NewBearerAuthorizer(azureAuth)

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
