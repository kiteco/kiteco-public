package kiteserver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseKiteServer(t *testing.T) {
	kiteURL, err := ParseKiteServerURL("https://tfserving.kite.local")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "https://tfserving.kite.local:443"), kiteURL)

	kiteURL, err = ParseKiteServerURL("http://tfserving.kite.local")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "http://tfserving.kite.local:80"), kiteURL)

	kiteURL, err = ParseKiteServerURL("http://tfserving.kite.local:1234")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "http://tfserving.kite.local:1234"), kiteURL)

	kiteURL, err = ParseKiteServerURL("https://tfserving-1.kite.local")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "https://tfserving-1.kite.local:443"), kiteURL)

	kiteURL, err = ParseKiteServerURL("tfserving.kite.local:1234")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "http://tfserving.kite.local:1234"), kiteURL)

	kiteURL, err = ParseKiteServerURL("tfserving.kite.local")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "https://tfserving.kite.local:443"), kiteURL)

	kiteURL, err = ParseKiteServerURL("tfserving.kite.local:1234")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "http://tfserving.kite.local:1234"), kiteURL)

	kiteURL, err = ParseKiteServerURL("https://clientname@kite.local")
	require.NoError(t, err)
	require.EqualValues(t, mustParse(t, "https://clientname@kite.local:443"), kiteURL)
}

func Test_Health(t *testing.T) {
	if os.ExpandEnv("CI") != "" {
		t.Skipf("test with network access disabled on CI")
	}

	_, _, err := GetHealth("http://tfserving.kite.com:8085")
	require.NoError(t, err)

	_, _, err = GetHealth("https://cloud.kite.com")
	require.NoError(t, err)
}

func mustParse(t *testing.T, value string) *url.URL {
	u, err := url.Parse(value)
	require.NoError(t, err)
	return u
}
