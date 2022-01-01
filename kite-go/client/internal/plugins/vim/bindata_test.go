package vim

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindata(t *testing.T) {
	// This test checks that the plugin is present within the bundled bindata.
	buf, err := AssetDir("vim-plugin/autoload")
	require.NoError(t, err)
	assert.NotEmpty(t, buf)

	fileBuf, err := Asset("vim-plugin/plugin/kite.vim")
	require.NoError(t, err)
	assert.NotEmpty(t, fileBuf)
}
