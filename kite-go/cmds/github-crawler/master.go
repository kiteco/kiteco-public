package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocarina/gocsv"
)

type repoEntry struct {
	ID         int64  `csv:"id"`
	Owner      string `csv:"owner"`
	Repo       string `csv:"repo"`
	ForkedFrom int64  `csv:"forked_from"`
	Forks      int    `csv:"-"`
}

type handlers struct {
	rw          sync.RWMutex
	entries     []*repoEntry
	entryByID   map[string]*repoEntry
	fetched     map[string]bool
	pending     map[string]time.Time
	fetchedPath string

	// stats
	unique        int
	forked        int
	uniqueCrawled int
	forkedCrawled int
	fetchedCount  int
}

func newHandlers(entries []*repoEntry, fetched string) *handlers {
	h := &handlers{
		entries:     entries,
		entryByID:   make(map[string]*repoEntry),
		fetched:     make(map[string]bool),
		pending:     make(map[string]time.Time),
		fetchedPath: fetched,
	}

	h.loadFetched(fetched)
	go h.writeFetched()

	for _, entry := range entries {
		h.entryByID[fmt.Sprintf("%d", entry.ID)] = entry
		if entry.ForkedFrom < 0 {
			h.unique++
		} else {
			h.forked++
		}
	}

	return h
}

func (h *handlers) handleProgress(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Github crawl status:\n\n")
	fmt.Fprintf(w, "Total repos: %d\n", len(h.entries))
	fmt.Fprintf(w, "    Unique: %d (%.02f)\n", h.unique, float64(h.unique)/float64(len(h.entries)))
	fmt.Fprintf(w, "    Forks: %d (%.02f)\n\n", h.forked, float64(h.forked)/float64(len(h.entries)))
	fmt.Fprintf(w, "Crawl progress: %d/%d (%.02f)\n", h.fetchedCount, len(h.entries), float64(h.fetchedCount)/float64(len(h.entries)))
	fmt.Fprintf(w, "    Unique: %d/%d (%.02f)\n", h.uniqueCrawled, h.unique, float64(h.uniqueCrawled)/float64(h.unique))
	fmt.Fprintf(w, "    Forks: %d/%d (%.02f)\n", h.forkedCrawled, h.forked, float64(h.forkedCrawled)/float64(h.forked))
}

func (h *handlers) handleNextRepos(w http.ResponseWriter, r *http.Request) {
	n := r.URL.Query().Get("n")
	if n == "" {
		http.Error(w, "query param 'n' required", http.StatusBadRequest)
		return
	}

	count, err := strconv.ParseInt(n, 10, 64)
	if err != nil {
		http.Error(w, "unable to parse param 'n' as an integer", http.StatusBadRequest)
		return
	}

	h.rw.RLock()
	defer h.rw.RUnlock()
	var nextRepos []*repoEntry
	for _, entry := range h.entries {
		if len(nextRepos) >= int(count) {
			break
		}

		if entry.ForkedFrom > 0 {
			continue
		}

		id := fmt.Sprintf("%d", entry.ID)

		if _, exists := h.fetched[id]; exists {
			continue
		}

		if ts, exists := h.pending[id]; exists {
			if time.Since(ts) < 5*time.Minute {
				continue
			}
			delete(h.pending, id)
		}

		nextRepos = append(nextRepos, entry)
		h.pending[id] = time.Now()
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(&nextRepos)
	if err != nil {
		http.Error(w, fmt.Sprintf("json marshal error: %s", err), http.StatusInternalServerError)
		return
	}
	io.Copy(w, &buf)
}

func (h *handlers) handleFetched(w http.ResponseWriter, r *http.Request) {
	ids := r.URL.Query().Get("ids")
	if ids == "" {
		http.Error(w, "query param 'ids' required", http.StatusBadRequest)
		return
	}

	parts := strings.Split(ids, ",")

	h.rw.Lock()
	defer h.rw.Unlock()
	for _, part := range parts {
		part = strings.TrimSpace(part)
		h.fetched[part] = true
	}
}

func (h *handlers) handleErrored(w http.ResponseWriter, r *http.Request) {
	ids := r.URL.Query().Get("ids")
	if ids == "" {
		http.Error(w, "query param 'ids' required", http.StatusBadRequest)
		return
	}

	parts := strings.Split(ids, ",")

	h.rw.Lock()
	defer h.rw.Unlock()
	for _, part := range parts {
		part = strings.TrimSpace(part)
		h.fetched[part] = false
	}
}

func (h *handlers) writeFetched() {
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		fetched := make(map[string]bool, len(h.fetched))
		func() {
			h.rw.RLock()
			defer h.rw.RUnlock()
			for k, v := range h.fetched {
				fetched[k] = v
			}
		}()

		if len(fetched) == 0 {
			continue
		}

		h.uniqueCrawled = 0
		h.forkedCrawled = 0
		h.fetchedCount = len(fetched)
		for _, entry := range h.entries {
			_, fetched := fetched[fmt.Sprintf("%d", entry.ID)]
			if !fetched {
				continue
			}
			if entry.ForkedFrom < 0 {
				h.uniqueCrawled++
			} else {
				h.forkedCrawled++
			}
		}

		buf, err := json.MarshalIndent(&fetched, "", "  ")
		if err != nil {
			log.Println(err)
			continue
		}

		func() {
			dir, err := ioutil.TempDir("", "fetched")
			if err != nil {
				log.Println(err)
				return
			}
			defer os.RemoveAll(dir)

			fn := filepath.Join(dir, "fetched.json.tmp")
			ioutil.WriteFile(fn, buf, os.ModePerm)

			err = os.Rename(fn, h.fetchedPath)
			if err != nil {
				log.Println(err)
			}
		}()
	}
}

func (h *handlers) loadFetched(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	fetched := make(map[string]bool)
	err = json.NewDecoder(f).Decode(&fetched)
	if err != nil {
		log.Println(err)
		return
	}

	h.rw.Lock()
	defer h.rw.Unlock()
	h.fetched = fetched
}

// --

func master(input, fetched, port string) {
	f, err := os.Open(input)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	log.Println("reading repos from csv...")

	entries := []*repoEntry{}
	err = gocsv.UnmarshalFile(f, &entries)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("found %d repos", len(entries))

	entriesByID := make(map[int64]*repoEntry)
	for _, entry := range entries {
		entriesByID[entry.ID] = entry
	}

	var nonForked int
	for _, entry := range entriesByID {
		if entry.ForkedFrom < 0 {
			nonForked++
			continue
		}
		if e := entriesByID[entry.ForkedFrom]; e != nil {
			e.Forks++
		}
	}

	log.Printf("%d are not forks (skipping %d forks)", nonForked, len(entries)-nonForked)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Forks > entries[j].Forks
	})

	h := newHandlers(entries, fetched)
	http.HandleFunc("/next-repos", h.handleNextRepos)
	http.HandleFunc("/fetched", h.handleFetched)
	http.HandleFunc("/errored", h.handleErrored)
	http.HandleFunc("/progress", h.handleProgress)
	log.Println("listening on", port)
	log.Fatalln(http.ListenAndServe(port, nil))
}
