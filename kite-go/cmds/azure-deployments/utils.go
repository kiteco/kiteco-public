package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

// uploadFile uploads a file to azure storage
//
// NOTE: data limit per block is 100mb - will error if trying to upload something bigger
func uploadFile(container *storage.Container, name string, data []byte) error {
	b := container.GetBlobReference(name)
	// if it exists, it will overwrite
	if err := b.CreateBlockBlob(nil); err != nil {
		return err
	}

	// the block just needs some base64 ID that's unique per request, probably
	blockID := base64.StdEncoding.EncodeToString([]byte("0"))
	err := b.PutBlock(blockID, data, nil)
	if err != nil {
		return err
	}

	list, err := b.GetBlockList(storage.BlockListTypeUncommitted, nil)
	if err != nil {
		return err
	}
	uncommittedBlocksList := make([]storage.Block, len(list.UncommittedBlocks))
	for i := range list.UncommittedBlocks {
		uncommittedBlocksList[i].ID = list.UncommittedBlocks[i].Name
		uncommittedBlocksList[i].Status = storage.BlockStatusUncommitted
	}

	// this commits the blocks or something
	err = b.PutBlockList(uncommittedBlocksList, nil)
	if err != nil {
		return err
	}
	return nil
}

// readFromTemplate renders a template file given by the file name and returns the resulting byte
// array
func readFromTemplate(templates *templateset.Set, fileName string, data interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := templates.RenderText(buf, fileName, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// findLBPool finds the load balancer backend address pool corresponding to the given pool name
func findLBPool(pools []network.BackendAddressPool, name string) (network.BackendAddressPool, error) {
	for _, pool := range pools {
		if *pool.Name == name {
			return pool, nil
		}
	}

	// if not found, return error
	return network.BackendAddressPool{}, fmt.Errorf("backend pool %s not found", name)
}

// getLBIP gets the IP address from the frontend ip config given the name
func getLBIP(lbfipc []network.FrontendIPConfiguration, name string) (string, error) {
	for _, fipc := range lbfipc {
		if *fipc.Name == name {
			return *fipc.PrivateIPAddress, nil
		}
	}

	// if not found, return error
	return "", fmt.Errorf("no IP address for lb fipc %s", name)
}

// checkIfReleaseInPool checks if the pool contains a scale set in the given release resource group
func checkIfReleaseInPool(pool network.BackendAddressPool, release string) bool {
	// false if no ip configs
	if pool.BackendIPConfigurations == nil {
		return false
	}

	// false if ip configs is empty
	ipcs := *pool.BackendIPConfigurations
	if len(ipcs) == 0 {
		return false
	}

	// return if there is an ip config ID that contains the name of the release resource group
	for _, ipc := range ipcs {
		return strings.Contains(*ipc.ID, release)
	}

	// if not found, return false
	return false
}

// sortedRegionNames returns an array of sorted region names
func sortedRegionNames(regions map[string]Region) []string {
	var regionNames []string
	for region := range regions {
		regionNames = append(regionNames, region)
	}
	sort.Strings(regionNames)

	return regionNames
}

// releaseRGName returns the azure resource group name based on the release and region name
func releaseRGName(release string, location string) string {
	return fmt.Sprintf("%s-%s", release, location)
}

func getUsermuxIPsForRelease(region Region, release string, environment string) ([]string, error) {
	var usermuxIPs []string
	//TODO: handle environment selection

	client := network.NewInterfacesClient(region.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	rgName := releaseRGName(release, region.Location)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ifaceList, err := client.ListVirtualMachineScaleSetNetworkInterfaces(ctx, rgName, "user-mux")
	if err != nil {
		return usermuxIPs, fmt.Errorf("no user-mux vmss found in %s in region %s", release, region.ResourceGroup)
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

func getHAProxyIPsForRegion(region Region, environment string) ([]string, error) {
	var haProxyIPs []string

	client := network.NewInterfacesClient(region.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	haProxyVMSSName := region.HAProxyVMSSNames[environment]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ifaceList, err := client.ListVirtualMachineScaleSetNetworkInterfaces(ctx, region.ResourceGroup, haProxyVMSSName)
	if err != nil {
		fmt.Printf("no haproxy vmss found in rg %s\n", region.ResourceGroup)
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

func getReleaseNamesFromRegion(region Region) ([]string, error) {
	var releaseNames []string

	groupsClient := resources.NewGroupsClient(region.SubscriptionID)
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(auth)

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

func runCommandOnHaproxy(haProxyIP string, command string) (string, error) {
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("ubuntu@%s", haProxyIP),
		fmt.Sprintf("sudo bash -c '. /root/haproxy-deploy-ctl.sh && %s'", command))
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

	cmdResult, err := runCommandOnHaproxy(haProxyIP, "haproxy_get_release_ips")
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

// getReleaseNameFromLB gets the release running on the load balancer.
// if the pools or ipcs are nil, returns blank string.
func getReleaseNameFromLB(lb *network.LoadBalancer, poolName string) (string, error) {
	// no pools
	if lb.BackendAddressPools == nil {
		return "", errNoRelease
	}

	pool, err := findLBPool(*lb.BackendAddressPools, poolName)
	// no scale set pool found
	if err != nil {
		return "", errNoRelease
	}

	// no ipcs
	if pool.BackendIPConfigurations == nil {
		return "", errNoRelease
	}

	ipcs := *pool.BackendIPConfigurations
	// emptyy ipcs
	if len(ipcs) == 0 {
		return "", errNoRelease
	}

	// load balancers with scale sets can only contain one scale set, so if the ipcs are not nil or
	// empy, grab the release name from the resource group part of the ipc's ID
	rg := getFieldFromID(*ipcs[0].ID, "resourceGroups")

	// release resourcee groups have region names suffixed, so remove suffix using split and join
	sep := "-"
	split := strings.Split(rg, sep)
	return strings.Join(split[:len(split)-1], ""), nil
}

// getFieldFromID parses an ID to return a field in the URI path of an ID
func getFieldFromID(id string, field string) string {
	exp := fmt.Sprintf("%s\\/([\\w-]+)\\/", field)
	re := regexp.MustCompile(exp)
	matches := re.FindStringSubmatch(id)
	// if smaller than two items, no submatch was found
	if len(matches) < 2 {
		return ""
	}

	// return first submatch
	return matches[1]
}
