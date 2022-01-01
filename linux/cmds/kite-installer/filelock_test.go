package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_LockFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "kite-lock")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	lockPath := filepath.Join(tmpDir, "lockfile.txt")
	lock1 := newFileLock(lockPath)
	err = lock1.Lock()
	require.NoError(t, err)
	require.FileExists(t, lockPath)

	lock2 := newFileLock(lockPath)
	err = lock2.Lock()
	require.Error(t, err)
	require.FileExists(t, lockPath)
	err = lock2.Lock()
	require.Error(t, err)
	require.FileExists(t, lockPath)

	lock1.Unlock()
	_, err = os.Stat(lockPath)
	require.True(t, os.IsNotExist(err))
}
