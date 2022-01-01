package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type app struct {
	localDir  string
	remoteDir string
	hosts     []string
	used      map[string]bool
	m         sync.Mutex
	allRemote []string
}

func newApp(localDir, remoteDir string, hosts []string) *app {
	var names []string
	for _, host := range hosts {
		remote, err := remoteFiles(host, remoteDir)
		if err != nil {
			log.Printf("error listing files for %s/%s: %s\n", host, remoteDir, err)
			continue
		}
		for _, r := range remote {
			names = append(names, localFilename(host, r))
		}
	}

	log.Printf("got %d remote files: %v\n", len(names), names)

	return &app{
		localDir:  localDir,
		remoteDir: remoteDir,
		hosts:     hosts,
		used:      make(map[string]bool),
		allRemote: names,
	}
}

type usedRequest struct {
	Used []string `json:"used"`
}

func (a *app) handleList(w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal(a.allRemote)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) handleUsed(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	var req usedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	for _, f := range req.Used {
		a.used[f] = true
	}

	w.WriteHeader(http.StatusOK)
}

func (a *app) handleReset(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	// purge files in directory to avoid a race condition with the python reader
	log.Println("resetting")
	for _, f := range a.allRemote {
		path := filepath.Join(a.localDir, f)
		log.Printf("deleting %s for reset\n", path)

		tryRemove(path)
	}

	a.used = make(map[string]bool)

	w.WriteHeader(http.StatusOK)
}

func (a *app) IsUsed(host, file string) bool {
	a.m.Lock()
	defer a.m.Unlock()

	path := localFilename(host, file)
	return a.used[path]
}

func (a *app) DeleteLoop() {
	for {
		used := func() []string {
			a.m.Lock()
			defer a.m.Unlock()

			used := make([]string, len(a.used))
			for f := range a.used {
				used = append(used, f)
			}
			return used
		}()

		for _, f := range used {
			if len(f) == 0 {
				continue
			}

			f = filepath.Join(a.localDir, f)
			if _, err := os.Stat(f); err != nil {
				continue
			}

			log.Println("trying to remove used file", f)
			tryRemove(f)
		}
		sleep()
	}
}
