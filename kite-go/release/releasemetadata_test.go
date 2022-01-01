package release

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-golib/errors"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	signature = "MCwCFAUMstsN7Hw=="
	gitHash   = "a01b23c45d67e89f"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		client    Platform
		version   string
		expected  string
		expectErr bool
	}{
		{
			client:   Windows,
			version:  "1.2021.1101.1",
			expected: "20211101.1",
		},
		{
			client:   Windows,
			version:  "1.2021.110.1",
			expected: "20210110.1",
		},
		{
			client:   Windows,
			version:  "1.2021.111.1",
			expected: "20210111.1",
		},
		{
			client:   Mac,
			version:  "0.20200307.0",
			expected: "20200307.0",
		},
		{
			client:   Linux,
			version:  "2.20201231.2",
			expected: "20201231.2",
		},
		{
			client:    Windows,
			version:   "1.20201231.2",
			expectErr: true,
		},
		{
			client:    Windows,
			version:   "1.2020.12.31.3",
			expectErr: true,
		},
	}

	for _, test := range tests {
		out, err := normalize(test.client, test.version)
		if test.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, test.expected, out)
	}
}

func TestDateInt(t *testing.T) {
	date := time.Date(2015, time.September, 28, 18, 22, 0, 0, time.UTC)
	assert.EqualValues(t, 20150928, dateInt(date))

	date = time.Date(1991, time.June, 21, 7, 57, 0, 0, time.UTC)
	assert.EqualValues(t, 19910621, dateInt(date))

	date = time.Date(2014, time.November, 17, 10, 30, 0, 0, time.UTC)
	assert.EqualValues(t, 20141117, dateInt(date))
}

func TestLatestWithNoRelease(t *testing.T) {
	for _, public := range []bool{true, false} {
		rmm := makeTestManager(public)
		m, err := rmm.LatestNonCanary(Mac)
		assert.NoError(t, err)
		assert.Nil(t, m)
	}
}

func TestNextVersionWithNoRelease(t *testing.T) {
	for _, public := range []bool{true, false} {
		rmm := makeTestManager(public)
		version, err := rmm.nextVersion(Mac)
		assert.NoError(t, err)
		res := strings.Split(version, ".")
		require.Equal(t, 3, len(res))
		assert.Equal(t, strconv.Itoa(majorVersion), res[0])
		assert.Equal(t, strconv.FormatInt(dateInt(time.Now()), 10), res[1])
		assert.Equal(t, "0", res[2])
	}
}

func TestOneRelease(t *testing.T) {
	for _, public := range []bool{true, false} {
		rmm := makeTestManager(public)
		version, err := rmm.nextVersion(Mac)
		assert.NoError(t, err)
		release, err := rmm.Create(MetadataCreateArgs{
			Client:            Mac,
			Version:           version,
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            public,
		})
		require.NoError(t, err)
		assert.EqualValues(t, 1, release.ID)
		assert.Equal(t, Mac, release.Client)
		assert.Equal(t, signature, release.DSASignature)
		assert.Equal(t, gitHash, release.GitHash)
		assert.WithinDuration(t, time.Now(), release.CreatedAt, time.Second)
		assert.Equal(t, fmt.Sprintf("0.%d.0", dateInt(time.Now())), release.Version)
	}
}

