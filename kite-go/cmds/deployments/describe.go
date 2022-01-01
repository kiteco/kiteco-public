package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/ec2"
	"github.com/kiteco/kiteco/kite-golib/scalinggroups"
	"github.com/mitchellh/cli"
)

type describeCommand struct{}

func (l *describeCommand) Help() string {
	return "run this command with a region and release, e.g 'deployments describe [region] [release]'"
}

func (l *describeCommand) Synopsis() string {
	return "describe a deployment in a region"
}

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

func (l *describeCommand) Run(args []string) int {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 2 && len(args) != 3 {
		return cli.RunResultHelp
	}

	region, release := args[0], args[1]
	jsonFormat := false
	if len(args) == 3 && args[2] == "json" {
		jsonFormat = true
	}

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

	var descrs []description
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			var name string
			for _, tag := range instance.Tags {
				switch tag.Key {
				case "Name":
					name = tag.Value
				}
			}
			var status string
			switch name {
			case "user-node", "local-code-worker", "user-mux", "sandbox-node":
				status = scalinggroups.Status(instance.PrivateIPAddress)
			}
			addr := net.ParseIP(instance.PrivateIPAddress)
			if addr == nil {
				ui.Error(fmt.Sprintf("unable to parse %s as IP address", instance.PrivateIPAddress))
				return 1
			}
			descrs = append(descrs, description{
				Name:   name,
				Addr:   addr,
				Status: status,
				Region: region,
			})
		}
	}

	// sort and print descriptions
	sort.Sort(byNameThenIP(descrs))
	if jsonFormat {
		// if descrs is nil, set it to an empty array so the json displays `[]` vs `null` which
		// is a bit difficult to work with downstream
		if descrs == nil {
			descrs = []description{}
		}
		// print JSON
		buf, err := json.Marshal(descrs)
		if err != nil {
			ui.Error(fmt.Sprintf("error marshalling description into JSON: %v", err))
			return 1
		}
		fmt.Println(string(buf))
	} else {
		// print human-readable
		fmt.Printf("region: %s, release: %s\n", awsRegion.Name, release)
		printDescriptions(descrs)
	}

	return 0
}

func printDescriptions(descriptions []description) {
	buf := &bytes.Buffer{}
	writer := tabwriter.NewWriter(buf, 16, 2, 2, ' ', 0)
	for _, descr := range descriptions {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", descr.Name, descr.Addr.String(), descr.Status)
	}
	writer.Flush()
	fmt.Println(buf.String())
}
