package filesystem

import "testing"

func Test_IsFilteredDir_Windows(t *testing.T) {
	testIsFilteredDir(t, "windows", "c:\\Users\\", false)
	testIsFilteredDir(t, "windows", "c:\\AppData\\", true)
}
