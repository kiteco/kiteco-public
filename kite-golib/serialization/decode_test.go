package serialization

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apple struct {
	Variety string
	Redness int
}

func gzipString(x string) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte(x))
	w.Close()
	return b.Bytes()
}

func zipString(x string) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write([]byte(x))
	w.Close()
	return b.Bytes()
}

func base64zip(x string) string {
	return base64.URLEncoding.EncodeToString(zipString(x))
}

func TestJSON(t *testing.T) {
	var apples []*apple
	d := []byte(`{"Variety": "x", "Redness": 2}{"Variety": "y", "Redness": 3}`)
	err := decodeAs(bytes.NewBuffer(d), "foo.json", func(a *apple) {
		apples = append(apples, a)
	})
	require.NoError(t, err)
	assert.Len(t, apples, 2)
}

func TestGzippedJSON(t *testing.T) {
	var apples []*apple
	d := gzipString(`{"Variety": "x", "Redness": 2}{"Variety": "y", "Redness": 3}`)
	err := decodeAs(bytes.NewBuffer(d), "s3://kite-data/bar.json.gz", func(a *apple) {
		apples = append(apples, a)
	})
	require.NoError(t, err)
	assert.Len(t, apples, 2)
}

func TestEMR(t *testing.T) {
	var apples []*apple
	d := fmt.Sprintf("key1\t%s\nkey2\t%s\n", base64zip(`{"Redness":1}`), base64zip(`{"Redness":2}`))
	err := decodeAs(bytes.NewBufferString(d), "/x/y.emr", func(a *apple) {
		apples = append(apples, a)
	})
	require.NoError(t, err)
	assert.Len(t, apples, 2)
}

func TestEMRKeyValue(t *testing.T) {
	var apples []*apple
	d := fmt.Sprintf("key1\t%s\nkey2\t%s\n", base64zip(`{"Redness":1}`), base64zip(`{"Redness":2}`))
	err := decodeAs(bytes.NewBufferString(d), "s3://kite-data/foo.emr", func(x *awsutil.KeyValue) {
		var apple apple
		x.JSONValue(&apple)
		apples = append(apples, &apple)
	})
	require.NoError(t, err)
	assert.Len(t, apples, 2)
}

func TestDecodeOneJSON(t *testing.T) {
	var apple apple
	d := []byte(`{"Variety": "x", "Redness": 2}`)
	err := decodeAs(bytes.NewBuffer(d), "foo.json", &apple)
	require.NoError(t, err)
	assert.EqualValues(t, "x", apple.Variety)
	assert.EqualValues(t, 2, apple.Redness)
}