func TestLatestWithPublicPrivate(t *testing.T) {
	// These entries in this order is nonsensical but
	// meant to test some edgecase and fallback behavior
	entries := []MetadataCreateArgs{
		{
			// Latest Public Non-Canary Linux
			Client:            Linux,
			Version:           "2.20210310.0",
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		},
		{
			// Latest Private Canary and Non-Canary Windows
			Client:            Windows,
			Version:           "1.2021.309.0",
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            false,
		},
		{
			Client:            Mac,
			Version:           "0.20210311.0",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 20,
			Public:            false,
		},
		{
			// Latest Private Non-Canary Mac
			Client:            Mac,
			Version:           "0.20210311.0",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            false,
		},
		{
			// Latest Private Canary Mac
			Client:            Mac,
			Version:           "0.20210311.1",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 40,
			Public:            false,
		},
		{
			Client:            Mac,
			Version:           "0.20210311.0",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 20,
			Public:            true,
		},
		{
			Client:            Mac,
			Version:           "0.20210311.0",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		},
		{
			// Latest Public Canary and Non-Canary Windows
			Client:            Windows,
			Version:           "1.2021.309.1",
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		},
		{
			// Latest Public Non-Canary Mac
			Client:            Mac,
			Version:           "0.20210311.1",
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
			Version:           "0.20210312.0",
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 5,
			Public:            true,
		},
	}

	pubMgr := makeTestManager(true)
	privMgr := makeTestManager(false)
	for _, entry := range entries {
		_, err := privMgr.Create(entry)
		require.NoError(t, err)
	}

	cases := map[string]struct {
		client     Platform
		latestFn   func(Platform) (*Metadata, error)
		expPublic  bool
		expVersion string
		expRelPerc uint8
	}{
		"MacLatestPrivateNonCanary": {
			client:     Mac,
			latestFn:   privMgr.LatestNonCanary,
			expPublic:  false,
			expVersion: "0.20210311.0",
			expRelPerc: 100,
		},
		"MacLatestPrivateCanary": {
			client:     Mac,
			latestFn:   privMgr.LatestCanary,
			expPublic:  false,
			expVersion: "0.20210311.1",
			expRelPerc: 40,
		},
		"LinuxLatestPublicNonCanary": {
			client:     Linux,
			latestFn:   pubMgr.LatestNonCanary,
			expPublic:  true,
			expVersion: "2.20210310.0",
			expRelPerc: 100,
		},
		"LinuxLatestPublicCanary": {
			client:     Linux,
			latestFn:   pubMgr.LatestCanary,
			expPublic:  true,
			expVersion: "2.20210310.0",
			expRelPerc: 100,
		},
		"MacLatestPublicNonCanary": {
			client:     Mac,
			latestFn:   pubMgr.LatestNonCanary,
			expPublic:  true,
			expVersion: "0.20210311.1",
			expRelPerc: 100,
		},
		"MacLatestPublicCanary": {
			client:     Mac,
			latestFn:   pubMgr.LatestCanary,
			expPublic:  true,
			expVersion: "0.20210312.0",
			expRelPerc: 5,
		},
		"WindowsLatestPrivateNonCanary": {
			client:     Windows,
			latestFn:   privMgr.LatestNonCanary,
			expPublic:  false,
			expVersion: "1.2021.309.0",
			expRelPerc: 100,
		},
		"WindowsLatestPrivateCanary": {
			client:     Windows,
			latestFn:   privMgr.LatestCanary,
			expPublic:  false,
			expVersion: "1.2021.309.0",
			expRelPerc: 100,
		},
		"WindowsLatestPublicNonCanary": {
			client:     Windows,
			latestFn:   pubMgr.LatestNonCanary,
			expPublic:  true,
			expVersion: "1.2021.309.1",
			expRelPerc: 100,
		},
		"WindowsLatestPublicCanary": {
			client:     Windows,
			latestFn:   pubMgr.LatestCanary,
			expPublic:  true,
			expVersion: "1.2021.309.1",
			expRelPerc: 100,
		},
	}

	for tname, tc := range cases {
		release, err := tc.latestFn(tc.client)
		require.NoError(t, err, tname)
		assert.Equal(t, tc.expPublic, release.Public, tname)
		assert.Equal(t, tc.expVersion, release.Version, tname)
		assert.Equal(t, tc.expRelPerc, release.ReleasePercentage, tname)
	}
}

func TestMultipleReleases(t *testing.T) {
	rmm := makeTestManager(true)
	date := dateInt(time.Now())

	var numReleases int
	for numReleases = 0; numReleases < 5; numReleases++ {
		version, err := rmm.nextVersion(Mac)
		assert.NoError(t, err)
		release, err := rmm.Create(MetadataCreateArgs{
			Client:            Mac,
			Version:           version,
			DSASignature:      signature,
			GitHash:           gitHash,
			ReleasePercentage: 100,
			Public:            true,
		})
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("0.%d.%d", date, numReleases), release.Version)

		latest, err := rmm.LatestNonCanary(Mac)
		require.NoError(t, err)
		assert.EqualValues(t, numReleases+1, latest.ID)
		assert.Equal(t, Mac, latest.Client)
		assert.Equal(t, signature, release.DSASignature)
		assert.Equal(t, gitHash, release.GitHash)
		assert.WithinDuration(t, time.Now(), release.CreatedAt, time.Second)
		assert.Equal(t, fmt.Sprintf("0.%d.%d", dateInt(time.Now()), numReleases), latest.Version)
	}
}

