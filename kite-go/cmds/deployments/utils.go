package main

import (
	"github.com/goamz/goamz/ec2"
)

func instanceIds(instances []ec2.Instance) []string {
	var ids []string
	for _, instance := range instances {
		ids = append(ids, instance.InstanceId)
	}
	return ids
}

func instanceIps(instances []ec2.Instance) []string {
	var ids []string
	for _, instance := range instances {
		ids = append(ids, instance.PrivateIPAddress)
	}
	return ids
}
