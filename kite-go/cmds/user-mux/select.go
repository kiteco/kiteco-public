package main

import (
	"fmt"
	"math/rand"

	spooky "github.com/dgryski/go-spooky"
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

type scoreAndIndex struct {
	index int
	score uint64
}

// selectTargetForUUID selects a healthy target using renzdevous hashing
// it mirrors selectTargetForUID, but with a string input
func selectTargetForUUID(targets []*proxyTarget, uuid string) (*proxyTarget, bool) {
	var hit bool
	var max scoreAndIndex
	for idx, t := range targets {
		if !t.isHealthy() {
			continue
		}

		buf := []byte(fmt.Sprintf("%s%s", t.target.String(), uuid))
		score := spooky.Hash64(buf)
		if (max == scoreAndIndex{} || score > max.score) {
			hit = true
			max = scoreAndIndex{
				index: idx,
				score: score,
			}
		}
	}

	if hit {
		return targets[max.index], true
	}

	return nil, false
}
