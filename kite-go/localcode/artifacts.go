package localcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/diskcache"
	"github.com/kiteco/kiteco/kite-golib/gziphttp"
)

type artifactClient struct {
	workers *workerGroup
	client  *http.Client
}

func newArtifactClient(workers *workerGroup) *artifactClient {
	return &artifactClient{
		workers: workers,
		client: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second, // using default via http.DefaultTransport
				}).Dial,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ResponseHeaderTimeout: 5 * time.Minute,
			},
		},
	}
}

func (a *artifactClient) getReader(artifact artifact, name string) (io.ReadCloser, error) {
	shard := a.workers.shard(artifact.UserID)
	getURL, err := a.workers.url(shard, fmt.Sprintf("/artifacts/%s/%s", artifact.UUID, name))
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Get(getURL.String())
	if err != nil {
		return nil, err
	}

	artifactStatusBreakdown.HitAndAdd(fmt.Sprintf("%d", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		io.Copy(ioutil.Discard, resp.Body)
		err := resp.Body.Close()
		if err != nil {
			log.Printf("error closing artifact body after status code %d: %s", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("got unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (a *artifactClient) findArtifact(input userMachineFile) (artifact, error) {
	var req findArtifactsRequest
	req = append(req, input)

	i := a.workers.shard(input.UserID)

	findURL, err := a.workers.url(i, "/artifacts/find-matching")
	if err != nil {
		log.Printf("error creating find-matching url for shard %d: %s", i, err)
		return artifact{}, err
	}

	buf, err := json.Marshal(req)
	if err != nil {
		log.Println("error marshalling find-matching request:", err)
		return artifact{}, err
	}

	resp, err := a.client.Post(findURL.String(), "application/json", bytes.NewBuffer(buf))
	if err != nil {
		log.Println("error sending find-matching POST:", err)
		return artifact{}, err
	}
	defer resp.Body.Close()

	var far findArtifactsResponse
	err = json.NewDecoder(resp.Body).Decode(&far)
	if err != nil {
		log.Println("error decoding response:", err)
		return artifact{}, err
	}

	for _, artifact := range far {
		return artifact, nil
	}

	return artifact{}, ErrNoArtifact("")
}

// --

var (
	artifactExpirationDuration = time.Minute * 30
)

type artifactServer struct {
	cache *diskcache.Cache

	rw      sync.RWMutex
	uuidMap map[string]*artifact
}

func newArtifactServer(cache *diskcache.Cache) *artifactServer {
	return &artifactServer{
		cache:   cache,
		uuidMap: make(map[string]*artifact),
	}
}

func (a *artifactServer) setupRoutes(mux *mux.Router) {
	mux.HandleFunc("/artifacts/find-matching", gziphttp.Wrap(a.handleFindArtifacts)).Methods("POST")
	mux.HandleFunc("/artifacts/{uuid}/{name}", a.handleServeArtifact).Methods("GET")
}

type artifact struct {
	UUID              string
	UserID            int64
	Machine           string
	Root              string
	Language          lang.Language
	Files             []string
	IndexedPathHashes map[string]bool
	LatestFileUpdate  time.Time
	Error             string

	publishedAt time.Time
	accessedAt  time.Time
}

func (a artifact) err() bool {
	return a.Error != ""
}

func (a artifact) subsumes(b artifact) bool {
	return a.UserID == b.UserID && a.Machine == b.Machine && reflect.DeepEqual(a.IndexedPathHashes, b.IndexedPathHashes)
}

func (a artifact) contains(name string) bool {
	return a.containsFile(name) || a.containsDir(name)
}

func (a artifact) containsFile(name string) bool {
	if lang.FromFilename(name) == a.Language {
		return strings.HasPrefix(name, a.Root) &&
			(a.IndexedPathHashes == nil || a.IndexedPathHashes[filePathHash(name)])
	}
	return false
}

func (a artifact) containsDir(name string) bool {
	if strings.HasPrefix(name, a.Root) {
		return a.IndexedPathHashes == nil || a.IndexedPathHashes[filePathHash(name)] ||
			a.IndexedPathHashes[filePathHash(withTrailingSlash(name))]
	}
	return false
}

func (a *artifactServer) publishArtifact(artifact artifact) {
	a.rw.Lock()
	defer a.rw.Unlock()

	artifact.publishedAt = time.Now()
	artifact.accessedAt = time.Now()
	for id, existing := range a.uuidMap {
		if artifact.subsumes(*existing) {
			delete(a.uuidMap, id)
		}
		if time.Since(existing.accessedAt) > artifactExpirationDuration {
			delete(a.uuidMap, id)
		}
	}

	a.uuidMap[artifact.UUID] = &artifact
}

func (a *artifactServer) artifactFor(umf userMachineFile) (*artifact, bool) {
	a.rw.RLock()
	defer a.rw.RUnlock()

	var newest *artifact
	for _, artifact := range a.uuidMap {
		if umf.UserID == artifact.UserID && umf.Machine == artifact.Machine && artifact.contains(umf.Filename) {
			if newest == nil || artifact.publishedAt.After(newest.publishedAt) {
				newest = artifact
			}
		}
	}

	return newest, newest != nil
}

type findArtifactsRequest []userMachineFile

type findArtifactsResponse map[string]artifact

func (a *artifactServer) handleFindArtifacts(w http.ResponseWriter, r *http.Request) {
	defer findMatchingDuration.DeferRecord(time.Now())

	var req findArtifactsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make(findArtifactsResponse)
	for _, umf := range req {
		if artifact, ok := a.artifactFor(umf); ok {
			resp[artifact.UUID] = *artifact
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&resp)
}

func (a *artifactServer) handleServeArtifact(w http.ResponseWriter, r *http.Request) {
	defer handleArtifactDuration.DeferRecord(time.Now())

	vars := mux.Vars(r)
	uuid, name := vars["uuid"], vars["name"]

	a.rw.RLock()
	if artifact, ok := a.uuidMap[uuid]; ok {
		artifact.accessedAt = time.Now()
	}
	a.rw.RUnlock()

	key := artifactKey(uuid, name)
	reader, err := a.cache.GetReader(key)
	if err != nil {
		if err == diskcache.ErrNoSuchKey {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, reader)
}

func artifactKey(uuid string, name string) []byte {
	return []byte(fmt.Sprintf("%s-%s", uuid, name))
}
