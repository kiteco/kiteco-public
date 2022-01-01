package manifest_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncodeDecode tests that Manifest methods Encode & Decode are inverses
func TestEncodeDecode(t *testing.T) {
	// verify encode/decode are inverses
	test := func(m0 manifest.Manifest) bool {
		var b1 bytes.Buffer
		var m1 manifest.Manifest
		if err := m0.Encode(&b1); err != nil {
			t.Error(err)
			return false
		}
		if err := m1.Decode(&b1); err != nil {
			t.Error(err)
			return false
		}

		if !reflect.DeepEqual(m0, m1) {
			return false
		}

		var b2 bytes.Buffer
		var m2 manifest.Manifest
		if err := m1.Encode(&b2); err != nil {
			t.Error(err)
			return false
		}
		if err := m2.Decode(&b2); err != nil {
			t.Error(err)
			return false
		}

		if !reflect.DeepEqual(m1, m2) {
			return false
		}

		return true
	}

	if err := quick.Check(test, &quick.Config{MaxCount: 2}); err != nil {
		t.Error(err)
	}
}

// TestLoadSucceeds tests that we can successfully load resource groups from a manifest;
// see TestKiteManifest for actually testing the underlying data is reasonable
func TestLoadSucceeds(t *testing.T) {
	test := func(m manifest.Manifest) bool {
		for _, dist := range m.Distributions() {
			_, err := m.Load(dist)
			if err != nil {
				return false
			}
		}
		return true
	}

	if err := quick.Check(test, &quick.Config{MaxCount: 2}); err != nil {
		t.Error(err)
	}
}

// TestKiteManifest tests actual manifest data from the Kite manifest (? or maybe a fixturized manifest)
func TestKiteManifestDistributions(t *testing.T) {
	var distNames []string
	for _, dist := range manifest.KiteManifest.Distributions() {
		distNames = append(distNames, strings.ToLower(dist.Name))
	}
	assert.Subset(t, distNames, []string{"requests", "numpy", "tensorflow", "flask", "werkzeug"})
}

// TestBindata verifies that the latest go-bindata generated asset matches the latest manifest.json
func TestBindata(t *testing.T) {
	bindataBytes := manifest.MustAsset("manifest.json")

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		t.Fatal("failed to lookup GOPATH")
	}
	manifestPath := path.Join(gopath, "src/github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/manifest.json")

	jsonBytes, err := ioutil.ReadFile(manifestPath)
	require.Nil(t, err)

	assert.True(t, bytes.Equal(bindataBytes, jsonBytes))
}
