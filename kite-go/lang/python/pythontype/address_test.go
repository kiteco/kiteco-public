package pythontype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddressExternalRoot(t *testing.T) {
	require.False(t, Address{IsExternalRoot: true}.Nil())
	require.True(t, Address{}.Nil())

	require.False(t, Address{}.Equals(Address{IsExternalRoot: true}))
	require.False(t, Address{IsExternalRoot: true}.Equals(Address{}))
}
