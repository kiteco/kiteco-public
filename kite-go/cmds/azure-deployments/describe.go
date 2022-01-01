package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/kiteco/kiteco/kite-golib/scalinggroups"
	"github.com/mitchellh/cli"
)

type describeCommand struct{}
type description struct {
	// NOTE: these fields are exported so we can json.Marshal them
	Name   string // Name is the name of the process
	Status string // Status is the status of the process
	Addr   net.IP // Addr is the IP address of the machine that the process is running on
	Region string // Region specifies which region this process belongs to
}

type byNameThenIP []description

func (xs byNameThenIP) Len() int      { return len(xs) }
func (xs byNameThenIP) Swap(i, j int) { xs[i], xs[j] = xs[j], xs[i] }
func (xs byNameThenIP) Less(i, j int) bool {
	if xs[i].Name == xs[j].Name {
		return bytes.Compare(xs[i].Addr, xs[j].Addr) < 0
	}
	return xs[i].Name < xs[j].Name
}

func (d *describeCommand) Help() string {
	return "run this command with a region and release and optional json format, e.g 'describe westus release_0 [json]'"
}

func (d *describeCommand) Synopsis() string {
	return "describe a deployment in a region"
}

func (d *describeCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 2 && len(args) != 3 {
		return cli.RunResultHelp
	}

	// read command line args
	region := Regions[args[0]]
	release := args[1]
	jsonFormat := false
	if len(args) == 3 && args[2] == "json" {
		jsonFormat = true
	}

	// get descriptions
	descriptions, err := getVMSSInfo(region, release)
	if err != nil {
		ui.Error(fmt.Sprintf("%s", err))
		return 1
	}

	// sort and print descriptions
	sort.Sort(byNameThenIP(descriptions))
	if jsonFormat {
		// if descrs is nil, set it to an empty array so the json displays `[]` vs `null` which
		// is a bit difficult to work with downstream
		if descriptions == nil {
			descriptions = []description{}
		}
		// print JSON
		buf, err := json.Marshal(descriptions)
		if err != nil {
			ui.Error(fmt.Sprintf("error marshalling description into JSON: %v", err))
			return 1
		}
		fmt.Println(string(buf))
	} else {
		// print human-readable
		fmt.Printf("region: %s, release: %s\n", region.Location, release)
		printDescriptions(descriptions)
	}

	return 0
}

func getVMSSInfo(region Region, release string) ([]description, error) {
	var descriptions []description
	//  init client
	client := network.NewInterfacesClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	// get network interfaces for the scale set instances in the release resource group
	for vmssName := range region.ScaleSets {
		rgName := releaseRGName(release, region.Location)
		l, err := client.ListVirtualMachineScaleSetNetworkInterfaces(context.Background(), rgName, vmssName)
		if err != nil {
			return descriptions, err
		}
		nis := l.Values()

		for _, ni := range nis {
			// nil pointer check
			if ni.IPConfigurations == nil {
				return descriptions, fmt.Errorf("nil pointer: ni.IPConfigurations")
			}
			// get IP from the IP config
			//
			// NOTE: there should only ever be exactly one IP config, but we iterate over them just in case
			for _, ipc := range *ni.IPConfigurations {
				// nil pointer check
				if ipc.PrivateIPAddress == nil {
					return descriptions, fmt.Errorf("nil pointer: ipc.PrivateIPAddress")
				}
				// call Status using the IP and add to description
				descriptions = append(descriptions, description{
					Name:   vmssName,
					Status: scalinggroups.Status(*ipc.PrivateIPAddress),
					Addr:   net.ParseIP(*ipc.PrivateIPAddress),
					Region: region.Location,
				})
			}
		}
	}

	return descriptions, nil
}

// NOTE: copied from aws deployments describe
func printDescriptions(descriptions []description) {
	buf := &bytes.Buffer{}
	writer := tabwriter.NewWriter(buf, 16, 2, 2, ' ', 0)
	for _, descr := range descriptions {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", descr.Name, descr.Addr.String(), descr.Status)
	}
	writer.Flush()
	fmt.Println(buf.String())
}
