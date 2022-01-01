package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/health"
)

// Tests correct functionality when endpoint is reachable
func TestTracker_Ok(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(health.Handler))
	defer ts.Close()

	endpoints := []endpoint{
		endpoint{Name: "test", Endpoint: ts.URL},
	}
	ts.URL = fmt.Sprintf("%s%s", ts.URL, health.Endpoint)

	tracker := newPollTracker(endpoints)
	if len(tracker.pollMap) != len(endpoints) {
		t.Fatalf("expected %d endpoints, got %d", len(endpoints), len(tracker.pollMap))
	}

	now := time.Now()
	tracker.poll()
	status, exists := tracker.pollMap["test"]
	if !exists {
		t.Fatalf("expected %s to be in map", "test")
	}
	checkOk(status, now, t)
}

// Tests correct functionality when endpoint is unreachable
func TestTracker_Unreachable(t *testing.T) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(health.Handler))
	endpoints := []endpoint{
		endpoint{Name: "test", Endpoint: ts.URL},
	}
	ts.URL = fmt.Sprintf("%s%s", ts.URL, health.Endpoint)

	tracker := newPollTracker(endpoints)
	if len(tracker.pollMap) != len(endpoints) {
		t.Fatalf("expected %d endpoints, got %d", len(endpoints), len(tracker.pollMap))
	}

	now := time.Now()
	tracker.poll()
	status, exists := tracker.pollMap["test"]
	if !exists {
		t.Fatalf("expected %s to be in map", "test")
	}
	checkUnreachable(status, now, t)
}

// Tests transition from reachable to unreachable
func TestTracker_Ok_To_Unreachable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(health.Handler))
	endpoints := []endpoint{
		endpoint{Name: "test", Endpoint: ts.URL},
	}
	ts.URL = fmt.Sprintf("%s%s", ts.URL, health.Endpoint)

	tracker := newPollTracker(endpoints)
	if len(tracker.pollMap) != len(endpoints) {
		t.Fatalf("expected %d endpoints, got %d", len(endpoints), len(tracker.pollMap))
	}

	now := time.Now()
	tracker.poll()
	status, exists := tracker.pollMap["test"]
	if !exists {
		t.Fatalf("expected %s to be in map", "test")
	}

	checkOk(status, now, t)

	ts.Close()
	now = time.Now()
	tracker.poll()

	checkUnreachable(status, now, t)
}

// Tests transition from unreachable to reachable
func TestTracker_Unreachable_To_Ok(t *testing.T) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(health.Handler))
	endpoints := []endpoint{
		endpoint{Name: "test", Endpoint: ts.URL},
	}

	tracker := newPollTracker(endpoints)
	if len(tracker.pollMap) != len(endpoints) {
		t.Fatalf("expected %d endpoints, got %d", len(endpoints), len(tracker.pollMap))
	}

	now := time.Now()
	tracker.poll()
	status, exists := tracker.pollMap["test"]
	if !exists {
		t.Fatalf("expected %s to be in map", "test")
	}

	checkUnreachable(status, now, t)

	ts.Start()
	defer ts.Close()
	now = time.Now()

	// HACK: Have to update the url since ts.URL isn't set until you start the server,
	// still tests desired state transition.
	status.endpoint = ts.URL
	ts.URL = fmt.Sprintf("%s%s", ts.URL, health.Endpoint)
	tracker.poll()

	checkOk(status, now, t)
}

// Simple check to make sure the http endpoint doesn't fail
func TestTracker_HTTP(t *testing.T) {
	endpoints := []endpoint{
		endpoint{Name: "test", Endpoint: "http://www.kite.com"},
	}

	tracker := newPollTracker(endpoints)
	if len(tracker.pollMap) != len(endpoints) {
		t.Fatalf("expected %d endpoints, got %d", len(endpoints), len(tracker.pollMap))
	}

	ts := httptest.NewServer(tracker)
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal("error with get:", err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// --

type emptyHandler struct{}

func (e *emptyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

func checkOk(status *pollStatus, now time.Time, t *testing.T) {
	if status.status != health.StatusOK {
		t.Fatalf("expected status %s, but got %s", health.StatusOK, status.status)
	}
	if status.lastPolled.Before(now) {
		t.Fatalf("expected poll time to be set, got %s", status.lastPolled)
	}
	if status.lastError != nil {
		t.Fatal("expected no error, got:", status.lastError)
	}
	if status.alerted {
		t.Fatalf("expected no alert, but alert was set")
	}
}

func checkUnreachable(status *pollStatus, now time.Time, t *testing.T) {
	if status.status != health.StatusUnreachable {
		t.Fatalf("expected status %s, but got %s", health.StatusUnreachable, status.status)
	}
	if status.lastPolled.Before(now) {
		t.Fatalf("expected poll time to be set, got %s", status.lastPolled)
	}
	if status.lastError == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !status.alerted {
		t.Fatalf("expected alert, but no alert was set")
	}
}
