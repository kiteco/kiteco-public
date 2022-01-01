package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

func getClusterIPs(clusterName string) ([]string, error) {
	initAzureCreds()

	var ips []string

	client := network.NewInterfacesClient(conf.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	rgName := resourceGroupName(clusterName)
	ifaceList, err := client.ListVirtualMachineScaleSetNetworkInterfaces(
		context.Background(), rgName, clusterName)
	if err != nil {
		return nil, err
	}

	ifaces := ifaceList.Values()

	for _, iface := range ifaces {
		if iface.IPConfigurations == nil {
			return nil, fmt.Errorf("nil pointer: iface.IPConfigurations")
		}

		for _, ipconfig := range *iface.IPConfigurations {
			if ipconfig.PrivateIPAddress == nil {
				return nil, fmt.Errorf("nil pointer: ipconfig.PrivateIPAddress")
			}

			ips = append(ips, *ipconfig.PrivateIPAddress)
		}
	}

	sort.Strings(ips)

	return ips, nil
}

func listClusters() ([]string, error) {
	initAzureCreds()

	client := resources.NewGroupsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	res, err := client.ListComplete(
		context.Background(),
		fmt.Sprintf("tagname eq '%s'", clusterTag),
		nil)
	if err != nil {
		return nil, err
	}

	var clusters []string

	for res.NotDone() {
		group := res.Value()
		clusters = append(clusters, *group.Tags[clusterTag])

		if err := res.Next(); err != nil {
			return nil, err
		}
	}

	return clusters, nil
}