func TestCanaryReleases(t *testing.T) {
	rmm := makeTestManager(true)

	versionStable := "0.2019.1009.0"
	versionCanary := "0.2019.1009.1"

	_, err := rmm.Create(MetadataCreateArgs{
		Client:            Windows,
		Version:           versionStable,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
		Public:            true,
	})
	require.NoError(t, err)
	latestStable, err := rmm.LatestNonCanary(Windows)
	require.NoError(t, err)
	assert.Equal(t, Windows, latestStable.Client)
	assert.Equal(t, versionStable, latestStable.Version)

	// canary release for 0% of users
	_, err = rmm.Create(MetadataCreateArgs{
		Client:            Windows,
		Version:           versionCanary,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 0,
		Public:            true,
	})
	require.NoError(t, err)
	latestCanary, err := rmm.LatestNonCanary(Windows)
	require.NoError(t, err)
	assert.Equal(t, Windows, latestCanary.Client)
	assert.Equal(t, versionStable, latestCanary.Version, "canary release for 0% must not be returned")

	// promote the canary release to all users
	err = rmm.Publish(Windows, versionCanary, 100)
	require.NoError(t, err)
	latestCanary, err = rmm.LatestNonCanary(Windows)
	require.NoError(t, err)
	assert.Equal(t, versionCanary, latestCanary.Version, "canary release for 100% must be returned")

	// demote it again
	// now the older, stable release must be returned
	err = rmm.Publish(Windows, versionCanary, 0)
	require.NoError(t, err)
	latestCanary, err = rmm.LatestNonCanary(Windows)
	require.NoError(t, err)
	assert.Equal(t, versionStable, latestCanary.Version, "stable release must be returned")
}

func TestFromVersion(t *testing.T) {
	rmm := makeTestManager(false)

	version, err := rmm.nextVersion(Mac)
	require.NoError(t, err)
	release, err := rmm.Create(MetadataCreateArgs{
		Client:            Mac,
		Version:           version,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
	})
	require.NoError(t, err)

	version2, err := rmm.nextVersion(Mac)
	require.NoError(t, err)
	_, err = rmm.Create(MetadataCreateArgs{
		Client:            Mac,
		Version:           version2,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
	})
	require.NoError(t, err)

	releaseFromVersion, err := rmm.fromVersion(Mac, version)
	require.NoError(t, err)

	assert.Equal(t, release.String(), releaseFromVersion.String())

	version, err = rmm.nextVersion(Mac)
	require.NoError(t, err)
	_, err = rmm.fromVersion(Mac, version)
	require.Error(t, err)
	assert.Equal(t, ErrVersionNotFound, err)
}

func TestDeltas(t *testing.T) {
	rmm := makeTestManager(true)

	version, err := rmm.nextVersion(Mac)
	require.NoError(t, err)
	_, err = rmm.Create(MetadataCreateArgs{
		Client:            Mac,
		Version:           version,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
	})
	require.NoError(t, err)

	version2, err := rmm.nextVersion(Mac)
	require.NoError(t, err)
	_, err = rmm.Create(MetadataCreateArgs{
		Client:            Mac,
		Version:           version2,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
	})
	require.NoError(t, err)

	version3, err := rmm.nextVersion(Mac)
	require.NoError(t, err)
	_, err = rmm.Create(MetadataCreateArgs{
		Client:            Mac,
		Version:           version3,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
	})
	require.NoError(t, err)

	// No deltas have been created yet
	deltas, err := rmm.DeltasToVersion(Mac, version)
	require.NoError(t, err)
	require.Empty(t, deltas)

	delta, err := rmm.CreateDelta(Mac, version, version3, signature)
	require.NoError(t, err)

	// Error is returned when creating delta for client, version, and from_version that exist
	_, err = rmm.CreateDelta(Mac, version, version3, signature)
	require.Error(t, err)

	delta2, err := rmm.CreateDelta(Mac, version2, version3, signature)
	require.NoError(t, err)

	// deltas should be ordered by from_version
	deltas, err = rmm.DeltasToVersion(Mac, version3)
	require.NoError(t, err)
	require.Len(t, deltas, 2)
	require.Equal(t, delta2.String(), deltas[0].String())
	require.Equal(t, delta.String(), deltas[1].String())
}

