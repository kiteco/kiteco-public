package scalinggroups

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	debugPort = "9091"
	timeout   = 10 * time.Second
)

// Status sends requests to the /health and /ready debug endpoints of the scaling groups and
// returns their current status.
func Status(ipaddr string) string {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: timeout, // using default via http.DefaultTransport
			}).Dial,
		},
	}

	// check health
	resp, err := client.Get(fmt.Sprintf("http://%s:%s/health", ipaddr, debugPort))
	if err != nil {
		return "down"
	}
	if resp.StatusCode != 200 {
		return fmt.Sprintf("/health returned %d", resp.StatusCode)
	}

	// check ready
	resp, err = client.Get(fmt.Sprintf("http://%s:%s/ready", ipaddr, debugPort))
	if err != nil {
		return "loading"
	} else if resp.StatusCode != 200 {
		return "loading"
	}

	// if all endpoints are good, return ready
	return "ready"
}
