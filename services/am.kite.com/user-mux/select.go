package main

import (
	"math/rand"
)

// selectRandomTarget selects a healthy target at random
func selectRandomTarget(targets []*proxyTarget) (*proxyTarget, bool) {
	var healthy []*proxyTarget
	for _, target := range targets {
		if target.isHealthy() {
			healthy = append(healthy, target)
		}
	}

	if len(healthy) == 0 {
		return nil, false
	}

	return healthy[rand.Intn(len(healthy))], true
}
