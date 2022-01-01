package localcode

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/scalinggroups"
)

const (
	workerGroupPollInterval = time.Minute
	workerGroupName         = "local-code-worker"
)

// workerGroup is used to help shard and construct urls to local-code-workers. It will
// poll the autoscaling group periodically to discover the ips and if any nodes have been
// added or removed.
type workerGroup struct {
	port    string
	release string

	m   sync.Mutex
	ips []string
}

func newWorkerGroup() *workerGroup {
	port := os.Getenv("LOCAL_WORKER_ENDPOINT")
	release := os.Getenv("RELEASE")

	s := &workerGroup{}
	if release == "" || port == "" {
		s.ips = []string{"127.0.0.1"}
		s.port = "9080"
	} else {
		s.release = release
		s.port = strings.TrimPrefix(port, ":")
		go s.loop()
	}

	return s
}

func newWorkerGroupHostPort(hostPort string) (*workerGroup, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}

	return &workerGroup{
		ips:  []string{host},
		port: port,
	}, nil
}

func (s *workerGroup) len() int {
	s.m.Lock()
	defer s.m.Unlock()
	return len(s.ips)
}

func (s *workerGroup) shard(uid int64) int {
	s.m.Lock()
	defer s.m.Unlock()
	return int(uid % int64(len(s.ips)))
}

func (s *workerGroup) url(idx int, path string) (*url.URL, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if len(s.ips) == 0 {
		return nil, fmt.Errorf("no servers")
	}

	if idx >= len(s.ips) {
		return nil, fmt.Errorf("index exceeds number of servers")
	}

	ip := s.ips[idx]

	u, err := url.Parse(fmt.Sprintf("http://%s:%s%s", ip, s.port, path))
	return u, err
}

// --

func (s *workerGroup) loop() {
	ticker := time.NewTicker(workerGroupPollInterval)
	defer ticker.Stop()

	update := func() {
		ips, err := scalinggroups.IPs(workerGroupName, s.release)
		if err != nil {
			log.Println(err)
			return
		}
		sort.Strings(ips)

		s.m.Lock()
		defer s.m.Unlock()
		if !reflect.DeepEqual(ips, s.ips) {
			log.Printf("localcode.workerGroup: discovered local-code-workers %s", ips)
		}
		s.ips = ips
	}

	update()
	for range ticker.C {
		update()
	}
}

// --
