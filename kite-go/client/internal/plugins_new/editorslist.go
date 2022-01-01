package plugins

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
)

// refreshRunning adds currently running editors to the list, purges paths which do not exist anymore and stores the data on disk
// when necessary
func (m *Manager) refreshRunning(ctx context.Context) {
	count, additional := m.detectRunning(ctx)
	purged := m.editors.purgeDetected()
	log.Printf("Detected %d running editors with %d new entries, purged %d entries", count, additional, purged)
	if count > 0 || purged > 0 {
		_ = m.editors.save()
	}
}

// detectRunning updates the editors.json file
// it returns the number of currently running and additionally detected editors
func (m *Manager) detectRunning(ctx context.Context) (int, int) {
	var detected, additional int
	for _, plugin := range m.pluginManagers {
		paths, err := plugin.DetectRunningEditors(ctx)
		if err != nil {
			continue
		}

		for _, path := range paths {
			if m.editors.addDetected(plugin.ID(), path) {
				additional++
			}
			detected++
		}
	}
	return detected, additional
}

type editorsList struct {
	mu       sync.Mutex
	filename string
	Detected map[string][]string `json:"detected"`
	Manual   map[string][]string `json:"manual"`
}

func newEditorsList(filename string) *editorsList {
	return &editorsList{
		filename: filename,
		Detected: make(map[string][]string),
		Manual:   make(map[string][]string),
	}
}

func (d *editorsList) detected(editorID string) []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.Detected[editorID]
}

// addDetected adds the entry, but only if it's not yet stored
// it returns true if it was added, false otherwise
func (d *editorsList) addDetected(editorID, path string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	current := d.Detected[editorID]
	if !shared.StringsContain(current, path) {
		d.Detected[editorID] = append(current, path)
		return true
	}
	return false
}

// cleanupDetected removes all detected locations which do not exist on disk anymore
func (d *editorsList) purgeDetected() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.purgeMapLocked(d.Detected)
}

// cleanupDetected removes all detected locations which do not exist on disk anymore
func (d *editorsList) purgeManual() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.purgeMapLocked(d.Manual)
}

func (d *editorsList) purgeMapLocked(data map[string][]string) int {
	var removed int
	for id, values := range data {
		var valid []string
		for _, path := range values {
			if _, err := os.Stat(path); err == nil {
				valid = append(valid, path)
			} else {
				removed++
			}
		}
		data[id] = valid
	}
	return removed
}

func (d *editorsList) removeDetected(editorID, path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var newList []string
	current := d.Detected[editorID]
	for _, v := range current {
		if v != path {
			newList = append(newList, v)
		}
	}
	d.Detected[editorID] = newList
}

func (d *editorsList) manual(editorID string) []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.Manual[editorID]
}

func (d *editorsList) addManual(editorID, path string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	current := d.Manual[editorID]
	if !shared.StringsContain(current, path) {
		d.Manual[editorID] = append(current, path)
		return true
	}
	return false
}

func (d *editorsList) removeManual(editorID, path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var newList []string
	current := d.Manual[editorID]
	for _, v := range current {
		if v != path {
			newList = append(newList, v)
		}
	}
	d.Manual[editorID] = newList
}

// loadDetectedEditors loads the last set of detected editors from disk
func (d *editorsList) load() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := ioutil.ReadFile(d.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, d)
}

func (d *editorsList) save() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(d.filename, data, 0600)
}
