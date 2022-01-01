package spyder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseCondaOutput(t *testing.T) {
	data := `[
  {
    "base_url": "https://conda.anaconda.org/spyder-ide",
    "build_number": 0,
    "build_string": "py37_0",
    "channel": "spyder-ide",
    "dist_name": "spyder-4.0.0rc1-py37_0",
    "name": "spyder",
    "platform": "linux-64",
    "version": "4.0.0rc1"
  },
  {
    "base_url": "https://repo.anaconda.com/pkgs/main",
    "build_number": 0,
    "build_string": "py37_0",
    "channel": "pkgs/main",
    "dist_name": "spyder-kernels-0.5.2-py37_0",
    "name": "spyder-kernels",
    "platform": "linux-64",
    "version": "0.5.2"
  }
]`

	list, err := parseCondaPackageList([]byte(data))
	require.NoError(t, err)

	assert.Len(t, list, 2)
	assert.EqualValues(t, "spyder", list[0].Name)
	assert.EqualValues(t, 4, list[0].majorVersion())
}
