package notifications

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/conversion"
	"github.com/kiteco/kiteco/kite-golib/remotectrl"
	"github.com/stretchr/testify/assert"
)

func TestSetNotification(t *testing.T) {
	td, err := ioutil.TempDir("", "test_set_notification")
	if err != nil {
		t.Fatalf("Could not initialize a temporary directory: %s", err)
	}
	defer os.RemoveAll(td)
	m := initTestManager(td)
	r := mux.NewRouter()
	m.RegisterHandlers(r)

	var tests = []*struct {
		msg remotectrl.Message
		pl  payload
	}{
		{
			msg: remotectrl.Message{
				ID:   "0",
				Type: "set_notification",
			},
			pl: payload{
				HTML: "<body>test set completions cta post trial</body>",
				ID:   conversion.CompletionsCTAPostTrial,
			},
		},
		{
			msg: remotectrl.Message{
				ID:   "0",
				Type: "set_notification",
			},
			pl: payload{
				HTML: "<body>test set pro launch</body>",
				ID:   conversion.ProLaunch,
			},
		},
	}

	for _, test := range tests {
		j, err := json.Marshal(test.pl)
		if err != nil {
			t.Fatal("Could not marshal payload for test: ", test)
		}
		test.msg.Payload = j
	}

	for _, test := range tests {
		m.HandleRemoteMessage(test.msg)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", fmt.Sprintf("/clientapi/notifications/%s", test.pl.ID), nil)
		r.ServeHTTP(rec, req)
		assert.EqualValues(t, test.pl.HTML, rec.Body.String(), "End point failed to return the set notification")
	}
}

func TestDesktopNotification(t *testing.T) {
	td, err := ioutil.TempDir("", "test_desktop_notifications")
	if err != nil {
		t.Fatalf("Could not initialize a temporary directory: %s", err)
	}
	defer os.RemoveAll(td)
	m := initTestManager(td)
	r := mux.NewRouter()
	m.RegisterHandlers(r)

	test := struct {
		msg remotectrl.Message
		pl  payload
		nt  *component.MockNotify
	}{
		msg: remotectrl.Message{
			ID:   "0",
			Type: "desktop_notification",
		},
		pl: payload{
			HTML: "<body>test desktop_notif</body>",
		},
		nt: &component.MockNotify{},
	}

	j, err := json.Marshal(test.pl)
	if err != nil {
		t.Fatal("Could not marshal payload for test: ", test)
	}
	test.msg.Payload = j

	m.notifyFn = test.nt.ShowNotificationByID
	m.HandleRemoteMessage(test.msg)
	assert.True(t, test.nt.ShowNotifCalled(), "m.notify was not called")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/clientapi/notifications/%s", test.nt.ShowNotifCalledWith()), nil)
	r.ServeHTTP(rec, req)
	assert.EqualValues(t, test.pl.HTML, rec.Body.String(), "End point failed to serve sent desktop notification")
}

func TestGetNotifByIDFallbacks(t *testing.T) {
	td, err := ioutil.TempDir("", "test_get_notif_by_id")
	if err != nil {
		t.Fatalf("Could not initialize a temporary directory: %s", err)
	}
	defer os.RemoveAll(td)
	m := initTestManager(td)
	r := mux.NewRouter()
	m.RegisterHandlers(r)

	var tests = []*struct {
		id     string
		expect []byte
	}{
		{
			id:     conversion.CompletionsCTAPostTrial,
			expect: MustAsset(fmt.Sprintf("static/%s.html", conversion.CompletionsCTAPostTrial)),
		},
		{
			id:     conversion.ProLaunch,
			expect: MustAsset(fmt.Sprintf("static/%s.html", conversion.ProLaunch)),
		},
		{
			id:     "Gibberish_NotAnAsset",
			expect: MustAsset("static/not_found.html"),
		},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", fmt.Sprintf("/clientapi/notifications/%s", test.id), nil)
		r.ServeHTTP(rec, req)
		assert.EqualValues(t, test.expect, rec.Body.String(), fmt.Sprintf("Served unexpected notification for id %s", test.id))
	}
}

func TestShowNotifByID(t *testing.T) {
	td, err := ioutil.TempDir("", "test_get_notif_by_id")
	if err != nil {
		t.Fatalf("Could not initialize a temporary directory: %s", err)
	}
	defer os.RemoveAll(td)
	m := initTestManager(td)

	var tests = []*struct {
		id string
		nt component.MockNotify
	}{
		{
			id: conversion.CompletionsCTAPostTrial,
			nt: component.MockNotify{},
		},
		{
			id: conversion.ProLaunch,
			nt: component.MockNotify{},
		},
	}

	for _, test := range tests {
		m.notifyFn = test.nt.ShowNotificationByID
		err := m.ShowNotificationByID(test.id)
		assert.Nil(t, err)
		assert.True(t, test.nt.ShowNotifCalled(), "m.notify was not called")
	}

	err = m.ShowNotificationByID("Gibberish_NotAnID")
	assert.NotNil(t, err)
}

func initTestManager(tempdir string) *Manager {
	m := NewManager(true)
	opts := component.InitializerOptions{
		Platform: &platform.Platform{
			IsUnitTestMode: true,
			KiteRoot:       tempdir,
		},
		UserIDs:    userids.NewUserIDs("", ""),
		AuthClient: auth.NewTestClient(10 * time.Second),
	}
	m.Initialize(opts)
	return m
}
