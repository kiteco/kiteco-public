package release

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeLatest(t *testing.T) {
	linuxNonCanaryVersion := "0.20210310.0"
	linuxCanaryVersion := "0.20210311.0"
	macNonCanaryVersion := "0.20210311.1"
	macCanaryVersion := "0.20210312.1"
	windowsVersion := "0.2021.309.1"

	dbEntries := []MetadataCreateArgs{
		{
			// Latest Public Non-Canary Linux
			Client:            Linux,
			Version:           linuxNonCanaryVersion,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		},
		{
			// Latest Public Canary Linux
			Client:            Linux,
			Version:           linuxCanaryVersion,
			GitHash:           gitHash,
			ReleasePercentage: 20,
			Public:            true,
		},
		{
			// Latest Public Canary and Non-Canary Windows
			Client:            Windows,
			Version:           windowsVersion,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		},
		{
			// Latest Public Non-Canary Mac
			Client:            Mac,
			Version:           macNonCanaryVersion,
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		},
		{
			Client:            Mac,
			Version:           "0.20210311.2",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		},
		{
			// Bad Release Mac
			Client:            Mac,
			Version:           "0.20210311.2",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 0,
			Public:            true,
		},
		{
			// Latest Public Canary Mac
			Client:            Mac,
			Version:           macCanaryVersion,
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 20,
			Public:            true,
		},
	}

	cases := map[string]struct {
		getsCanary           func(sum64 uint64, releasePercentage uint8) bool
		route                string
		expectStatus         int
		expectLocContains    *string
		expectLocNotContains *string
		failmsg              string
	}{
		"MacDownloadsLatest": {
			getsCanary:        getsCanaryFromHashNever,
			route:             "/dls/mac/current",
			expectStatus:      http.StatusSeeOther,
			expectLocContains: proto.String(fmt.Sprintf("/mac/%s/Kite.dmg", macNonCanaryVersion)),
			failmsg:           "Must get latest release from /dls/mac/current",
		},
		"MacDownloadsNoCanary": {
			getsCanary:           getsCanaryFromHashAlways,
			route:                "/dls/mac/current",
			expectStatus:         http.StatusSeeOther,
			expectLocNotContains: proto.String(fmt.Sprintf("/mac/%s/Kite.dmg", macCanaryVersion)),
			failmsg:              "Must never get non-100% canary release from /dls/mac/current",
		},
		"LinuxDownloadsLatest": {
			getsCanary:        getsCanaryFromHashNever,
			route:             "/dls/linux/current",
			expectStatus:      http.StatusSeeOther,
			expectLocContains: proto.String(fmt.Sprintf("/linux/%s/kite-installer.sh", linuxNonCanaryVersion)),
			failmsg:           "Must get latest release from /dls/linux/current",
		},
		"LinuxDownloadsNoCanary": {
			getsCanary:           getsCanaryFromHashAlways,
			route:                "/dls/linux/current",
			expectStatus:         http.StatusSeeOther,
			expectLocNotContains: proto.String(fmt.Sprintf("/linux/%s/kite-installer.sh", linuxCanaryVersion)),
			failmsg:              "Must never get non-100% canary release from /dls/linux/current",
		},
		"WindowsDownloadsLatest": {
			getsCanary:        getsCanaryFromHashNever,
			route:             "/dls/windows/current",
			expectStatus:      http.StatusSeeOther,
			expectLocContains: proto.String(fmt.Sprintf("/windows/%s/KiteSetup.exe", windowsVersion)),
			failmsg:           "Must get latest release from /dls/windows/current",
		},
		"NonExistentPlatform": {
			getsCanary:   getsCanaryFromHashNever,
			route:        "/dls/android/current",
			expectStatus: http.StatusNotFound,
			failmsg:      "For unknown platform, must 404",
		},
	}

	r, server := makeTestRouterServer(t, true, dbEntries)

	for tname, tc := range cases {
		server.getsCanary = tc.getsCanary

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", tc.route, nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, tc.expectStatus, rec.Code, tname, tc.failmsg)
		if rec.Code > 400 {
			continue
		}
		if tc.expectLocContains != nil {
			assert.Contains(t, rec.Result().Header.Get("Location"), *tc.expectLocContains, tname, tc.failmsg)
		}
		if tc.expectLocNotContains != nil {
			assert.NotContains(t, rec.Result().Header.Get("Location"), *tc.expectLocNotContains, tname, tc.failmsg)
		}
	}
}

func makeTestRouterServer(t *testing.T, readsPublic bool, entries []MetadataCreateArgs) (*mux.Router, *Server) {
	rout := mux.NewRouter()
	mgr := makeTestManager(readsPublic)
	for _, entry := range entries {
		_, err := mgr.Create(entry)
		require.NoError(t, err)
	}
	return rout, NewServer(rout, mgr, DefaultPlatforms)
}

func getsCanaryFromHashAlways(sum64 uint64, releasePercentage uint8) bool {
	return true
}

func getsCanaryFromHashNever(sum64 uint64, releasePercentage uint8) bool {
	return false
}
