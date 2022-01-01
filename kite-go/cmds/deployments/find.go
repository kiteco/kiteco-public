package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"regexp"
	"sort"
	"text/tabwriter"

	"github.com/goamz/goamz/ec2"
	"github.com/mitchellh/cli"
)

type foundItem struct {
	region  Region
	release string // name of release
	name    string // name of executable being run, e.g. "user-node"
	addr    net.IP
}

type byRegionReleaseName []foundItem

func (xs byRegionReleaseName) Len() int      { return len(xs) }
func (xs byRegionReleaseName) Swap(i, j int) { xs[i], xs[j] = xs[j], xs[i] }
func (xs byRegionReleaseName) Less(i, j int) bool {
	if xs[i].region != xs[j].region {
		return xs[i].region.AwsRegion.Name < xs[j].region.AwsRegion.Name
	}
	if xs[i].release != xs[j].release {
		return xs[i].release < xs[j].release
	}
	if xs[i].name != xs[j].name {
		return xs[i].name < xs[j].name
	}
	return bytes.Compare(xs[i].addr, xs[j].addr) < 0
}

type findCommand struct{}

func (l *findCommand) Help() string {
	return "run this command with the name of an executable, e.g 'user-node'"
}

func (l *findCommand) Synopsis() string {
	return "find instances running the given target"
}

func (l *findCommand) Run(args []string) int {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 1 {
		return cli.RunResultHelp
	}

	pattern, err := regexp.Compile(args[0])
	if err != nil {
		ui.Error(fmt.Sprintf("unable to parse query as regex: %v", err))
		return 1
	}

	var items []foundItem
	for awsRegion, region := range Regions {
		client := ec2.New(auth, awsRegion)

		filter := ec2.NewFilter()
		filter.Add("vpc-id", region.VpcID)
		resp, err := client.DescribeInstances(nil, filter)
		if err != nil {
			ui.Error(fmt.Sprintf("error querying instances in region: %s, vpc: %s: %s", awsRegion.Name, region.VpcID, err))
			return 1
		}

		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				addr := net.ParseIP(instance.PrivateIPAddress)
				if addr == nil {
					ui.Error(fmt.Sprintf("unable to parse %s as IP address", instance.PrivateIPAddress))
					return 1
				}

				item := foundItem{
					region: region,
					addr:   addr,
				}
				for _, tag := range instance.Tags {
					switch tag.Key {
					case "Name":
						item.name = tag.Value
					case "Release":
						item.release = tag.Value
					}
				}

				if pattern.MatchString(item.name) {
					items = append(items, item)
				}
			}
		}
	}

	sort.Sort(byRegionReleaseName(items))

	w := tabwriter.NewWriter(os.Stdout, 16, 2, 2, ' ', 0)
	for i, item := range items {
		if i > 0 && item.release != items[i-1].release {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", item.name, item.addr.String(), item.release, item.region.AwsRegion.Name)
	}
	w.Flush()

	return 0
}
