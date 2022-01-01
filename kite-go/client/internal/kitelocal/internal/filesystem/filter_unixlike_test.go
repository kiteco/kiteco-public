// +build !windows

package filesystem

import (
	"testing"
)

func Test_IsFilteredDir(t *testing.T) {
	// test contains filters
	testIsFilteredDir(t, "darwin", "/Applications/Kite/", true)
	testIsFilteredDir(t, "darwin", "/App/Kite/", false)

	testIsFilteredDir(t, "linux", "/home/user/.local/share/Trash/", true)
	testIsFilteredDir(t, "linux", "/home/user/local/share/Trash/", false)

	testIsFilteredDir(t, "linux", "/home/user/.config/autostart", true)
	testIsFilteredDir(t, "linux", "/home/user/config/autostart", false)

	testIsFilteredDir(t, "linux", "/home/user/.PyCharm2018.2/system/python_stubs/1365348222/", true)
	testIsFilteredDir(t, "linux", "/home/user/PyCharm2018.2/system/python_stubs/1365348222/", false)

	testIsFilteredDir(t, "linux", "/home/user/.IntelliJIDEA2018.2/system/python_stubs/1365348222/", true)
	testIsFilteredDir(t, "linux", "/home/user/IntelliJIDEA2018.2/system/python_stubs/1365348222/", false)

	// test starts with filters
	testIsFilteredDir(t, "darwin", "/dev/test/", true)
	testIsFilteredDir(t, "darwin", "/test/dev/", false)
	testIsFilteredDir(t, "darwin", "/var/", false)
}
