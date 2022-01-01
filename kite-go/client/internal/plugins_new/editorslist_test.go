package plugins

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetected(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-editors")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	list := newEditorsList(filepath.Join(tempDir, "kite-editors.json"))

	added := list.addDetected("pycharm", "/Applications/PyCharm 2019.2")
	assert.True(t, added, "new entries must be added")
	assert.EqualValues(t, []string{"/Applications/PyCharm 2019.2"}, list.detected("pycharm"))
	assert.Empty(t, list.manual("pycharm"))

	added = list.addDetected("pycharm", "/Applications/PyCharm 2019.2")
	assert.False(t, added, "existing entries must not be added")
	assert.Len(t, list.detected("pycharm"), 1, "duplicate values must not be stored")

	list.removeDetected("pycharm", "/Applications/PyCharm 2019.2")
	assert.Empty(t, list.detected("pycharm"))
	assert.Empty(t, list.manual("pycharm"))

	list.addDetected("pycharm", "/Applications/PyCharm 2019.2")
	assert.Len(t, list.detected("pycharm"), 1, "duplicate values must not be stored")
	list.purgeDetected()
	assert.Empty(t, list.detected("pycharm"))
}

func TestManual(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-editors")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	list := newEditorsList(filepath.Join(tempDir, "kite-editors.json"))

	added := list.addManual("pycharm", "/Applications/PyCharm 2019.2")
	assert.True(t, added, "new entries must be added")
	assert.EqualValues(t, []string{"/Applications/PyCharm 2019.2"}, list.manual("pycharm"))
	assert.Empty(t, list.detected("pycharm"))

	added = list.addManual("pycharm", "/Applications/PyCharm 2019.2")
	assert.False(t, added, "existing entries must not be added")
	assert.Len(t, list.manual("pycharm"), 1, "duplicate values must not be stored")

	list.removeManual("pycharm", "/Applications/PyCharm 2019.2")
	assert.Empty(t, list.manual("pycharm"))
	assert.Empty(t, list.detected("pycharm"))

	list.addManual("pycharm", "/Applications/PyCharm 2019.2")
	assert.Len(t, list.manual("pycharm"), 1, "duplicate values must not be stored")
	list.purgeManual()
	assert.Empty(t, list.manual("pycharm"))
}
