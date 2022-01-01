// +build !windows

package installid

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_InstallID(t *testing.T) {
	var err error
	kiteRoot, err = ioutil.TempDir("", "kite-root")
	require.NoError(t, err)
	defer os.RemoveAll(kiteRoot)

	id, set := IDIfSet()
	require.Empty(t, id)
	require.False(t, set)

	id, err = LoadInstallID(kiteRoot)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	id2, set := IDIfSet()
	require.True(t, set)
	assert.Equal(t, id, id2)
}
