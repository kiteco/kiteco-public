package capture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HandleCapture(t *testing.T) {
	server, _, db := makeTestServer()
	defer db.Close()

	installid := "0"
	machineid := "1"
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	filename := "testfile"

	vals := url.Values{}
	vals.Set("filename", filename)
	vals.Set("machineid", machineid)
	vals.Set("installid", installid)
	vals.Set("timestamp", ts)
	captureURL := makeTestURL(server.URL, "capture")
	ep, err := captureURL.Parse("?" + vals.Encode())
	var buf bytes.Buffer
	resp, err := http.Post(ep.String(), "text/plain", &buf)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	data := struct {
		URL string `json:"url"`
	}{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&data)
	require.NoError(t, err)
	key := fmt.Sprintf("%s/%s/%s/%s/%s.gz", "dev", installid, machineid, ts, filename)
	urlStr := fmt.Sprintf("%s/%s/%s", s3URL, bucketName, key)
	require.Equal(t, urlStr, data.URL)
}
