package permissions

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SanitizeRelativePath(t *testing.T) {
	path, err := sanitizePath("test.py")
	assert.Error(t, err)
	assert.Equal(t, "test.py", path)

	path, err = sanitizePath("../test.py")
	assert.Error(t, err)
	assert.Equal(t, "../test.py", path)

	path, err = sanitizePath("./test.py")
	assert.Error(t, err)
	assert.Equal(t, "./test.py", path)
}

func Test_SanitizeUnicode(t *testing.T) {
	// These two strings use two different unicode characters for the accented "e".
	str1 := "/Users/user/Jos\u00e9"  // José, i.e. Jos + LATIN SMALL LETTER E WITH ACUTE
	str2 := "/Users/user/Jose\u0301" // José, i.e. Jose + COMBINING ACUTE ACCENT

	if runtime.GOOS == "windows" {
		str1 = "C:\\Users\\user\\Jos\u00e9"  // José, i.e. Jos + LATIN SMALL LETTER E WITH ACUTE
		str2 = "C:\\Users\\user\\Jose\u0301" // José, i.e. Jose + COMBINING ACUTE ACCENT
	}

	assert.NotEqual(t, str1, str2)

	sanitized1, err := sanitizePath(str1)
	assert.NoError(t, err)

	sanitized2, err := sanitizePath(str2)
	assert.NoError(t, err)

	assert.Equal(t, sanitized1, sanitized2)
}
