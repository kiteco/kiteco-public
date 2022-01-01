package test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/stretchr/testify/require"
)

func Test_PanicInit(t *testing.T) {
	failingComponent := &panicComponent{initPanic: true}
	p, err := clientapp.StartEmptyTestEnvironment(failingComponent)
	require.NoError(t, err, "expected no error even when component panics during init")
	defer p.Close()
}

func Test_PanicRegisterHandlers(t *testing.T) {
	failingComponent := &panicComponent{handlersPanic: true}
	p, err := clientapp.StartEmptyTestEnvironment(failingComponent)
	require.NoError(t, err)
	defer p.Close()
}

func Test_PortInUse(t *testing.T) {
	p, err := clientapp.NewTestEnvironment()
	require.NoError(t, err)
	defer p.Close()

	err = p.StartPortNoDists(0) // random port
	require.NoError(t, err)

	p2, err := clientapp.NewTestEnvironment()
	require.NoError(t, err)
	defer p2.Close()

	port, err := strconv.Atoi(p.Kited.URL.Port())
	require.NoError(t, err)
	err = p2.StartPortNoDists(port)
	require.Error(t, err, "expected error when port was already in use")
	require.True(t, strings.Index(err.Error(), clientapp.ErrPortInUse.Error()) >= 0, "expected ErrPortInUse (wrapped by test setup)")
}
