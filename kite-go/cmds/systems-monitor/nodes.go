package main

import (
	"log"
	"sort"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/deployments"
)

// Node is a single backend node such as a machine or a process that exposes the set of metrics
// APIs that we can use to get various metrics. Note that node names must only contain
// alphanumerics, underscores, and hyphens.
type Node interface {
	fetchSources() error
	name() []string
	metrics() []*Metric
}

// getNodes gets all nodes and group nodes
func getNodes() ([]Node, []Node, error) {
	var nodes []Node
	var groups []Node

	// deployment nodes
	dn, dg, err := getDeploymentNodes()
	if err != nil {
		return nodes, groups, err
	}
	nodes = append(nodes, dn...)
	groups = append(nodes, dg...)

	return nodes, groups, nil
}

func getDeploymentNodes() ([]Node, []Node, error) {
	var nodes []Node
	var groups []Node

	// get deployments
	regions, err := deployments.Production()
	if err != nil {
		log.Printf("could not get deployments: %v", err)
		return nodes, groups, err
	}

	// some janky stuff to separate azure metrics
	// TODO: be less janky
	globalName := "global"
	if provider == "azure" {
		globalName = "azure-global"
	}
	// global deployment group
	globalDeps := deploymentGroup{Region: globalName}
	groups = append(groups, &globalDeps)

	for _, region := range regions {
		// deployment group for region
		regionDeps := deploymentGroup{Region: region.Region}
		groups = append(groups, &regionDeps)

		// create user node nodes
		sort.Strings(region.UsernodeIPs)
		for n, ip := range region.UsernodeIPs {
			node := &deployment{
				Region: region.Region,
				Type:   "user-node",
				IP:     ip,
				Num:    strconv.Itoa(n + 1), // we use numbered labels for each instance for consistent metric names
			}
			nodes = append(nodes, Node(node))
			globalDeps.deployments = append(globalDeps.deployments, node)
			regionDeps.deployments = append(regionDeps.deployments, node)
		}
		// create user mux nodes
		sort.Strings(region.UserMuxIPs)
		for n, ip := range region.UserMuxIPs {
			node := &deployment{
				Region: region.Region,
				Type:   "user-mux",
				IP:     ip,
				Num:    strconv.Itoa(n + 1), // we use numbered labels for each instance for consistent metric names
			}
			nodes = append(nodes, Node(node))
			globalDeps.deployments = append(globalDeps.deployments, node)
			regionDeps.deployments = append(regionDeps.deployments, node)
		}
		// create local code worker nodes
		sort.Strings(region.LocalCodeWorkerIPs)
		for n, ip := range region.LocalCodeWorkerIPs {
			node := &deployment{
				Region: region.Region,
				Type:   "local-code-worker",
				IP:     ip,
				Num:    strconv.Itoa(n + 1), // we use numbered labels for each instance for consistent metric names
			}
			nodes = append(nodes, Node(node))
			globalDeps.deployments = append(globalDeps.deployments, node)
			regionDeps.deployments = append(regionDeps.deployments, node)
		}
	}

	return nodes, groups, nil
}
