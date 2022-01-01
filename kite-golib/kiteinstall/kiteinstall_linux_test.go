package kiteinstall

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ServiceStatus(t *testing.T) {
	require.True(t, isActiveOutput([]byte("ActiveState=active")))

	require.False(t, isActiveOutput([]byte("ActiveState=inactive")))
	require.False(t, isActiveOutput([]byte("error")))
}
