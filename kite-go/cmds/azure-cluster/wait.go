package main

import (
	"log"
	"sort"
	"time"
)

const (
	maxAttempts = 20
)

func wait(clusterName string) {
	ips, err := getClusterIPs(clusterName)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("waiting for %d instances in cluster %s", len(ips), clusterName)

	sort.Strings(ips)
	for idx, ip := range ips {
		var attempt int
		for {
			if err := runRemoteCmd(ip, "ls /var/kite/provisioned > /dev/null"); err == nil {
				log.Printf("(%d/%d) %s \xE2\x9C\x94", idx+1, len(ips), ip)
				break
			}
			attempt++
			if attempt >= maxAttempts {
				log.Fatalf("made %d attempts to ssh into %s, aborting", maxAttempts, ip)
			}
			time.Sleep(5 * time.Second)
		}
	}

	log.Printf("cluster %s is ready", clusterName)
}
