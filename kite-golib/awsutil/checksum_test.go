package awsutil

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const expected = "900150983cd24fb0d6963f7d28e17f72"

func TestChecksumLocal(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = f.WriteString("abc")
	require.NoError(t, err)
	f.Close()

	h, err := checksumLocal(f.Name())
	require.NoError(t, err)
	assert.Equal(t, expected, string(h))
}

func TestChecksumS3(t *testing.T) {
	if !awsTests {
		t.Skip(`Use "go test -aws" to run tests that rely on AWS connectivity`)
	}

	h, err := ChecksumS3("s3://kite-data/experiments/testdata/abc.txt")
	require.NoError(t, err)
	assert.Equal(t, expected, string(h))
}