func TestWindowsRequests(t *testing.T) {
	rmm := makeTestManager(true)

	version, err := rmm.nextVersion(Windows)
	require.NoError(t, err)
	_, err = rmm.Create(MetadataCreateArgs{
		Client:            Windows,
		Version:           version,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
		Public:            true,
	})
	require.NoError(t, err)

	nextVersion, err := rmm.nextVersion(Windows)
	require.NoError(t, err)
	_, err = rmm.Create(MetadataCreateArgs{
		Client:            Windows,
		Version:           nextVersion,
		DSASignature:      signature,
		GitHash:           gitHash,
		ReleasePercentage: 100,
		Public:            true,
	})
	require.NoError(t, err)

	// only serve for Windows, NewServer terminates if there's no release for a platform
	router := mux.NewRouter()
	s := NewServer(router, rmm, []Platform{Windows})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	httpServer := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: router,
	}
	go httpServer.Serve(listener)
	defer httpServer.Shutdown(context.Background())

	httpClient := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.Errorf("no redirect!")
		},
	}

	// finally test the request handling
	// no update info for missing will-attempt-update-if-provided=true
	resp, _ := httpClient.Get(fmt.Sprintf("http://%s/windows/kite-app/update-check?version=%s", listener.Addr().String(), version))
	require.NotNil(t, resp)
	require.EqualValues(t, http.StatusOK, resp.StatusCode)

	// no update info for current version
	resp, _ = httpClient.Get(fmt.Sprintf("http://%s/windows/kite-app/update-check?will-attempt-update-if-provided=true&version=%s", listener.Addr().String(), nextVersion))
	require.NotNil(t, resp)
	require.EqualValues(t, http.StatusOK, resp.StatusCode)

	// redirect for available update info
	resp, _ = httpClient.Get(fmt.Sprintf("http://%s/windows/kite-app/update-check?will-attempt-update-if-provided=true&version=%s", listener.Addr().String(), version))
	require.NotNil(t, resp)
	require.EqualValues(t, http.StatusSeeOther, resp.StatusCode)
	require.EqualValues(t, fmt.Sprintf("%s/windows/%s/KiteUpdateInfo.xml", BunnyVolumeDownloadPrefix, nextVersion), resp.Header["Location"][0])

	// test serving of KiteUpdater.exe data
	url := fmt.Sprintf("http://%s/windows/%s/KiteUpdater.exe", listener.Addr().String(), nextVersion)
	resp, _ = httpClient.Get(url)
	require.NotNil(t, resp)
	require.EqualValues(t, http.StatusSeeOther, resp.StatusCode)

	// create patch updater entry and test URLs
	_, err = rmm.CreateDelta(Windows, version, nextVersion, "foo==")
	require.NoError(t, err)

	s.updateCache()

	resp, _ = httpClient.Get(fmt.Sprintf("http://%s/windows/kite-app/update-check?will-attempt-update-if-provided=true&version=%s", listener.Addr().String(), version))
	require.NotNil(t, resp)
	require.EqualValues(t, http.StatusSeeOther, resp.StatusCode)
	require.EqualValues(t, fmt.Sprintf("%s/windows/%s/deltaFrom/%s/KiteDeltaUpdateInfo.xml", BunnyVolumeDownloadPrefix, nextVersion, version), resp.Header["Location"][0])

	url = fmt.Sprintf("http://%s/windows/%s/KitePatchUpdater%s-%s.exe", listener.Addr().String(), nextVersion, version, nextVersion)
	resp, _ = httpClient.Get(url)
	require.NotNil(t, resp)
	require.EqualValues(t, http.StatusSeeOther, resp.StatusCode)
}

func makeTestManager(readsPublic bool) *MetadataManagerImpl {
	db := DB("postgres", "postgres://communityuser:kite@localhost/account_test?sslmode=disable")
	db.Migrator().DropTable(&Metadata{})
	db.Migrator().DropTable(&Delta{})

	rmm := &MetadataManagerImpl{
		db:          db,
		readsPublic: readsPublic,
	}
	rmm.Migrate()
	return rmm
}

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
