package process

import (
	"context"
	"os"
	"testing"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/require"
)

func TestIsProcessRunning(t *testing.T) {
	// the current process, i.e. the test binary, must be running
	cur, err := process.NewProcess(int32(os.Getpid()))
	require.NoError(t, err)

	name, err := cur.Name()
	require.NoError(t, err)

	mgr := NewManager()
	running, err := mgr.IsProcessRunning(context.Background(), name)
	require.NoError(t, err)
	require.True(t, running)

	running, err = mgr.IsProcessRunning(context.Background(), "not-existing-proceses")
	require.NoError(t, err)
	require.False(t, running)
}
