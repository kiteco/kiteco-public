package platform

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// [kited] 2018/01/12 18:12:07.867786 log_test.go:23: Log entry 1
var logLinematcher = regexp.MustCompile(`\[kited] \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d{6} log_test.go:\d+: Log entry \d+`)

func Test_Logger(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "kite_logger.txt")
	defer os.Remove(tmpfile)
	logWriter, err := logWriter(tmpfile, false)
	assert.NoError(t, err)

	logger := newLogger(logWriter)
	logger.Println("Log entry 1")
	logger.Println("Log entry 2")

	logWriter.(*os.File).Close()

	file, err := os.Open(tmpfile)
	assert.NoError(t, err)

	//scan the log entries
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var lines = 0
	for scanner.Scan() {
		line := scanner.Text()
		assert.True(t, logLinematcher.MatchString(line), "The log isn't matching the expected pattern: %s", line)
		lines++
	}

	assert.EqualValues(t, 2, lines, "Logfile %s doesn't contain the expected lines", tmpfile)
}

func Test_Rotation(t *testing.T) {
	tmpfile, err := setupLogfile()
	require.NoError(t, err)

	tmpdir := filepath.Dir(tmpfile)
	defer os.RemoveAll(tmpdir)

	rotateLogs(tmpfile, maxLogFiles)
	assert.False(t, fileExists(tmpfile), "Logfile must be renamed during rotation")
	assert.EqualValues(t, 1, countFiles(tmpdir, "client.log"))

	rotateLogs(tmpfile, 1)
	assert.False(t, fileExists(tmpfile))
	assert.EqualValues(t, 1, countFiles(tmpdir, "client.log"))
}

func Test_RotationCurrentLog(t *testing.T) {
	tmpfile, err := setupLogfile()
	require.NoError(t, err)

	tmpdir := filepath.Dir(tmpfile)
	defer os.RemoveAll(tmpdir)

	//this needs a clean setup and not rotated logfiles with a timestamp
	rotateLogs(tmpfile, 0)
	assert.False(t, fileExists(tmpfile))
	assert.EqualValues(t, 1, countFiles(tmpdir, "client.log"))
}

//returns the path to the logfile
func setupLogfile() (string, error) {
	tmpdir, err := ioutil.TempDir("", "kite-log")
	if err != nil {
		return "", err
	}

	tmpfile := filepath.Join(tmpdir, "client.log")
	file, err := os.Create(tmpfile)
	if err != nil {
		return "", err
	}

	file.Close()

	return tmpfile, nil
}

func fileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func countFiles(dirpath string, prefix string) int {
	files, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			count++
		}
	}
	return count
}
