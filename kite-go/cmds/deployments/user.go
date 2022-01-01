package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/goamz/goamz/ec2"
	"github.com/mitchellh/cli"
)

type finduserCommand struct{}

func (u *finduserCommand) Help() string {
	return "find user-node instance supporting the given user name"
}

func (u *finduserCommand) Synopsis() string {
	return "find user-node instance supporting the given user name"
}

func (u *finduserCommand) Run(args []string) int {
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

				if strings.Contains(item.name, "user-node") {
					raw := addr.String() + ":9091/users"
					url, err := url.Parse(raw)
					if err != nil {
						ui.Error(fmt.Sprintf("unable to parse url %s: %v", raw, err))
						return 1
					}
					url.Scheme = "http"

					resp, err := http.Get(url.String())
					if err != nil {
						ui.Error(fmt.Sprintf("error getting %v: %v", url, err))
						continue
					}

					if resp.StatusCode != 200 {
						ui.Error(fmt.Sprintf("got response code %d getting %v, expected 200", resp.StatusCode, url))
						continue
					}
					defer resp.Body.Close()

					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						ui.Error(fmt.Sprintf("error reading body from %v: %v", url, err))
						continue
					}
					if pattern.MatchString(string(body)) {
						items = append(items, item)
					}
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
