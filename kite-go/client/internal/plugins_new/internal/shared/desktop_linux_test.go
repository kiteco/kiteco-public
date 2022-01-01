package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ReadDesktopFile(t *testing.T) {
	f, err := ReadDesktopFile("./test/jetbrains-pycharm.desktop")
	require.NoError(t, err)

	assert.EqualValues(t, "\"/opt/JetBrains/apps/PyCharm-P/ch-0/191.6605.12/bin/pycharm.sh\" %f", f.data["Exec"])
	v, err := f.Value("Exec")
	require.NoError(t, err)
	assert.EqualValues(t, "\"/opt/JetBrains/apps/PyCharm-P/ch-0/191.6605.12/bin/pycharm.sh\" %f", v)

	exec, err := f.ExecPath()
	require.NoError(t, err)
	assert.EqualValues(t, "/opt/JetBrains/apps/PyCharm-P/ch-0/191.6605.12/bin/pycharm.sh", exec)
}

func Test_ReadDesktopFileTryExec(t *testing.T) {
	f, err := ReadDesktopFile("./test/jetbrains-pycharm-tryexec.desktop")
	require.NoError(t, err)

	exec, err := f.ExecPath()
	require.NoError(t, err)
	assert.EqualValues(t, "/opt/JetBrains/apps/PyCharm-P/ch-0/191.6605.12/bin/pycharm.sh", exec)
}
